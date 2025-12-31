package models

import "time"

// Package represents a published Cangjie package
type Package struct {
	ID            int64     `xorm:"pk autoincr" json:"id"`
	Organization  string    `xorm:"index" json:"organization"`
	Name          string    `xorm:"index" json:"name"`
	Version       string    `xorm:"index" json:"version"`
	CjcVersion    string    `json:"cjc_version"`
	Description   string    `json:"description"`
	ArtifactType  string    `json:"artifact_type"`
	Executable    bool      `json:"executable"`
	Authors       string    `json:"authors"` // JSON array string
	Repository    string    `json:"repository"`
	Homepage      string    `json:"homepage"`
	Documentation string    `json:"documentation"`
	Tags          string    `json:"tags"` // JSON array string
	Categories    string    `json:"categories"` // JSON array string
	Licenses      string    `json:"licenses"` // JSON array string
	MetaVersion   int       `json:"meta_version"`
	MetaData      string    `xorm:"text" json:"meta_data"` // Complete meta-data.json
	Readme        string    `xorm:"text" json:"readme"` // README content

	// File storage info
	TarballPath   string    `xorm:"text" json:"tarball_path"`
	TarballSize   int64     `json:"tarball_size"`
	TarballSHA256 string    `xorm:"index" json:"tarball_sha256"`

	// Statistics
	DownloadCount int64     `xorm:"default 0" json:"download_count"`

	// Timestamps
	CreatedAt time.Time `xorm:"created" json:"created_at"`
	UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
	DeletedAt time.Time `xorm:"deleted" json:"deleted_at"` // Soft delete timestamp
}

// User represents a user with API token
type User struct {
	ID        int64     `xorm:"pk autoincr" json:"id"`
	Username  string    `xorm:"unique" json:"username"`
	Token     string    `xorm:"unique index 'token'" json:"token"`
	Email     string    `json:"email"`
	IsActive  bool      `xorm:"default true" json:"is_active"`
	CreatedAt time.Time `xorm:"created" json:"created_at"`
}

// PublishLog records package publishing operations
type PublishLog struct {
	ID           int64     `xorm:"'i_d' pk autoincr" json:"id"`
	Organization string    `json:"organization"`
	PackageName  string    `xorm:"package_name" json:"package_name"`
	Version      string    `json:"version"`
	Status       string    `json:"status"` // success/failed
	ErrorMessage string    `xorm:"'error_message' text" json:"error"`
	IPAddr       string    `xorm:"'i_p_addr'" json:"ip_addr"`
	UserAgent    string    `xorm:"user_agent" json:"user_agent"`
	CreatedAt    time.Time `xorm:"created" json:"created_at"`
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
