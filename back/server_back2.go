package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// 保持原始日志结构
func logRequest(r *http.Request) {
	fmt.Println("\n=== 收到新请求 ===")
	fmt.Printf("方法: %s\n内容类型: %s\n内容长度: %s\n",
		r.Method,
		r.Header.Get("Content-Type"),
		r.Header.Get("Content-Length"),
	)
}

// 保持原始FormData结构
type FormData struct {
	FileContent []byte
	FileName    string
	Fields      map[string]string
}

// 还原原始解析逻辑
func parseCustomFormData(r *http.Request) (*FormData, error) {
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败: %w", err)
	}
	defer r.Body.Close()

	result := &FormData{
		Fields: make(map[string]string),
	}

	lines := bytes.Split(bodyData, []byte("\n"))
	var (
		currentField string
		fieldContent bytes.Buffer
		inField      bool
		boundary     string
	)

	for _, line := range lines {
		lineStr := string(line)
		lineStr = strings.TrimRight(lineStr, "\r")

		switch {
		case strings.HasPrefix(lineStr, "Content-Disposition: form-data;"):
			if currentField != "" && inField {
				saveField(currentField, result, fieldContent.Bytes())
				fieldContent.Reset()
			}

			fieldName := extractFieldName(lineStr)
			if fieldName == "" {
				return nil, fmt.Errorf("无效字段名: %s", lineStr)
			}
			currentField = fieldName

			if fieldName == "source" {
				result.FileName = extractFileName(lineStr)
				if result.FileName != "" {
					fmt.Printf("检测到文件上传: %s\n", result.FileName)
				}
			}
			inField = false

		case strings.HasPrefix(lineStr, "--") && boundary == "":
			boundary = lineStr

		case boundary != "" && strings.HasPrefix(lineStr, boundary):
			if currentField != "" && inField {
				saveField(currentField, result, fieldContent.Bytes())
				fieldContent.Reset()
			}
			inField = false

		case lineStr == "" && currentField != "":
			inField = true

		case inField:
			if fieldContent.Len() > 0 {
				fieldContent.WriteByte('\n')
			}
			fieldContent.Write(line)
		}
	}

	if currentField != "" && inField {
		saveField(currentField, result, fieldContent.Bytes())
	}

	return result, nil
}

// 还原原始字段保存逻辑
func saveField(field string, form *FormData, content []byte) {
	content = bytes.TrimRight(content, "\r\n")

	if field == "source" && form.FileName != "" {
		form.FileContent = content
		if err := saveFile(form.FileName, content); err != nil {
			fmt.Printf("文件保存错误: %v\n", err)
		}
	} else {
		form.Fields[field] = string(content)
	}
}

// 还原原始辅助函数
func extractFieldName(line string) string {
	start := strings.Index(line, `name="`) + len(`name="`)
	if start <= len(`name="`) {
		return ""
	}
	end := strings.Index(line[start:], `"`)
	if end == -1 {
		return ""
	}
	return line[start : start+end]
}

func extractFileName(line string) string {
	start := strings.Index(line, `filename="`)
	if start == -1 {
		return ""
	}
	start += len(`filename="`)
	end := strings.Index(line[start:], `"`)
	if end == -1 {
		return ""
	}
	return filepath.Base(line[start : start+end])
}

// 还原原始文件保存逻辑
func saveFile(filename string, content []byte) error {
	content = bytes.TrimRight(content, "\r")
	if err := os.WriteFile(filename, content, 0644); err != nil {
		return err
	}
	info, _ := os.Stat(filename)
	fmt.Printf("文件保存成功: %s (%d 字节)\n", filename, info.Size())

	// ZIP文件头验证
	if len(content) > 3 && content[0] == 0x50 && content[1] == 0x4B {
		fmt.Printf("验证: 有效的ZIP文件头(PK开头)\n")
	}
	return nil
}

// 保持原始处理函数
func handleDepotPublish(w http.ResponseWriter, r *http.Request) {
	logRequest(r)

	formData, err := parseCustomFormData(r)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("解析成功! 字段数: %d\n", len(formData.Fields))
	for k, v := range formData.Fields {
		fmt.Printf("字段[%s] => %s\n", k, v)
	}

	if formData.FileName != "" {
		fmt.Printf("文件大小验证: %d 字节\n", len(formData.FileContent))
	}

	fmt.Fprintf(w, "文件上传处理完成!")
}

func main() {
	http.HandleFunc("/depot/publish/", handleDepotPublish)
	fmt.Println("文件上传服务已启动 (:8060)")
	fmt.Println("测试命令: curl -F 'source=@test.zip' http://localhost:8060/depot/publish/")
	http.ListenAndServe(":8060", nil)
}
