package models

import "time"

// AdminLog 管理员操作日志
type AdminLog struct {
	ID        int64     `xorm:"'i_d' pk autoincr" json:"id"`
	Action    string    `json:"action"`    // 操作类型: delete_package, create_user, reset_token, etc.
	Target    string    `json:"target"`    // 操作目标 ID
	Details   string    `xorm:"text" json:"details"`   // JSON 格式的详细信息
	IPAddr    string    `xorm:"'i_p_addr'" json:"ip_addr"`   // 操作者 IP
	UserAgent string    `json:"user_agent"` // 操作者 User-Agent
	CreatedAt time.Time `xorm:"created" json:"created_at"`
}

// TableName returns the table name for AdminLog
func (AdminLog) TableName() string {
	return "admin_log"
}
