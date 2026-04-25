package task

import (
	"context"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

func TestEventSubscribe(t *testing.T) {
	tm := &TaskManager{
		cancels: make(map[int64]context.CancelFunc),
		subs:    make(map[int64][]chan Event),
	}

	// 订阅
	ch1 := tm.Subscribe(1)
	ch2 := tm.Subscribe(1)
	ch3 := tm.Subscribe(2)

	if len(tm.subs[1]) != 2 {
		t.Errorf("plan 1 should have 2 subscribers, got %d", len(tm.subs[1]))
	}
	if len(tm.subs[2]) != 1 {
		t.Errorf("plan 2 should have 1 subscriber, got %d", len(tm.subs[2]))
	}

	// 发送事件
	tm.emit(1, Event{Type: "status", Payload: "running"})

	select {
	case evt := <-ch1:
		if evt.Type != "status" {
			t.Errorf("expected status event, got %s", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event on ch1")
	}

	select {
	case evt := <-ch2:
		if evt.Type != "status" {
			t.Errorf("expected status event, got %s", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event on ch2")
	}

	// ch3 不应该收到事件
	select {
	case <-ch3:
		t.Error("plan 2 subscriber should not receive plan 1 events")
	case <-time.After(100 * time.Millisecond):
	}

	// 取消订阅
	tm.Unsubscribe(1, ch1)
	if len(tm.subs[1]) != 1 {
		t.Errorf("after unsubscribe, plan 1 should have 1 subscriber, got %d", len(tm.subs[1]))
	}

	tm.Unsubscribe(1, ch2)
	if len(tm.subs[1]) != 0 {
		t.Errorf("after all unsubscribe, plan 1 should have 0 subscribers, got %d", len(tm.subs[1]))
	}
}

func TestPauseResume(t *testing.T) {
	tm := &TaskManager{
		cancels: make(map[int64]context.CancelFunc),
		subs:    make(map[int64][]chan Event),
	}

	// 暂停未运行的计划应该报错
	if err := tm.Pause(1); err == nil {
		t.Error("expected error when pausing non-running plan")
	}

	// 订阅并验证事件
	ch := tm.Subscribe(1)
	defer tm.Unsubscribe(1, ch)

	// 直接操作 cancels map 模拟运行状态
	ctx, cancel := context.WithCancel(context.Background())
	tm.cancels[1] = cancel

	// 暂停
	if err := tm.Pause(1); err != nil {
		t.Errorf("pause failed: %v", err)
	}

	// 验证 cancelled
	select {
	case <-ctx.Done():
		// context was cancelled
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled")
	}

	// 验证 cancels 中已删除
	if _, ok := tm.cancels[1]; ok {
		t.Error("plan should be removed from cancels after pause")
	}

	// 验证 paused 事件
	select {
	case evt := <-ch:
		if evt.Type != "status" || evt.Payload != "paused" {
			t.Errorf("expected paused status event, got %s=%v", evt.Type, evt.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for paused event")
	}
}

func TestConcurrentEmit(t *testing.T) {
	tm := &TaskManager{
		cancels: make(map[int64]context.CancelFunc),
		subs:    make(map[int64][]chan Event),
	}

	// 多个订阅者并发接收事件
	const numSubs = 10
	var chans []chan Event
	for i := 0; i < numSubs; i++ {
		ch := tm.Subscribe(1)
		chans = append(chans, ch)
	}

	var wg sync.WaitGroup
	wg.Add(numSubs)

	for i, ch := range chans {
		go func(idx int, c chan Event) {
			defer wg.Done()
			select {
			case evt := <-c:
				if evt.Type != "ping" {
					t.Errorf("subscriber %d: expected ping, got %s", idx, evt.Type)
				}
			case <-time.After(time.Second):
				t.Errorf("subscriber %d: timeout", idx)
			}
		}(i, ch)
	}

	tm.emit(1, Event{Type: "ping"})
	wg.Wait()
}

func TestInitPausesRunningPlans(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	engine.Sync2(new(models.PublishPlan))

	// 创建运行中的计划
	engine.Insert(&models.PublishPlan{
		Name:   "test-plan",
		Status: "running",
	})

	// Init 应该暂停运行中的计划
	tm := &TaskManager{
		engine:  engine,
		cancels: make(map[int64]context.CancelFunc),
		subs:    make(map[int64][]chan Event),
	}
	tm.Init()

	var plan models.PublishPlan
	engine.ID(1).Get(&plan)
	if plan.Status != "paused" {
		t.Errorf("expected status=paused after Init, got %s", plan.Status)
	}
}

func TestStartInvalidPlan(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	engine.Sync2(new(models.PublishPlan), new(models.PublishPlanItem))

	tm := &TaskManager{
		engine:  engine,
		cancels: make(map[int64]context.CancelFunc),
		subs:    make(map[int64][]chan Event),
	}

	// 启动不存在的计划 — Start 不会验证 DB 存在性，不会报错
	// 但不会有实际效果，因为没有对应的 plan 和 items
	if err := tm.Start(999); err != nil {
		t.Errorf("expected no error for non-existent plan, got %v", err)
	}
	tm.Pause(999)
}

func TestDoubleStart(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	engine.Sync2(new(models.PublishPlan), new(models.PublishPlanItem))

	engine.Insert(&models.PublishPlan{
		Name:   "test",
		Status: "pending",
	})

	tm := &TaskManager{
		engine:  engine,
		cancels: make(map[int64]context.CancelFunc),
		subs:    make(map[int64][]chan Event),
	}

	// 第一次启动（可能失败或成功，取决于 run 是否执行完）
	err1 := tm.Start(1)
	_ = err1

	// 第二次启动同一计划应该报错
	err2 := tm.Start(1)
	if err2 == nil {
		t.Error("expected error when starting already-running plan")
	}
}
