package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// 保存文件到配置路径
func SaveFile(path, filename string, data []byte) error {
	// 确保路径是绝对路径
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}

	// 确保目录存在
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 创建完整文件路径
	fullPath := filepath.Join(path, filename)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	fmt.Printf("文件已保存到: %s (%d字节)\n", fullPath, len(data))
	return nil
}

// 文件处理
func ProcessFile(data []byte) []byte {
	// 定义变量
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var builder strings.Builder

	// 死循环
	for scanner.Scan() {
		// 定义变量
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// 注释包含 setManifest 且未被注释的行
		if strings.Contains(trimmedLine, "setManifest") && !strings.HasPrefix(trimmedLine, "--") {
			builder.WriteString("-- ")
			fmt.Printf("已注释: %s\n", strings.TrimSpace(line))
		}

		builder.WriteString(line)
		builder.WriteByte('\n') // 保留换行符
	}

	// 返回
	fmt.Println(Division)
	return []byte(builder.String())
}

// 修补 DepotKey
func PatchDepotkey(APPID string, data []byte, depotkeys map[string]string) []byte {
	depotkey, exists := depotkeys[APPID]
	if !exists {
		fmt.Printf("没有找到AppID %s 的 DepotKey\n", APPID)
		return data
	}

	fmt.Printf("找到 AppID %s 的 DepotKey: %s\n", APPID, depotkey)

	// 创建正则表达式
	patternStr := `addappid\s*\(\s*` + regexp.QuoteMeta(APPID) + `\s*\)`
	fmt.Printf("使用的正则表达式: %s\n", patternStr)
	pattern := regexp.MustCompile(patternStr)

	// 检查匹配
	if matches := pattern.Find(data); matches != nil {
		fmt.Printf("发现匹配内容: %s\n", string(matches))
		fmt.Printf("发现需要修补的 addappid(%s)\n", APPID)

		// 替换为带 DepotKey 的版本
		replacement := fmt.Sprintf("addappid(%s,1,\"%s\")", APPID, depotkey)
		fmt.Printf("替换为: %s\n", replacement)

		patched := pattern.ReplaceAll(data, []byte(replacement))

		fmt.Println("已修补 DepotKey")
		fmt.Println(Division)
		return patched
	}

	fmt.Printf("未找到需要修补的 addappid(%s)\n", APPID)
	fmt.Println(Division)
	return data
}

// 添加 DLC 到 Lua 文件
func AddDLC(appid, luaFilePath string) error {
	// 获取游戏的基本信息
	mainDLCs, _, err := GetDLCInfo(appid)
	if err != nil {
		return fmt.Errorf("获取主游戏DLC失败: %v", err)
	}

	// 筛选无仓库的DLC
	var dlcIDs []string
	for _, dlcID := range mainDLCs {
		_, hasDepots, err := GetDLCInfo(dlcID)
		if err != nil {
			fmt.Printf("获取DLC %s 信息失败: %v\n", dlcID, err)
			continue
		}

		if !hasDepots {
			dlcIDs = append(dlcIDs, dlcID)
		}
	}

	if len(dlcIDs) == 0 {
		return fmt.Errorf("未找到无仓库的DLC")
	}

	// 读取现有LUA内容
	var existingLines []string
	if _, err := os.Stat(luaFilePath); err == nil {
		file, err := os.Open(luaFilePath)
		if err != nil {
			return fmt.Errorf("打开文件失败: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				existingLines = append(existingLines, line)
			}
		}
	}

	// 过滤已存在的DLC
	existingAppids := make(map[string]bool)
	for _, line := range existingLines {
		if matches := regexp.MustCompile(`addappid\((\d+)`).FindStringSubmatch(line); len(matches) > 1 {
			existingAppids[matches[1]] = true
		}
	}

	// 添加新DLC
	var newLines []string
	for _, dlcID := range dlcIDs {
		if !existingAppids[dlcID] {
			newLines = append(newLines, fmt.Sprintf("addappid(%s)", dlcID))
		}
	}

	if len(newLines) == 0 {
		return fmt.Errorf("所有无仓库的DLC已存在于解锁文件中")
	}

	// 保存回文件
	file, err := os.OpenFile(luaFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	for _, line := range newLines {
		if _, err := file.WriteString(line + "\n"); err != nil {
			fmt.Printf("写入DLC %s 失败: %v\n", line, err)
		} else {
			fmt.Printf("添加DLC: %s\n", line)
		}
	}

	return nil
}

// 获取DLC信息
func GetDLCInfo(appid string) ([]string, bool, error) {
	url := fmt.Sprintf(DLCInfoURL, appid)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, false, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("HTTP状态码错误: %d", resp.StatusCode)
	}

	var info DLCInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, false, fmt.Errorf("解析JSON失败: %v", err)
	}

	appData, ok := info.Data[appid]
	if !ok {
		return nil, false, fmt.Errorf("未找到AppID %s 的信息", appid)
	}

	// 提取所有可能的DLC ID来源
	dlcIDs := make(map[string]bool)

	// 从common.listofdlc中提取
	if listStr, ok := appData.Common["listofdlc"].(string); ok {
		re := regexp.MustCompile(`\d+`)
		matches := re.FindAllString(listStr, -1)
		for _, id := range matches {
			dlcIDs[id] = true
		}
	}

	// 从extended.listofdlc中提取
	if listStr, ok := appData.Extended["listofdlc"].(string); ok {
		re := regexp.MustCompile(`\d+`)
		matches := re.FindAllString(listStr, -1)
		for _, id := range matches {
			dlcIDs[id] = true
		}
	}

	// 从depots.dlc列表中提取
	if appData.Depots != nil {
		if depotsMap, ok := appData.Depots.(map[string]interface{}); ok {
			if dlcMap, ok := depotsMap["dlc"]; ok {
				// dlcMap 可能是 map 也可能是其他类型
				switch v := dlcMap.(type) {
				case map[string]interface{}:
					// 遍历这个map的键
					for dlcID := range v {
						dlcIDs[dlcID] = true
					}
				case string:
					// 如果是字符串, 跳过或者记录日志
					fmt.Printf("警告: DLC 字段是字符串: %s\n", v)
				default:
					fmt.Printf("警告: DLC 字段的类型异常: %T\n", v)
				}
			}
		} else {
			fmt.Printf("Warning: depots is not a map: %T\n", appData.Depots)
		}
	}

	// 从dlc字典中提取
	for id := range appData.DLC {
		dlcIDs[id] = true
	}

	// 转换为切片并排序
	dlcIDsSlice := make([]string, 0, len(dlcIDs))
	for id := range dlcIDs {
		dlcIDsSlice = append(dlcIDsSlice, id)
	}
	sort.Slice(dlcIDsSlice, func(i, j int) bool {
		a, _ := strconv.Atoi(dlcIDsSlice[i])
		b, _ := strconv.Atoi(dlcIDsSlice[j])
		return a < b
	})

	// 检查是否有仓库
	hasDepots := false
	if depots, ok := appData.Depots.(map[string]interface{}); ok && len(depots) > 0 {
		hasDepots = true
	} else if _, ok := appData.Depots.(string); ok {
		// 字符串类型的depots也算有仓库
		hasDepots = true
	}

	return dlcIDsSlice, hasDepots, nil
}
