package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// 读取整行输入
func GetUserInput(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		// 如果是 EOF（比如输入被关闭或用户按了 Ctrl+Z），向上返回原始错误，调用方处理
		if err == io.EOF {
			return "", err
		}
		return "", fmt.Errorf("读取输入失败: %v", err)
	}
	// 去除输入前后的空格和换行符
	return strings.TrimSpace(input), nil
}

// 提取 AppID
func ExtractAppID(userInput string) (int, error) {
	input := strings.TrimSpace(userInput)
	if input == "" {
		return 0, fmt.Errorf("输入不能为空")
	}

	// 匹配链接中的 AppID
	appIDRegex := regexp.MustCompile(`(?:/app/|steamdb\.info/app/)(\d+)`)
	matches := appIDRegex.FindStringSubmatch(input)

	// 从URL提取AppID
	if len(matches) >= 2 {
		appIDStr := matches[1]
		appID, err := strconv.Atoi(appIDStr)
		if err != nil {
			return 0, fmt.Errorf("提取的AppID不是有效数字: %s", appIDStr)
		}
		fmt.Printf("从输入中提取到AppID: %d\n", appID)
		return appID, nil
	}

	// 输入本身是纯数字
	appID, err := strconv.Atoi(input)
	if err == nil {
		fmt.Printf("输入为纯数字AppID: %d\n", appID)
		return appID, nil
	}

	// 情况3：无效输入
	return 0, fmt.Errorf("无效格式，请输入含空格的游戏名称、Steam URL或纯数字AppID")
}

// 按游戏名称搜索AppID
func FindAppID(gameName string) ([]Game, error) {
	gameName = strings.TrimSpace(gameName)
	if gameName == "" {
		return nil, fmt.Errorf("游戏名称不能为空")
	}

	// 处理URL编码（支持空格、特殊字符）
	encodedName := url.QueryEscape(gameName)
	apiURL := fmt.Sprintf("https://steamui.com/api/loadGames.php?search=%s", encodedName)
	fmt.Printf("正在搜索游戏: %s (请求URL: %s)\n", gameName, apiURL)

	// 发送请求
	resp, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("搜索API返回错误状态码: %d", resp.StatusCode)
	}

	// 解析 JSON
	var response LoadGamesResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %v（可能API返回格式变更）", err)
	}

	// 过滤空结果
	if len(response.Games) == 0 {
		fmt.Printf("未找到与 '%s' 匹配的游戏\n", gameName)
		return nil, nil
	}

	// 输出搜索结果
	fmt.Printf("找到 %d 个匹配的游戏:\n", len(response.Games))
	for i, game := range response.Games {
		// 格式化输出，对齐显示
		fmt.Printf(" %d. %-30s | AppID: %d\n",
			i+1, game.Name, game.AppID)
	}

	return response.Games, nil
}

// AppID 选择
func GetAppID() (int, error) {
	fmt.Println(Division)
	// 读取整行输入
	input, err := GetUserInput("请输入游戏名称/AppID/Steam链接/SteamDB链接:")
	if err != nil {
		// 将 EOF 原样传递，调用方可选择退出
		if err == io.EOF {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("读取输入失败: %v", err)
	}

	// 优先尝试提取AppID
	appID, err := ExtractAppID(input)
	if err == nil {
		return appID, nil
	}

	// 提取失败，尝试按名称搜索
	fmt.Printf("无法直接提取AppID，将尝试按名称 '%s' 搜索...\n", input)
	games, err := FindAppID(input)
	if err != nil {
		return 0, fmt.Errorf("搜索游戏失败: %v", err)
	}

	// 无搜索结果
	if len(games) == 0 {
		return 0, fmt.Errorf("未找到任何匹配的游戏，请重新输入")
	}

	// 让用户选择游戏
	selectionStr, err := GetUserInput("请输入需要下载的游戏序号:")
	if err != nil {
		if err == io.EOF {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("读取选择失败: %v", err)
	}
	selection, err := strconv.Atoi(selectionStr)
	if err != nil {
		return 0, fmt.Errorf("序号必须是数字，请输入 1-%d 之间的序号", len(games))
	}

	// 验证序号合法性
	if selection < 1 || selection > len(games) {
		return 0, fmt.Errorf("无效序号，请选择 1-%d 之间的数字", len(games))
	}

	// 返回选中的AppID
	targetGame := games[selection-1]
	fmt.Printf("已选择游戏: %s (AppID: %d)\n", targetGame.Name, targetGame.AppID)
	return targetGame.AppID, nil
}
