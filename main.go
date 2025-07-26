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
	search := flag.String("search", "", "搜索POC文件（根据关键词搜索）")
	listPocs := flag.Bool("list", false, "列出所有可用的POC文件")

	// 解析命令行输入
	flag.Parse()

	// 如果使用列表功能
	if *listPocs {
		err := listAllPOCs()
		if err != nil {
			log.Fatalf("列出POC失败: %v", err)
		}
		return
	}

	// 如果使用搜索功能
	if *search != "" {
		if *targetURL == "" {
			fmt.Println("搜索模式需要指定目标URL:")
			fmt.Println("Usage: ./go-poc --search <keyword> --target <url> [--debug]")
			os.Exit(1)
		}

		// 搜索POC文件
		results, err := searchPOCs(*search)
		if err != nil {
			log.Fatalf("搜索POC失败: %v", err)
		}

		if len(results) == 0 {
			fmt.Printf("未找到包含关键词 '%s' 的POC文件\n", *search)
			os.Exit(1)
		}

		var selectedPOC *PocFileInfo
		if len(results) == 1 {
			// 只有一个结果，直接使用
			selectedPOC = &results[0]
			fmt.Printf("找到1个匹配的POC，直接执行: %s\n", selectedPOC.Name)
		} else {
			// 多个结果，让用户选择
			var err error
			selectedPOC, err = promptUserSelection(results)
			if err != nil {
				log.Fatalf("选择POC失败: %v", err)
			}
		}

		// 设置pocFile为选中的POC路径
		*pocFile = selectedPOC.Path
		fmt.Printf("\n执行POC: %s\n", selectedPOC.Name)
	}

	// 检查必需的参数
	if *pocFile == "" || *targetURL == "" {
		fmt.Println("Usage:")
		fmt.Println("  执行指定POC: ./go-poc --poc <poc_file> --target <url> [--debug]")
		fmt.Println("  搜索并执行POC: ./go-poc --search <keyword> --target <url> [--debug]")
		fmt.Println("  列出所有POC: ./go-poc --list")
		fmt.Println("\n参数说明:")
		fmt.Println("  --poc      指定POC文件路径")
		fmt.Println("  --search   根据关键词搜索POC文件")
		fmt.Println("  --target   目标网站URL")
		fmt.Println("  --debug    启用详细调试信息")
		fmt.Println("  --list     列出所有可用的POC文件")
		fmt.Println("\n示例:")
		fmt.Println("  ./go-poc --poc xray/pocs/apache-ambari-default-password.yml --target http://example.com")
		fmt.Println("  ./go-poc --search apache --target http://example.com")
		fmt.Println("  ./go-poc --list")
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
