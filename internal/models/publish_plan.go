package models

import "time"

// PublishPlan 发布计划
type PublishPlan struct {
	ID             int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	Name           string    `xorm:"VARCHAR(200) NOT NULL 'name'" json:"name"`
	TargetUpstream int64     `xorm:"NOT NULL 'target_upstream'" json:"target_upstream"`
	Status         string    `xorm:"VARCHAR(20) NOT NULL 'status'" json:"status"`
	TotalCount     int       `xorm:"NOT NULL DEFAULT 0 'total_count'" json:"total_count"`
	CompletedCount int       `xorm:"NOT NULL DEFAULT 0 'completed_count'" json:"completed_count"`
	PollInterval   int       `xorm:"NOT NULL DEFAULT 60 'poll_interval'" json:"poll_interval"`
	PollTimeout    int       `xorm:"NOT NULL DEFAULT 600 'poll_timeout'" json:"poll_timeout"`
	CreatedAt      time.Time `xorm:"created 'created_at'" json:"created_at"`
	UpdatedAt      time.Time `xorm:"updated 'updated_at'" json:"updated_at"`
}

// PublishPlanItem 计划项
type PublishPlanItem struct {
	ID          int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	PlanID      int64     `xorm:"NOT NULL INDEX 'plan_i_d'" json:"plan_id"`
	PackageID   int64     `xorm:"NOT NULL INDEX 'package_i_d'" json:"package_id"`
	Order       int       `xorm:"NOT NULL 'order'" json:"order"`
	Category    string    `xorm:"VARCHAR(20) NOT NULL 'category'" json:"category"`
	Status      string    `xorm:"VARCHAR(20) NOT NULL 'status'" json:"status"`
	Selected    bool      `xorm:"DEFAULT false 'selected'" json:"selected"`
	Error       string    `xorm:"TEXT 'error'" json:"error"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}
