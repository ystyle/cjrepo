package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FormData struct {
	FileContent []byte
	FileName    string
	Source      string
	Fields      map[string]string
}

func parseFormData(body []byte, boundary string) (*FormData, error) {
	data := &FormData{
		Source: "default",
		Fields: make(map[string]string),
	}

	log.Printf("[FORM-DATA] Start parsing (body length: %d bytes)", len(body))

	boundaryStart := []byte("--" + boundary)
	delimiter := append([]byte("\r\n"), boundaryStart...)
	fullBoundaryEnd := append(boundaryStart, []byte("--")...)

	cursor := 0
	isFirstPart := true

	for cursor < len(body) {
		// 1. 处理part起始位置
		if isFirstPart {
			isFirstPart = false
			if !bytes.HasPrefix(body, boundaryStart) {
				if pos := bytes.Index(body, []byte("Content-Disposition")); pos != -1 {
					cursor = pos
				} else {
					return nil, fmt.Errorf("no valid part found")
				}
			} else {
				cursor = len(boundaryStart)
			}
		} else {
			delimPos := bytes.Index(body[cursor:], delimiter)
			if delimPos == -1 {
				if bytes.HasSuffix(body[cursor:], fullBoundaryEnd) {
					break
				}
				if data.Source != "default" || len(data.Fields) > 0 || data.FileName != "" {
					break // 已有有效数据时允许优雅退出
				}
				return nil, fmt.Errorf("missing boundary delimiter")
			}
			cursor += delimPos + len(delimiter)
		}

		// 2. 解析头部（完全避免打印二进制内容）
		headerEnd := bytes.Index(body[cursor:], []byte("\r\n\r\n"))
		if headerEnd == -1 {
			headerEnd = bytes.Index(body[cursor:], []byte("\n\n"))
			if headerEnd == -1 {
				return nil, fmt.Errorf("invalid part headers")
			}
		}

		headers := body[cursor : cursor+headerEnd]
		cursor += headerEnd + 2

		// 3. 处理source字段（仅记录字段存在）
		if bytes.Contains(headers, []byte(`name="source"`)) {
			end := bytes.Index(body[cursor:], delimiter)
			if end == -1 {
				end = len(body[cursor:])
			}
			data.Source = string(bytes.TrimSpace(body[cursor : cursor+end]))
			log.Printf("[FORM-DATA] Found source field")
			cursor += end
			continue
		}

		// 4. 提取字段信息（不打印任何字段值）
		var name, filename string
		if namePos := bytes.Index(headers, []byte(`name="`)); namePos != -1 {
			nameStart := namePos + 6
			nameEnd := bytes.IndexByte(headers[nameStart:], '"')
			if nameEnd != -1 {
				name = string(headers[nameStart : nameStart+nameEnd])
			}
		}
		if filePos := bytes.Index(headers, []byte(`filename="`)); filePos != -1 {
			fileStart := filePos + 10
			fileEnd := bytes.IndexByte(headers[fileStart:], '"')
			if fileEnd != -1 {
				filename = string(headers[fileStart : fileStart+fileEnd])
				log.Printf("[FORM-DATA] Found file attachment: %s", filename)
			}
		}

		// 5. 提取数据内容（完全不记录内容）
		nextDelim := bytes.Index(body[cursor:], delimiter)
		if nextDelim == -1 {
			if bytes.HasSuffix(body[cursor:], fullBoundaryEnd) {
				nextDelim = len(body[cursor:]) - len(fullBoundaryEnd)
			} else {
				nextDelim = len(body[cursor:])
			}
		}

		content := bytes.TrimRight(body[cursor:cursor+nextDelim], "\r\n")
		cursor += nextDelim

		// 6. 存储数据
		if filename != "" {
			data.FileName = filename
			data.FileContent = content
		} else if name != "" {
			data.Fields[name] = string(content)
		}
	}

	log.Printf("[FORM-DATA] Parse completed (fields: %d, hasFile: %v)",
		len(data.Fields), data.FileName != "")
	return data, nil
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[UPLOAD] New request from %s %s", r.RemoteAddr, r.Method)
	log.Printf("[UPLOAD] Headers: %v", r.Header)

	contentType := r.Header.Get("Content-Type")
	boundary := ""
	if strings.Contains(contentType, "boundary=") {
		boundary = strings.Trim(strings.Split(contentType, "boundary=")[1], `"`)
	}
	log.Printf("[UPLOAD] Content-Type: %s", contentType)
	log.Printf("[UPLOAD] Boundary: %s", boundary)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[UPLOAD] ERROR reading body: %v", err)
		http.Error(w, "read failed", http.StatusBadRequest)
		return
	}
	log.Printf("[UPLOAD] Body size: %d bytes", len(body))

	formData, err := parseFormData(body, boundary)
	if err != nil {
		log.Printf("[UPLOAD] ERROR parsing form data: %v", err)
		log.Printf("[UPLOAD] Body dump (first 200 bytes): %x", body[:200])
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[UPLOAD] Parsed form data - Fields: %d, Has file: %v", len(formData.Fields), formData.FileName != "")
	if formData.FileName != "" {
		log.Printf("[UPLOAD] Processing file: %s (%d bytes)", formData.FileName, len(formData.FileContent))
	}

	if formData.FileName != "" {
		safeFileName := filepath.Base(formData.FileName)
		log.Printf("[UPLOAD] Saving file: %s (%d bytes)", safeFileName, len(formData.FileContent))
		if err := os.WriteFile(safeFileName, formData.FileContent, 0644); err != nil {
			log.Printf("[UPLOAD] ERROR saving file: %v (path: %s)", err, safeFileName)
			http.Error(w, "save failed: invalid path or permissions", http.StatusInternalServerError)
			return
		}
		log.Printf("[UPLOAD] File saved successfully: %s (%d bytes)", safeFileName, len(formData.FileContent))
	}

	response := fmt.Sprintf(
		"File: %s\nSource: %s\nFields: %v",
		formData.FileName,
		formData.Source,
		formData.Fields,
	)
	w.Write([]byte(response))
	log.Printf("[UPLOAD] Response sent (length: %d bytes)", len(response))
}

func main() {
	log.Printf("[SERVER] Starting on :8060")
	log.Printf("[SERVER] Route: POST /depot/publish/")

	http.HandleFunc("/depot/publish/", uploadHandler)
	log.Fatal(http.ListenAndServe(":8060", nil))
}
