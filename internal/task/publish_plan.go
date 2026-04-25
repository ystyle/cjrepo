package task

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
	"ystyle.top/go/cjrepo/internal/upstream"
)

// Event 发布事件（用于 SSE 推送）
type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// TaskManager 发布计划执行引擎
type TaskManager struct {
	engine       *xorm.Engine
	upstreamSync *upstream.Sync
	mu           sync.RWMutex
	cancels      map[int64]context.CancelFunc
	subs         map[int64][]chan Event
}

func NewTaskManager(engine *xorm.Engine, upstreamSync *upstream.Sync) *TaskManager {
	return &TaskManager{
		engine:       engine,
		upstreamSync: upstreamSync,
		cancels:      make(map[int64]context.CancelFunc),
		subs:         make(map[int64][]chan Event),
	}
}

// Subscribe 订阅计划事件
func (tm *TaskManager) Subscribe(planID int64) chan Event {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	ch := make(chan Event, 64)
	tm.subs[planID] = append(tm.subs[planID], ch)
	return ch
}

// Unsubscribe 取消订阅
func (tm *TaskManager) Unsubscribe(planID int64, ch chan Event) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	subs := tm.subs[planID]
	for i, c := range subs {
		if c == ch {
			tm.subs[planID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

func (tm *TaskManager) emit(planID int64, evt Event) {
	tm.mu.RLock()
	subs := tm.subs[planID]
	tm.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- evt:
		default:
		}
	}
}

// Start 开始执行计划
func (tm *TaskManager) Start(planID int64) error {
	tm.mu.Lock()
	if _, ok := tm.cancels[planID]; ok {
		tm.mu.Unlock()
		return fmt.Errorf("plan %d is already running", planID)
	}
	ctx, cancel := context.WithCancel(context.Background())
	tm.cancels[planID] = cancel
	tm.mu.Unlock()

	go tm.run(ctx, planID)
	return nil
}

// Pause 暂停计划
func (tm *TaskManager) Pause(planID int64) error {
	tm.mu.Lock()
	cancel, ok := tm.cancels[planID]
	if !ok {
		tm.mu.Unlock()
		return fmt.Errorf("plan %d is not running", planID)
	}
	cancel()
	delete(tm.cancels, planID)
	tm.mu.Unlock()

	tm.emit(planID, Event{Type: "status", Payload: "paused"})
	return nil
}

// Resume 恢复计划
func (tm *TaskManager) Resume(planID int64) error {
	return tm.Start(planID)
}

func (tm *TaskManager) run(ctx context.Context, planID int64) {
	tm.emit(planID, Event{Type: "status", Payload: "running"})

	// 读取计划的轮询配置
	var plan models.PublishPlan
	tm.engine.ID(planID).Get(&plan)
	pollInterval := time.Duration(plan.PollInterval) * time.Second
	taskTimeout := time.Duration(plan.PollTimeout) * time.Second
	if pollInterval < 5*time.Second {
		pollInterval = 5 * time.Second
	}
	if taskTimeout < 60*time.Second {
		taskTimeout = 60 * time.Second
	}
	// 用 WithTimeout 替代手动 deadl ine 检查
	ctx, cancel := context.WithTimeout(ctx, taskTimeout)
	defer cancel()

	// 更新计划状态为 running
	tm.engine.ID(planID).Cols("status").Update(&models.PublishPlan{Status: "running"})

	var items []models.PublishPlanItem
	tm.engine.Where("plan_i_d = ? AND selected = ?", planID, true).OrderBy("\"order\"").Find(&items)

	// 更新每个处理中的项为 pending（重新开始）
	for i := range items {
		if items[i].Status == "publishing" || items[i].Status == "waiting_index" {
			tm.engine.ID(items[i].ID).Cols("status", "error").Update(&models.PublishPlanItem{Status: "pending"})
			items[i].Status = "pending"
		}
	}

	for i := range items {
		item := &items[i]
		if item.Status == "completed" {
			continue
		}

		select {
		case <-ctx.Done():
			tm.engine.ID(planID).Cols("status").Update(&models.PublishPlan{Status: "paused"})
			return
		default:
		}

		// 更新当前项为 publishing
		tm.engine.ID(item.ID).Cols("status", "started_at").Update(&models.PublishPlanItem{
			Status:    "publishing",
			StartedAt: time.Now(),
		})

		// 获取包信息
		var pkg models.Package
		if ok, _ := tm.engine.ID(item.PackageID).Get(&pkg); !ok {
			tm.failItem(planID, item, "package not found")
			continue
		}

		// 发布到上游
		if err := tm.publishToUpstream(ctx, pkg); err != nil {
			tm.failItem(planID, item, err.Error())
			continue
		}

		tm.engine.ID(item.ID).Cols("status").Update(&models.PublishPlanItem{Status: "waiting_index"})

		// 轮询索引
		if err := tm.pollIndex(ctx, pkg, pollInterval); err != nil {
			tm.failItem(planID, item, err.Error())
			continue
		}

		// 标记完成
		tm.engine.ID(item.ID).Cols("status", "completed_at").Update(&models.PublishPlanItem{
			Status:      "completed",
			CompletedAt: time.Now(),
		})
		tm.engine.ID(planID).Incr("completed_count").Update(&models.PublishPlan{})
	}

	// 检查是否全部完成
	var remain int64
	tm.engine.Where("plan_i_d = ? AND status != ? AND selected = ?", planID, "completed", true).Count(&remain)
	status := "completed"
	if remain > 0 {
		status = "failed"
	}
	tm.engine.ID(planID).Cols("status").Update(&models.PublishPlan{Status: status})
	tm.emit(planID, Event{Type: "status", Payload: status})
}

func (tm *TaskManager) failItem(planID int64, item *models.PublishPlanItem, errMsg string) {
	tm.engine.ID(item.ID).Cols("status", "error").Update(&models.PublishPlanItem{
		Status: "failed",
		Error:  errMsg,
	})
}

func (tm *TaskManager) publishToUpstream(ctx context.Context, pkg models.Package) error {
	upstream, err := tm.getUpstreamForPlan()
	if err != nil {
		return fmt.Errorf("get upstream: %w", err)
	}

	tarballPath := pkg.TarballPath
	if tarballPath == "" {
		return fmt.Errorf("package tarball path is empty")
	}
	tarballData, err := os.ReadFile(tarballPath)
	if err != nil {
		return fmt.Errorf("read tarball: %w", err)
	}

	// 构造二进制协议 body: [meta_version][meta_size][meta_json][tarball_version][tarball_size][tarball_data]
	metaJSON := pkg.MetaData
	if metaJSON == "" {
		return fmt.Errorf("package meta_data is empty")
	}
	metaLen := len(metaJSON)

	metaVersion := byte(1)
	tarballVersion := byte(1)

	buf := new(bytes.Buffer)
	buf.WriteByte(metaVersion)
	binary.Write(buf, binary.LittleEndian, int32(metaLen))
	buf.WriteString(metaJSON)
	buf.WriteByte(tarballVersion)
	binary.Write(buf, binary.LittleEndian, int32(len(tarballData)))
	buf.Write(tarballData)

	return tm.upstreamSync.PublishPackage(upstream, &pkg, buf.Bytes())
}

func (tm *TaskManager) pollIndex(ctx context.Context, pkg models.Package, pollInterval time.Duration) error {
	upstream, err := tm.getUpstreamForPlan()
	if err != nil {
		return fmt.Errorf("get upstream: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}

		// 查询上游索引，确认版本已同步
		data, err := tm.upstreamSync.FetchIndex(upstream, pkg.Name, pkg.Organization)
		if err != nil {
			log.Printf("[PublishPlan] poll index error: %v", err)
			continue
		}
		if data == nil {
			continue
		}

		// 解析索引，检查版本是否存在
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var entry struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			if entry.Name == pkg.Name && entry.Version == pkg.Version {
				found = true
				break
			}
		}
		if found {
			log.Printf("[PublishPlan] index confirmed: %s/%s@%s", pkg.Organization, pkg.Name, pkg.Version)
			return nil
		}
	}

	return fmt.Errorf("publish cancelled: %s/%s@%s", pkg.Organization, pkg.Name, pkg.Version)
}

func (tm *TaskManager) getUpstreamForPlan() (*models.Upstream, error) {
	// 简化：取第一个启用的上游
	return tm.upstreamSync.GetEnabledUpstream()
}

// Init 启动时扫描，暂停所有 running 的计划
func (tm *TaskManager) Init() {
	var plans []models.PublishPlan
	tm.engine.Where("status = ?", "running").Find(&plans)
	for _, p := range plans {
		tm.engine.ID(p.ID).Cols("status").Update(&models.PublishPlan{Status: "paused"})
		log.Printf("[PublishPlan] Plan %d set to paused (server restart)", p.ID)
	}
}
