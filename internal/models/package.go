package models

import "time"

// Package represents a published Cangjie package
type Package struct {
	ID            int64     `xorm:"pk autoincr"`
	Organization  string    `xorm:"index"`
	Name          string    `xorm:"index"`
	Version       string    `xorm:"index"`
	Description   string
	ArtifactType  string
	Executable    bool
	Authors       string    // JSON array string
	Repository    string
	Homepage      string
	Documentation string
	Tags          string    // JSON array string
	Categories    string    // JSON array string
	Licenses      string    // JSON array string
	MetaVersion   int
	MetaData      string    `xorm:"text"` // Complete meta-data.json

	// File storage info
	TarballPath   string    `xorm:"text"`
	TarballSize   int64
	TarballSHA256 string    `xorm:"index"`

	// Timestamps
	CreatedAt time.Time `xorm:"created"`
	UpdatedAt time.Time `xorm:"updated"`
}

// User represents a user with API token
type User struct {
	ID        int64     `xorm:"pk autoincr"`
	Username  string    `xorm:"unique"`
	Token     string    `xorm:"unique index 'token'"`
	Email     string
	IsActive  bool      `xorm:"default true"`
	CreatedAt time.Time `xorm:"created"`
}

// PublishLog records package publishing operations
type PublishLog struct {
	ID           int64     `xorm:"pk autoincr"`
	Organization string
	PackageName  string
	Version      string
	Status       string    // success/failed
	ErrorMessage string    `xorm:"text"`
	IPAddr       string
	UserAgent    string
	CreatedAt    time.Time `xorm:"created"`
}

// TableName returns the table name for Package
func (Package) TableName() string {
	return "packages"
}

// TableName returns the table name for User
func (User) TableName() string {
	return "users"
}

// TableName returns the table name for PublishLog
func (PublishLog) TableName() string {
	return "publish_logs"
}
