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

// 手动解析非标准的 multipart/form-data
func parseCustomFormData(r *http.Request) (map[string]string, error) {
	// 读取所有请求体数据
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// 用于存储解析结果
	formData := make(map[string]string)
	
	// 查找边界字符串
	boundary := findBoundary(bodyData)
	if boundary == "" {
		return nil, fmt.Errorf("no boundary found")
	}
	fmt.Printf("Detected boundary: %s\n", boundary)

	// 按边界分割数据
	boundaryBytes := []byte(boundary)
	parts := bytes.Split(bodyData, boundaryBytes)
	
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		
		// 跳过开头和结尾的空parts
		if i == 0 || i == len(parts)-1 {
			continue
		}
		
		// 查找Content-Disposition行
		headerEnd := bytes.Index(part, []byte("\r\n\r\n"))
		if headerEnd == -1 {
			headerEnd = bytes.Index(part, []byte("\n\n"))
		}
		if headerEnd == -1 {
			continue
		}
		
		headerPart := part[:headerEnd]
		contentPart := part[headerEnd:]
		
		// 去掉content开头的换行符
		if bytes.HasPrefix(contentPart, []byte("\r\n\r\n")) {
			contentPart = contentPart[4:]
		} else if bytes.HasPrefix(contentPart, []byte("\n\n")) {
			contentPart = contentPart[2:]
		}
		
		// 解析header
		headerLines := bytes.Split(headerPart, []byte("\n"))
		var fieldName, fileName string
		
		for _, line := range headerLines {
			lineStr := strings.TrimRight(string(line), "\r")
			
			if strings.HasPrefix(lineStr, "Content-Disposition: form-data;") {
				fieldName = extractFieldName(lineStr)
				if fieldName == "source" {
					fileName = extractFileName(lineStr)
					fmt.Printf("Found field: %s, filename: %s\n", fieldName, fileName)
				} else {
					fmt.Printf("Found field: %s\n", fieldName)
				}
			}
		}
		
		if fieldName == "" {
			continue
		}
		
		// 去掉content结尾的换行符
		if bytes.HasSuffix(contentPart, []byte("\r\n")) {
			contentPart = contentPart[:len(contentPart)-2]
		} else if bytes.HasSuffix(contentPart, []byte("\n")) {
			contentPart = contentPart[:len(contentPart)-1]
		}
		
		// 如果是source字段，保存文件
		if fieldName == "source" && fileName != "" {
			err := saveFileBytes(fileName, contentPart)
			if err != nil {
				fmt.Printf("Error saving file: %v\n", err)
			}
		}
		
		formData[fieldName] = string(contentPart)
	}

	return formData, nil
}

// 查找边界字符串
func findBoundary(data []byte) string {
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		lineStr := strings.TrimRight(string(line), "\r")
		if strings.HasPrefix(lineStr, "--") {
			return lineStr
		}
	}
	return ""
}

// 提取字段名
func extractFieldName(line string) string {
	start := strings.Index(line, `name="`)
	if start == -1 {
		return ""
	}
	start += len(`name="`)
	end := strings.Index(line[start:], `"`)
	if end == -1 {
		return ""
	}
	return line[start : start+end]
}

// 提取文件名
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
	fullPath := line[start : start+end]
	return filepath.Base(fullPath)
}

// 保存二进制文件到当前目录
func saveFileBytes(filename string, content []byte) error {
	err := os.WriteFile(filename, content, 0644)
	if err != nil {
		return err
	}
	
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	
	fmt.Printf("File saved: %s (%d bytes)\n", filename, info.Size())
	return nil
}

func HelloHandle(w http.ResponseWriter, req *http.Request) {
	fmt.Println("method:", req.Method)
	fmt.Println("Content-Type:", req.Header.Get("Content-Type"))
	fmt.Println("Content-Length:", req.Header.Get("Content-Length"))
	
	// 解析 multipart/form-data
	formData, err := parseCustomFormData(req)
	if err != nil {
		fmt.Printf("Error parsing form data: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Parsed %d fields\n", len(formData))
	
	// 输出解析结果
	if len(formData) == 0 {
		fmt.Fprintf(w, "No form data found\n")
	} else {
		for key, value := range formData {
			if key == "source" {
				fmt.Printf("Field: %s, Value: [binary data - %d bytes]\n", key, len(value))
				fmt.Fprintf(w, "Field: %s, Value: [binary data - %d bytes]\n", key, len(value))
			} else {
				// 检查是否包含二进制数据
				isBinary := false
				for _, b := range []byte(value) {
					if b < 32 && b != 9 && b != 10 && b != 13 {
						isBinary = true
						break
					}
				}
				
				if isBinary {
					fmt.Printf("Field: %s, Value: [binary data - %d bytes]\n", key, len(value))
					fmt.Fprintf(w, "Field: %s, Value: [binary data - %d bytes]\n", key, len(value))
				} else {
					// 只显示前50个字符，避免过长内容
					displayValue := value
					if len(value) > 50 {
						displayValue = value[:50] + "..."
					}
					fmt.Printf("Field: %s, Value: %s\n", key, displayValue)
					fmt.Fprintf(w, "Field: %s, Value: %s\n", key, value)
				}
			}
		}
	}
}

func main() {
	http.HandleFunc("/depot/publish/", HelloHandle)
	fmt.Println("Server starting on :8060")
	err := http.ListenAndServe(":8060", nil)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}