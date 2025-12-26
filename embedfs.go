package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed frontend/dist
var webFS embed.FS

// WebFS 返回前端文件的 HTTP 文件系统
func WebFS() http.FileSystem {
	fsys, _ := fs.Sub(webFS, "frontend/dist")
	return http.FS(fsys)
}
