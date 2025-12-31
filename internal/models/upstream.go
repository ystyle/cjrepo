package models

import "time"

// Upstream 上游配置
type Upstream struct {
	ID         int64     `xorm:"pk autoincr 'id'" json:"id"`
	Name       string    `xorm:"VARCHAR(255) NOT NULL UNIQUE 'name'" json:"name"`           // 上游名称
	URL        string    `xorm:"VARCHAR(500) NOT NULL 'url'" json:"url"`                     // 上游地址
	Enabled    bool      `xorm:"DEFAULT true 'enabled'" json:"enabled"`                      // 是否启用
	CacheTTL   int       `xorm:"DEFAULT 86400 'cache_ttl'" json:"cache_ttl"`                 // 缓存时间（秒），默认24小时
	AuthToken  string    `xorm:"VARCHAR(500) 'auth_token'" json:"auth_token"`                // 可选：上游需要的认证
	LastSyncAt time.Time `xorm:"'last_sync_at'" json:"last_sync_at"`                         // 最后同步时间
	CreatedAt  time.Time `xorm:"created 'created_at'" json:"created_at"`
	UpdatedAt  time.Time `xorm:"updated 'updated_at'" json:"updated_at"`
}

// TableName 指定表名
func (Upstream) TableName() string {
	return "upstreams"
}
