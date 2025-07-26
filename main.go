package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	// 定义命令行标志
	pocFile := flag.String("poc", "", "POC YAML文件路径")
	targetURL := flag.String("target", "", "目标网站URL")
	debug := flag.Bool("debug", false, "启用调试模式")

	// 解析命令行输入
	flag.Parse()

	// 检查必需的参数
	if *pocFile == "" || *targetURL == "" {
		fmt.Println("Usage: go run main.go --poc <poc_file> --target <url> [--debug]")
		os.Exit(1)
	}

	// 读取并解析POC文件
	poc, err := loadPOC(*pocFile)
	if err != nil {
		log.Fatalf("加载POC文件失败: %v", err)
	}

	// 输出POC信息
	fmt.Printf("POC名称: %s\n", poc.Name)
	fmt.Printf("目标URL: %s\n", *targetURL)
	fmt.Printf("传输协议: %s\n", poc.Transport)
	fmt.Printf("规则数量: %d\n", len(poc.Rules))

	if poc.Detail.Author != "" {
		fmt.Printf("作者: %s\n", poc.Detail.Author)
	}

	if poc.Detail.Description != "" {
		fmt.Printf("描述: %s\n", poc.Detail.Description)
	}

	if *debug {
		fmt.Println("调试模式已启用")
		fmt.Printf("表达式: %s\n", poc.Expression)
		fmt.Println("规则详情:")
		for ruleName, rule := range poc.Rules {
			fmt.Printf("  规则 %s:\n", ruleName)
			fmt.Printf("    方法: %s\n", rule.Request.Method)
			fmt.Printf("    路径: %s\n", rule.Request.Path)
			if len(rule.Request.Headers) > 0 {
				fmt.Printf("    请求头: %v\n", rule.Request.Headers)
			}
			if rule.Request.Body != "" {
				fmt.Printf("    请求体: %s\n", rule.Request.Body)
			}
			fmt.Printf("    表达式: %s\n", rule.Expression)
		}
	}

	fmt.Println("开始执行POC...")
	success, err := executePOC(poc, *targetURL)
	if err != nil {
		log.Fatalf("POC执行失败: %v", err)
	}

	if success {
		fmt.Println("POC执行成功，目标存在漏洞！")
	} else {
		fmt.Println("POC执行完成，未发现漏洞。")
	}
}
