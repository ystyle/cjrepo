package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/docs
var docsFS embed.FS

// DocsFS 返回文档站点的 HTTP 文件系统
func DocsFS() http.FileSystem {
	fsys, _ := fs.Sub(docsFS, "dist/docs")
	return http.FS(fsys)
}
