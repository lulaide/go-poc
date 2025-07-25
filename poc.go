package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// POC 结构体定义
type POC struct {
	Name       string            `yaml:"name"`
	Manual     bool              `yaml:"manual"`
	Transport  string            `yaml:"transport"`
	Set        map[string]string `yaml:"set"`
	Rules      map[string]Rule   `yaml:"rules"`
	Expression string            `yaml:"expression"`
	Detail     Detail            `yaml:"detail"`
}

// Rule 结构体定义
type Rule struct {
	Request    Request `yaml:"request"`
	Expression string  `yaml:"expression"`
	Output     Output  `yaml:"output"`
}

// Request 结构体定义
type Request struct {
	Method          string            `yaml:"method"`
	Path            string            `yaml:"path"`
	Headers         map[string]string `yaml:"headers"`
	Body            string            `yaml:"body"`
	Cache           bool              `yaml:"cache"`
	FollowRedirects bool              `yaml:"follow_redirects"`
}

// Output 结构体定义
type Output struct {
	Search string `yaml:"search"`
	Filen  string `yaml:"filen"`
}

// Detail 结构体定义
type Detail struct {
	Author        string        `yaml:"author"`
	Links         []string      `yaml:"links"`
	Warning       string        `yaml:"warning"`
	Description   string        `yaml:"description"`
	Fingerprint   Fingerprint   `yaml:"fingerprint"`
	Vulnerability Vulnerability `yaml:"vulnerability"`
}

/*
Fingerprint 结构体定义
fingerprint: # 当该插件为指纹插件时才需要填写，且一般会自动生成

		id: # 指纹库id，一般自动生成，可以不用填写
	    name: # 通用名称，一般自动生成，可以不用填写
	    version: '{{version}}' # 用来接收指纹插件匹配到的版本信息，一般直接用这样的固定格式即可，会自动将output中匹配到的内容渲染过来
	    cpe: # 输出该指纹对应的产品的cpe，一般自动生成，可以不用填写
	    xxxx: # 一些希望被输出的内容，输出时，xxxx就是字段名称，后面就是输出值
*/
type Fingerprint struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	CPE     string `yaml:"cpe"`
}

/*
Vulnerability 结构体定义
vulnerability: # 当该插件为漏洞插件且需要输出一些内容时才需要填写，且一般会自动生成

	id: # 漏洞库id，一般自动生成，可以不用填写
	level: # 漏洞危害等级，一般自动根据漏洞信息生成，可以不用填写
	match: # 一些证明漏洞存在的信息，比如信息泄露泄露的一些敏感数据等
	xxxx: # 一些希望被输出的内容，输出时，xxxx就是字段名称，后面就是输出值
*/
type Vulnerability struct {
	ID    string `yaml:"id"`
	Level string `yaml:"level"`
	Match string `yaml:"match"`
}

// 读取POC文件并解析
func loadPOC(pocFile string) (POC, error) {
	// 检查文件是否存在
	if _, err := os.Stat(pocFile); os.IsNotExist(err) {
		return POC{}, fmt.Errorf("POC文件不存在: %s", pocFile)
	}

	// 打开文件
	file, err := os.Open(pocFile)
	if err != nil {
		return POC{}, fmt.Errorf("无法打开POC文件: %v", err)
	}
	defer file.Close()

	// 读取文件内容
	var poc POC
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&poc); err != nil {
		return POC{}, fmt.Errorf("解析YAML文件失败: %v", err)
	}

	// 验证必要字段
	if poc.Name == "" {
		return POC{}, fmt.Errorf("POC文件缺少必要字段: name")
	}
	if poc.Transport == "" {
		poc.Transport = "http" // 设置默认值
	}
	if len(poc.Rules) == 0 {
		return POC{}, fmt.Errorf("POC文件缺少必要字段: rules")
	}
	if poc.Expression == "" {
		return POC{}, fmt.Errorf("POC文件缺少必要字段: expression")
	}

	return poc, nil
}
