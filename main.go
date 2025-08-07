package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	// 定义子命令
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runPocFile := runCmd.String("poc", "", "POC YAML文件路径 (必需)")
	runTargetURL := runCmd.String("target", "", "目标网站URL (必需)")
	runDebug := runCmd.Bool("debug", false, "启用调试模式")

	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	searchKeyword := searchCmd.String("keyword", "", "用于搜索POC的关键词 (必需)")
	searchTargetURL := searchCmd.String("target", "", "目标网站URL (必需)")
	searchAll := searchCmd.Bool("all", false, "自动执行所有搜索到的POC")
	searchDebug := searchCmd.Bool("debug", false, "启用调试模式")

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)

	// 检查输入了哪个子命令
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd.Parse(os.Args[2:])
		if *runPocFile == "" || *runTargetURL == "" {
			fmt.Println("Usage: go run . run --poc <poc_file> --target <url> [--debug]")
			os.Exit(1)
		}
		executeSinglePOC(*runPocFile, *runTargetURL, *runDebug)

	case "search":
		searchCmd.Parse(os.Args[2:])
		if *searchKeyword == "" || *searchTargetURL == "" {
			fmt.Println("Usage: go run . search --keyword <keyword> --target <url> [--all] [--debug]")
			os.Exit(1)
		}

		// 搜索POC
		results, err := searchPOCs(*searchKeyword)
		if err != nil {
			log.Fatalf("搜索POC失败: %v", err)
		}

		if len(results) == 0 {
			fmt.Println("未找到匹配的POC。")
			return
		}

		if *searchAll {
			// 自动执行所有找到的POC
			fmt.Printf("找到 %d 个匹配的POC，将全部自动执行...\n\n", len(results))
			for i, pocInfo := range results {
				fmt.Printf("--- (%d/%d) 开始执行: %s ---\n", i+1, len(results), pocInfo.Name)
				executeSinglePOC(pocInfo.Path, *searchTargetURL, *searchDebug)
				fmt.Printf("--- %s 执行完毕 ---\n\n", pocInfo.Name)
			}
		} else {
			// 手动选择
			selectedPOC, err := promptUserSelection(results)
			if err != nil {
				log.Fatalf("选择POC失败: %v", err)
			}
			executeSinglePOC(selectedPOC.Path, *searchTargetURL, *searchDebug)
		}

	case "list":
		listCmd.Parse(os.Args[2:])
		if err := listAllPOCs(); err != nil {
			log.Fatalf("列出POC失败: %v", err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

// executeSinglePOC 封装了执行单个POC的逻辑
func executeSinglePOC(pocFile, targetURL string, debug bool) {
	// 读取并解析POC文件
	poc, err := loadPOC(pocFile)
	if err != nil {
		log.Printf("加载POC文件 %s 失败: %v", pocFile, err)
		return
	}

	// 输出POC信息
	fmt.Printf("POC名称: %s\n", poc.Name)
	fmt.Printf("目标URL: %s\n", targetURL)
	if debug {
		fmt.Println("调试模式已启用")
	}

	// 执行POC
	success, err := executePOC(poc, targetURL, debug)
	if err != nil {
		log.Printf("POC %s 执行失败: %v", poc.Name, err)
		return
	}

	if success {
		fmt.Printf("【成功】POC %s 执行成功，目标可能存在漏洞！\n", poc.Name)
	} else {
		fmt.Printf("【失败】POC %s 执行完成，未发现漏洞。\n", poc.Name)
	}
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println("Usage: go run . <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  run      执行单个POC文件")
	fmt.Println("  search   根据关键词搜索并执行POC")
	fmt.Println("  list     列出所有可用的POC")
	fmt.Println("\nUse 'go run . <command> --help' for more information about a command.")
}
