package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// executePOC 执行单个POC
func executePOC(poc POC, targetURL string, debug bool) (bool, error) {
	ruleResults := make(map[string]func() bool)

	if debug {
		fmt.Println("=== 开始执行POC规则 ===")
	}

	for ruleName, rule := range poc.Rules {
		if debug {
			fmt.Printf("\n--- 执行规则 %s ---\n", ruleName)
		}

		// 1. 构造HTTP请求
		req, err := buildRequest(rule.Request, targetURL)
		if err != nil {
			return false, fmt.Errorf("规则 '%s' 请求构建失败: %v", ruleName, err)
		}

		if debug {
			// 输出请求信息
			fmt.Printf("请求方法: %s\n", req.Method)
			fmt.Printf("请求URL: %s\n", req.URL.String())
			if len(req.Header) > 0 {
				fmt.Println("请求头:")
				for k, v := range req.Header {
					fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
				}
			}
			if req.Body != nil {
				bodyBytes, _ := io.ReadAll(req.Body)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // 重新填充body
				if len(bodyBytes) > 0 {
					fmt.Printf("请求体: %s\n", string(bodyBytes))
				}
			}
		}

		// 2. 发送HTTP请求
		client := &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if rule.Request.FollowRedirects {
					return nil
				}
				return http.ErrUseLastResponse
			},
		}

		if debug {
			fmt.Println("发送请求...")
		}
		resp, err := client.Do(req)
		if err != nil {
			if debug {
				fmt.Printf("请求失败: %v\n", err)
			}
			return false, fmt.Errorf("规则 '%s' 请求失败: %v", ruleName, err)
		}
		defer resp.Body.Close()

		// 3. 读取响应体
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, fmt.Errorf("规则 '%s' 读取响应体失败: %v", ruleName, err)
		}

		if debug {
			// 输出响应信息
			fmt.Printf("响应状态: %d %s\n", resp.StatusCode, resp.Status)
			if len(resp.Header) > 0 {
				fmt.Println("响应头:")
				for k, v := range resp.Header {
					fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
				}
			}
			fmt.Printf("响应体长度: %d 字节\n", len(respBody))
			if len(respBody) > 0 {
				bodyStr := string(respBody)
				if len(bodyStr) > 500 {
					fmt.Printf("响应体预览: %s...\n", bodyStr[:500])
				} else {
					fmt.Printf("响应体: %s\n", bodyStr)
				}
			}
		}

		// 4. 执行规则表达式
		if debug {
			fmt.Printf("执行表达式: %s\n", rule.Expression)
		}
		result, err := evaluateCELExpression(rule.Expression, resp, respBody)
		if err != nil {
			if debug {
				fmt.Printf("表达式执行失败: %v\n", err)
			}
			return false, fmt.Errorf("规则 '%s' 表达式执行失败: %v", ruleName, err)
		}

		if debug {
			fmt.Printf("规则 %s 执行结果: %t\n", ruleName, result)
		}

		// 保存规则结果为函数
		ruleResult := result
		ruleResults[ruleName] = func() bool { return ruleResult }
	}

	// 5. 执行顶层表达式
	if debug {
		fmt.Printf("\n=== 执行顶层表达式 ===\n")
		fmt.Printf("顶层表达式: %s\n", poc.Expression)
	}

	finalResult, err := evaluateTopLevelExpression(poc.Expression, ruleResults)
	if err != nil {
		if debug {
			fmt.Printf("顶层表达式执行失败: %v\n", err)
		}
		return false, fmt.Errorf("顶层表达式执行失败: %v", err)
	}

	if debug {
		fmt.Printf("最终结果: %t\n", finalResult)
		fmt.Println("=== POC执行完成 ===")
	}

	return finalResult, nil
}

// evaluateCELExpression 评估CEL类似的表达式
func evaluateCELExpression(expression string, resp *http.Response, body []byte) (bool, error) {
	// 创建响应对象
	responseObj := createResponseObject(resp, body)

	// 简单的表达式评估器，处理常见的POC表达式模式
	return evaluateExpression(expression, responseObj)
}

