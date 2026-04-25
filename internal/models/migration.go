package models

import "time"

type Migration struct {
	ID        int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	Version   string    `xorm:"VARCHAR(50) NOT NULL UNIQUE 'version'" json:"version"`
	Name      string    `xorm:"VARCHAR(255) NOT NULL 'name'" json:"name"`
	ExecutedAt time.Time `xorm:"created 'executed_at'" json:"executed_at"`
}

func (Migration) TableName() string {
	return "migrations"
}