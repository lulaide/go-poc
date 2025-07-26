package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PocFileInfo 存储POC文件信息
type PocFileInfo struct {
	Name        string
	Path        string
	Description string
}

// searchPOCs 搜索POC文件
func searchPOCs(keyword string) ([]PocFileInfo, error) {
	var results []PocFileInfo
	pocDir := "xray/pocs"

	// 检查POC目录是否存在
	if _, err := os.Stat(pocDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("POC目录不存在: %s", pocDir)
	}

	err := filepath.Walk(pocDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理.yml文件
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".yml") {
			// 检查文件名是否包含关键词
			fileName := strings.ToLower(info.Name())
			searchKeyword := strings.ToLower(keyword)

			if strings.Contains(fileName, searchKeyword) {
				// 尝试读取POC文件获取更多信息
				pocInfo := PocFileInfo{
					Name: info.Name(),
					Path: path,
				}

				// 尝试从文件中提取描述信息
				if desc, err := extractPOCDescription(path); err == nil {
					pocInfo.Description = desc
				}

				results = append(results, pocInfo)
			}
		}

		return nil
	})

	return results, err
}

// extractPOCDescription 从POC文件中提取描述信息
func extractPOCDescription(filePath string) (string, error) {
	// 直接使用loadPOC函数来获取完整的POC信息
	poc, err := loadPOC(filePath)
	if err != nil {
		return "", err
	}

	// 构建描述信息
	var description strings.Builder
	if poc.Name != "" {
		description.WriteString(poc.Name)
	}

	if poc.Detail.Description != "" {
		if description.Len() > 0 {
			description.WriteString(" - ")
		}
		description.WriteString(poc.Detail.Description)
	}

	if poc.Detail.Author != "" {
		if description.Len() > 0 {
			description.WriteString(" (作者: ")
			description.WriteString(poc.Detail.Author)
			description.WriteString(")")
		}
	}

	if description.Len() == 0 {
		return "无描述信息", nil
	}

	return description.String(), nil
} // promptUserSelection 提示用户选择POC
func promptUserSelection(results []PocFileInfo) (*PocFileInfo, error) {
	fmt.Printf("\n找到 %d 个匹配的POC：\n\n", len(results))

	for i, poc := range results {
		fmt.Printf("[%d] %s\n", i+1, poc.Name)
		if poc.Description != "" {
			fmt.Printf("    %s\n", poc.Description)
		}
		fmt.Printf("    路径: %s\n\n", poc.Path)
	}

	fmt.Print("请选择要执行的POC (输入序号): ")
	scanner := bufio.NewScanner(os.Stdin)

	if !scanner.Scan() {
		return nil, fmt.Errorf("读取用户输入失败")
	}

	input := strings.TrimSpace(scanner.Text())
	choice, err := strconv.Atoi(input)
	if err != nil {
		return nil, fmt.Errorf("无效的选择: %s", input)
	}

	if choice < 1 || choice > len(results) {
		return nil, fmt.Errorf("选择超出范围: %d", choice)
	}

	return &results[choice-1], nil
}

// listAllPOCs 列出所有可用的POC文件
func listAllPOCs() error {
	pocDir := "xray/pocs"

	// 检查POC目录是否存在
	if _, err := os.Stat(pocDir); os.IsNotExist(err) {
		return fmt.Errorf("POC目录不存在: %s", pocDir)
	}

	fmt.Println("所有可用的POC文件：")

	count := 0
	err := filepath.Walk(pocDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理.yml文件
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".yml") {
			count++
			fmt.Printf("[%d] %s\n", count, info.Name())

			// 尝试提取POC描述
			if desc, err := extractPOCDescription(path); err == nil {
				fmt.Printf("    %s\n", desc)
			}
			fmt.Printf("    路径: %s\n\n", path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("总共找到 %d 个POC文件。\n", count)
	return nil
}
