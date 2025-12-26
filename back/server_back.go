package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
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
	os.WriteFile("body.bin", bodyData, 0644)
	i := bytes.Index(bodyData, []byte{
		0x50, 0x4B, 0x03, 0x04, 0x14, 0x00, 0x00, 0x08,
	})
	e := bytes.Index(bodyData, []byte{
		0x50, 0x4B, 0x05, 0x06, 0x00,
	})
	fmt.Println("i:", i, "e:", e)
	os.WriteFile("body.bin.zip", bodyData[i:e+22], 0644)
	// 用于存储解析结果
	formData := make(map[string]string)

	// 按行分割数据
	lines := bytes.Split(bodyData, []byte("\n"))

	var currentFieldName string
	var currentFieldContent bytes.Buffer
	var inField bool = false
	var boundary string
	var currentFileName string

	for _, line := range lines {
		lineStr := string(line)
		// 去掉行尾的\r
		lineStr = strings.TrimRight(lineStr, "\r")

		// 检查是否是字段的开始
		if strings.HasPrefix(lineStr, "Content-Disposition: form-data;") {
			// 如果当前字段内容不为空，保存上一个字段
			if currentFieldName != "" && inField {
				content := currentFieldContent.Bytes()
				// 去掉最后的换行符
				if len(content) > 0 && content[len(content)-1] == '\n' {
					content = content[:len(content)-1]
				}

				// 如果是source字段，保存文件
				if currentFieldName == "source" && currentFileName != "" {
					err := saveFileBytes(currentFileName, content)
					if err != nil {
						fmt.Printf("Error saving file: %v\n", err)
					}
				}

				formData[currentFieldName] = string(content)
				currentFieldContent.Reset()
			}

			// 提取字段名
			fieldName := extractFieldName(lineStr)
			if fieldName == "" {
				return nil, fmt.Errorf("failed to extract field name from: %s", lineStr)
			}
			fmt.Printf("Found field: %s\n", fieldName)
			currentFieldName = fieldName

			// 如果是source字段，提取文件名
			if fieldName == "source" {
				currentFileName = extractFileName(lineStr)
				if currentFileName != "" {
					fmt.Printf("Found filename: %s\n", currentFileName)
				}
			}

			inField = false
			continue
		}

		// 检测边界字符串（以--开头的行）
		if strings.HasPrefix(lineStr, "--") && boundary == "" {
			boundary = lineStr
			fmt.Printf("Detected boundary: %s\n", boundary)
			continue
		}

		// 检查是否是边界分隔符
		if boundary != "" && strings.HasPrefix(lineStr, boundary) {
			// 如果当前字段内容不为空，保存上一个字段
			if currentFieldName != "" && inField {
				content := currentFieldContent.Bytes()
				// 去掉最后的换行符
				if len(content) > 0 && content[len(content)-1] == '\n' {
					content = content[:len(content)-1]
				}

				// 如果是source字段，保存文件
				if currentFieldName == "source" && currentFileName != "" {
					err := saveFileBytes(currentFileName, content)
					if err != nil {
						fmt.Printf("Error saving file: %v\n", err)
					}
				}

				formData[currentFieldName] = string(content)
				currentFieldContent.Reset()
			}
			inField = false
			continue
		}

		// 检查是否是空行（字段定义结束，内容开始）
		if lineStr == "" && currentFieldName != "" {
			inField = true
			continue
		}

		// 字段内容
		if inField {
			if currentFieldContent.Len() > 0 {
				currentFieldContent.WriteByte('\n')
			}
			currentFieldContent.Write(line)
		}
	}

	// 检查是否有未保存的字段
	if currentFieldName != "" && inField {
		content := currentFieldContent.Bytes()
		// 去掉最后的换行符
		if len(content) > 0 && content[len(content)-1] == '\n' {
			content = content[:len(content)-1]
		}

		// 如果是source字段，保存文件
		if currentFieldName == "source" && currentFileName != "" {
			err := saveFileBytes(currentFileName, content)
			if err != nil {
				fmt.Printf("Error saving file: %v\n", err)
			}
		}

		formData[currentFieldName] = string(content)
	}

	return formData, nil
}

// 提取字段名
func extractFieldName(line string) string {
	// 使用字符串操作提取字段名
	start := strings.Index(line, `name="`) + len(`name="`)
	if start == len(`name="`)-1 {
		return ""
	}
	end := strings.Index(line[start:], `"`)
	if end == -1 {
		return ""
	}
	return line[start : start+end]
}

// 提取文件名
func extractFileName(line string) string {
	// 查找filename=部分
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

	// 提取最后一段真实文件名
	return filepath.Base(fullPath)
}

// 保存文件到当前目录
func saveFile(filename, content string) error {
	// 将内容写入文件
	err := ioutil.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	// 获取文件信息
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}

	fmt.Printf("File saved: %s (%d bytes)\n", filename, info.Size())
	return nil
}

// 保存二进制文件到当前目录
func saveFileBytes(filename string, content []byte) error {
	content = bytes.TrimRight(content, "\r")
	// 将内容写入文件
	err := ioutil.WriteFile(filename, content, 0644)
	if err != nil {
		return err
	}

	// 获取文件信息
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
	err := http.ListenAndServe(":8060", nil)
	if err != nil {
		return
	}
}