// evaluateExpression 评估表达式
func evaluateExpression(expr string, responseObj map[string]interface{}) (bool, error) {
	// 清理表达式，移除多行字符串的格式
	expr = strings.TrimSpace(expr)
	expr = strings.ReplaceAll(expr, "\n", " ")
	expr = strings.ReplaceAll(expr, "\r", "")

	// 特殊情况：直接返回 true
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}

	// 处理 && 操作符
	if strings.Contains(expr, "&&") {
		parts := strings.Split(expr, "&&")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			result, err := evaluateExpression(part, responseObj)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// 处理 || 操作符
	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			result, err := evaluateExpression(part, responseObj)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// 处理具体的表达式
	expr = strings.TrimSpace(expr)

	// reverse.wait(5) - 反向连接等待，暂时返回false
	if strings.Contains(expr, "reverse.wait") {
		return false, nil
	}

	// response.status == 200 或 response.status != 200
	if strings.HasPrefix(expr, "response.status") {
		return evaluateNumericComparison(expr, responseObj["status"].(int))
	}

	// response.headers["location"] == "/menu.gch"
	if strings.Contains(expr, "response.headers[") {
		re := regexp.MustCompile(`response\.headers\["([^"]+)"\]\s*==\s*"([^"]*)"`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 2 {
			headerName := strings.ToLower(matches[1])
			expectedValue := matches[2]
			headers := responseObj["headers"].(map[string]string)
			actualValue, exists := headers[headerName]
			if !exists {
				return false, nil
			}
			return actualValue == expectedValue, nil
		}
	}

	// "set-cookie" in response.headers
	if strings.Contains(expr, "in response.headers") {
		re := regexp.MustCompile(`"([^"]+)"\s+in\s+response\.headers`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			headerName := strings.ToLower(matches[1])
			headers := responseObj["headers"].(map[string]string)
			_, exists := headers[headerName]
			return exists, nil
		}
	}

	// response.content_type.contains("json")
	if strings.Contains(expr, "response.content_type.contains") {
		re := regexp.MustCompile(`response\.content_type\.contains\("([^"]+)"\)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			contentType := responseObj["content_type"].(string)
			return strings.Contains(strings.ToLower(contentType), strings.ToLower(matches[1])), nil
		}
	}

	// response.body.bcontains(b"text")
	if strings.Contains(expr, "response.body.bcontains(b\"") {
		re := regexp.MustCompile(`response\.body\.bcontains\(b"((?:\\.|[^"])*)"\)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			contentToSearch := strings.ReplaceAll(matches[1], `\"`, `"`)
			bodyStr := responseObj["body"].(string)
			return strings.Contains(bodyStr, contentToSearch), nil
		}
	}

	// response.body.bcontains(bytes(...)) - 复杂的字节表达式，暂时返回false
	if strings.Contains(expr, "response.body.bcontains(bytes(") {
		return false, nil
	}

	// "pattern".matches(response.body_string) 或 "pattern".matches(response.body)
	if strings.Contains(expr, ".matches(response.body") {
		re := regexp.MustCompile(`"([^"]+)"\.matches\(response\.body[_a-z]*\)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			pattern := matches[1]
			bodyStr := responseObj["body_string"].(string)
			matched, err := regexp.MatchString(pattern, bodyStr)
			if err != nil {
				return false, fmt.Errorf("正则表达式错误: %v", err)
			}
			return matched, nil
		}
	}

	// "pattern".bmatches(response.body) - 字节匹配
	if strings.Contains(expr, ".bmatches(response.body") {
		re := regexp.MustCompile(`"([^"]+)"\.bmatches\(response\.body[_a-z]*\)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			pattern := matches[1]
			bodyStr := responseObj["body_string"].(string)
			matched, err := regexp.MatchString(pattern, bodyStr)
			if err != nil {
				return false, fmt.Errorf("正则表达式错误: %v", err)
			}
			return matched, nil
		}
	}

	// response.body_string.matches("pattern") 或类似形式
	if strings.Contains(expr, "response.body") && strings.Contains(expr, ".matches(") {
		re := regexp.MustCompile(`response\.body[_a-z]*\.matches\("([^"]+)"\)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			pattern := matches[1]
			bodyStr := responseObj["body_string"].(string)
			matched, err := regexp.MatchString(pattern, bodyStr)
			if err != nil {
				return false, fmt.Errorf("正则表达式错误: %v", err)
			}
			return matched, nil
		}
	}

	// response.body.bmatches("pattern") 或类似形式
	if strings.Contains(expr, "response.body") && strings.Contains(expr, ".bmatches(") {
		re := regexp.MustCompile(`response\.body[_a-z]*\.bmatches\("([^"]+)"\)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) > 1 {
			pattern := matches[1]
			bodyStr := responseObj["body_string"].(string)
			matched, err := regexp.MatchString(pattern, bodyStr)
			if err != nil {
				return false, fmt.Errorf("正则表达式错误: %v", err)
			}
			return matched, nil
		}
	}

	return false, fmt.Errorf("无法解析表达式: %s", expr)
} // evaluateNumericComparison 评估数值比较
func evaluateNumericComparison(expr string, actualValue int) (bool, error) {
	if strings.Contains(expr, "==") {
		parts := strings.Split(expr, "==")
		if len(parts) != 2 {
			return false, fmt.Errorf("无效的比较表达式: %s", expr)
		}
		expectedStr := strings.TrimSpace(parts[1])
		expected, err := strconv.Atoi(expectedStr)
		if err != nil {
			return false, fmt.Errorf("无法解析数值: %s", expectedStr)
		}
		return actualValue == expected, nil
	}

	if strings.Contains(expr, "!=") {
		parts := strings.Split(expr, "!=")
		if len(parts) != 2 {
			return false, fmt.Errorf("无效的比较表达式: %s", expr)
		}
		expectedStr := strings.TrimSpace(parts[1])
		expected, err := strconv.Atoi(expectedStr)
		if err != nil {
			return false, fmt.Errorf("无法解析数值: %s", expectedStr)
		}
		return actualValue != expected, nil
	}

	return false, fmt.Errorf("不支持的比较操作符: %s", expr)
}

// evaluateTopLevelExpression 评估顶层表达式 (如 r0() && r1() && r2())
func evaluateTopLevelExpression(expression string, ruleResults map[string]func() bool) (bool, error) {
	// 简单的表达式解析器，支持 && 和 || 操作符
	// 将 r0() && r1() && r2() 这样的表达式转换为实际的布尔运算

	// 清理表达式，移除多行字符串的格式
	expression = strings.TrimSpace(expression)
	expression = strings.ReplaceAll(expression, "\n", "")
	expression = strings.ReplaceAll(expression, "\r", "")

	// 替换规则函数调用
	for ruleName, ruleFunc := range ruleResults {
		pattern := ruleName + "()"
		result := ruleFunc()
		replacement := fmt.Sprintf("%t", result)
		expression = strings.ReplaceAll(expression, pattern, replacement)
	}

	// 现在表达式应该是类似 "true && true && true" 的形式
	// 使用简单的解析
	return evaluateSimpleBooleanExpression(expression)
}

// evaluateSimpleBooleanExpression 评估简单的布尔表达式
func evaluateSimpleBooleanExpression(expression string) (bool, error) {
	// 移除空格和换行符
	expr := strings.TrimSpace(expression)
	expr = strings.ReplaceAll(expr, " ", "")
	expr = strings.ReplaceAll(expr, "\n", "")
	expr = strings.ReplaceAll(expr, "\r", "")
	expr = strings.ReplaceAll(expr, "\t", "")

	// 处理 && 操作符
	if strings.Contains(expr, "&&") {
		parts := strings.Split(expr, "&&")
		for _, part := range parts {
			result, err := evaluateSimpleBooleanExpression(part)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// 处理 || 操作符
	if strings.Contains(expr, "||") {
		parts := strings.Split(expr, "||")
		for _, part := range parts {
			result, err := evaluateSimpleBooleanExpression(part)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// 基本布尔值
	if expr == "true" {
		return true, nil
	} else if expr == "false" {
		return false, nil
	}

	return false, fmt.Errorf("无法解析表达式: %s", expr)
}

// createResponseObject 创建用于CEL表达式的响应对象
func createResponseObject(resp *http.Response, body []byte) map[string]interface{} {
	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[strings.ToLower(k)] = strings.Join(v, " ")
	}

	responseObj := map[string]interface{}{
		"status":       resp.StatusCode,
		"headers":      headers,
		"body":         string(body),
		"body_string":  string(body), // 添加 body_string 别名
		"content_type": resp.Header.Get("Content-Type"),
	}

	return responseObj
}

// buildRequest 根据规则构建HTTP请求
func buildRequest(reqData Request, targetURL string) (*http.Request, error) {
	fullURL := targetURL + reqData.Path
	req, err := http.NewRequest(reqData.Method, fullURL, bytes.NewBufferString(reqData.Body))
	if err != nil {
		return nil, err
	}

	for key, value := range reqData.Headers {
		req.Header.Set(key, value)
	}
	// 添加默认的User-Agent
	if _, ok := reqData.Headers["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	}

	return req, nil
}
