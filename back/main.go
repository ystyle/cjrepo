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

	log.Printf("[FORM-DATA] Parsing started (size: %d bytes)", len(body))

	// 预处理边界标记
	boundaryStart := []byte("--" + boundary)
	delimiter := append([]byte("\r\n"), boundaryStart...)
	endMarker := append(boundaryStart, []byte("--")...)

	// 查找第一个有效部分
	cursor := findFirstPart(body, boundaryStart)
	if cursor < 0 {
		return nil, fmt.Errorf("no valid part found")
	}

	// 主解析循环
	for cursor < len(body) {
		// 解析头部
		headers, contentStart, err := parseHeaders(body, cursor)
		if err != nil {
			return nil, err
		}
		cursor = contentStart

		// 提取元数据
		name, filename := extractMetadata(headers)
		if filename != "" {
			log.Printf("[FORM-DATA] Found file: %s", filename)
		}

		// 提取内容
		content, nextPart := extractContent(body, cursor, delimiter, endMarker)
		cursor = nextPart

		// 存储数据
		if filename != "" {
			data.FileName = filepath.Base(filename)
			data.FileContent = content
		} else if name != "" {
			data.Fields[name] = string(content)
		}

		// 检查结束标记
		if bytes.HasPrefix(body[cursor:], endMarker) {
			break
		}
	}

	log.Printf("[FORM-DATA] Parse completed (fields: %d, file: %v)",
		len(data.Fields), data.FileName != "")
	return data, nil
}

// 辅助函数 1: 查找第一个有效部分
func findFirstPart(body, boundaryStart []byte) int {
	if bytes.HasPrefix(body, boundaryStart) {
		return len(boundaryStart)
	}
	if pos := bytes.Index(body, []byte("Content-Disposition")); pos != -1 {
		log.Printf("[FORM-DATA] Starting at Content-Disposition (pos: %d)", pos)
		return pos
	}
	return -1
}

// 辅助函数 2: 解析头部
func parseHeaders(body []byte, start int) ([]byte, int, error) {
	// 支持多种换行格式
	for _, sep := range [][]byte{[]byte("\r\n\r\n"), []byte("\n\n"), []byte("\r\n")} {
		if end := bytes.Index(body[start:], sep); end != -1 {
			headers := body[start : start+end]
			return headers, start + end + len(sep), nil
		}
	}
	return nil, 0, fmt.Errorf("invalid part headers")
}

// 辅助函数 3: 提取元数据
func extractMetadata(headers []byte) (name, filename string) {
	if namePos := bytes.Index(headers, []byte(`name="`)); namePos != -1 {
		if nameVal, ok := extractQuotedValue(headers[namePos+6:]); ok {
			name = nameVal
		}
	}
	if filePos := bytes.Index(headers, []byte(`filename="`)); filePos != -1 {
		if fileVal, ok := extractQuotedValue(headers[filePos+10:]); ok {
			filename = fileVal
		}
	}
	return
}

// 辅助函数 4: 提取引号内容
func extractQuotedValue(data []byte) (string, bool) {
	if end := bytes.IndexByte(data, '"'); end != -1 {
		return string(data[:end]), true
	}
	return "", false
}

// 辅助函数 5: 提取内容
func extractContent(body []byte, start int, delimiter, endMarker []byte) ([]byte, int) {
	// 查找下一个边界
	if end := bytes.Index(body[start:], delimiter); end != -1 {
		return bytes.TrimRight(body[start:start+end], "\r\n"), start + end
	}
	// 查找结束标记
	if bytes.HasSuffix(body[start:], endMarker) {
		end := len(body[start:]) - len(endMarker)
		return bytes.TrimRight(body[start:start+end], "\r\n"), len(body)
	}
	// 作为最后一个部分
	return bytes.TrimRight(body[start:], "\r\n"), len(body)
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
	fmt.Println(response)
	log.Printf("[UPLOAD] Response sent (length: %d bytes)", len(response))
}

func main() {
	log.Printf("[SERVER] Starting on :8060")
	log.Printf("[SERVER] Route: POST /depot/publish/")

	http.HandleFunc("/depot/publish/", uploadHandler)
	log.Fatal(http.ListenAndServe(":8060", nil))
}
