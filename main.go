package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 配置结构体
type Config struct {
	DownloadPath string
}

// 输出分割
var Division = strings.Repeat("=", 40)

// DLC信息API
const DLCInfoURL = "https://api.steamcmd.net/v1/info/%s"

// DLC信息结构
type DLCInfo struct {
	Data map[string]struct {
		Common   map[string]interface{} `json:"common"`
		Extended map[string]interface{} `json:"extended"`
		Depots   interface{}            `json:"depots"`
		DLC      map[string]interface{} `json:"dlc"`
	} `json:"data"`
}

// 下载源
var Sources = []string{
	"https://raw.githubusercontent.com/SteamAutoCracks/ManifestHub/%s/%s.lua", // 原始源
	"https://cdn.jsdelivr.net/gh/SteamAutoCracks/ManifestHub@%s/%s.lua",       // jsDelivr CDN
	"https://gcore.jsdelivr.net/gh/SteamAutoCracks/ManifestHub@%s/%s.lua",     // G-Core CDN
	"https://fastly.jsdelivr.net/gh/SteamAutoCracks/ManifestHub@%s/%s.lua",    // Fastly CDN
}

// DepotKeys 镜像源
var DepotkeySources = []string{
	"https://raw.githubusercontent.com/SteamAutoCracks/ManifestHub/main/depotkeys.json",
	"https://cdn.jsdmirror.com/gh/SteamAutoCracks/ManifestHub@main/depotkeys.json",
	"https://raw.gitmirror.com/SteamAutoCracks/ManifestHub/main/depotkeys.json",
	"https://raw.dgithub.xyz/SteamAutoCracks/ManifestHub/main/depotkeys.json",
	"https://gh.akass.cn/SteamAutoCracks/ManifestHub/main/depotkeys.json",
}

// HTTP客户端
var httpClient = &http.Client{
	Timeout: 3 * time.Second, // 3秒超时
}

// 加载配置
func LoadConfig() *Config {
	config := &Config{
		DownloadPath: ".", // 默认当前目录
	}

	// 检查配置文件是否存在
	configFile := "config.ini"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// 配置文件不存在, 创建默认配置
		if err := CreateConfig(); err != nil {
			fmt.Printf("创建配置文件失败: %v, 使用默认路径\n", err)
			return config
		}
		fmt.Println("已创建默认配置文件: config.ini")
	}

	// 尝试读取配置文件
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("读取配置文件失败: %v, 使用默认路径\n", err)
		return config
	}

	// 解析配置文件
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			// 跳过注释行和空行
			continue
		}

		if strings.HasPrefix(line, "downloadPath") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				path := strings.TrimSpace(parts[1])
				// 移除可能存在的引号
				path = strings.Trim(path, `"'`)

				// 检查路径是否有效
				if path != "" {
					// 转换为绝对路径
					absPath, err := filepath.Abs(path)
					if err != nil {
						fmt.Printf("路径转换失败: %v, 使用原路径\n", err)
						config.DownloadPath = path
					} else {
						config.DownloadPath = absPath
					}
				}
				break
			}
		}
	}

	fmt.Printf("下载路径: %s\n", config.DownloadPath)
	return config
}

// 创建默认配置文件
func CreateConfig() error {
	// 默认配置内容
	content := `downloadPath = "."`

	// 写入配置文件
	if err := os.WriteFile("config.ini", []byte(content), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	return nil
}

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

// 下载depotkeys.json
func DownloadDepotkeys() (map[string]string, error) {
	var lastError error
	totalSources := len(DepotkeySources)

	// 尝试每个下载源
	for i, source := range DepotkeySources {
		fmt.Printf("尝试 DepotKey 源 #%d: %s\n", i+1, source)

		// 创建请求
		req, err := http.NewRequest("GET", source, nil)
		if err != nil {
			lastError = err
			continue
		}

		// 添加超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		// 执行请求
		resp, err := httpClient.Do(req)
		if err != nil {
			lastError = fmt.Errorf("DepotKey 源 #%d 失败: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

		// 检查状态码
		if resp.StatusCode != http.StatusOK {
			lastError = fmt.Errorf("DepotKey 源 #%d 状态码 %d", i+1, resp.StatusCode)
			continue
		}

		// 读取数据
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastError = fmt.Errorf("DepotKey 源 #%d 读取失败: %v", i+1, err)
			continue
		}

		// 解析JSON
		depotkeys := make(map[string]string)
		if err := json.Unmarshal(data, &depotkeys); err != nil {
			lastError = fmt.Errorf("DepotKey 源 #%d 解析失败: %v", i+1, err)
			continue
		}

		// 下载完成返回
		fmt.Printf("成功从源 #%d 下载 depotkeys.json (%d个条目)\n", i+1, len(depotkeys))
		fmt.Println(Division)
		return depotkeys, nil
	}

	// 所有源都失败
	return nil, fmt.Errorf("所有 %d 个 DepotKey 源尝试失败: %v", totalSources, lastError)
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

// 多源下载
func TrySources(APPID string) ([]byte, error) {
	var lastError error
	totalSources := len(Sources)

	// 尝试每个下载源
	for i, source := range Sources {
		url := fmt.Sprintf(source, APPID, APPID)
		fmt.Printf("尝试源 #%d: %s\n", i+1, url)

		// 创建请求
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastError = err
			continue
		}

		// 添加超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		// 执行请求
		resp, err := httpClient.Do(req)
		if err != nil {
			lastError = fmt.Errorf("源 #%d 失败: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

		// 检查状态码
		if resp.StatusCode != http.StatusOK {
			lastError = fmt.Errorf("源 #%d 状态码 %d", i+1, resp.StatusCode)
			continue
		}

		// 读取数据
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastError = fmt.Errorf("源 #%d 读取失败: %v", i+1, err)
			continue
		}

		// 下载完成返回
		fmt.Printf("成功从源 #%d 下载\n", i+1)
		fmt.Println(Division)
		return data, nil
	}

	// 所有源都失败
	return nil, fmt.Errorf("所有 %d 个源尝试失败: %v", totalSources, lastError)
}

// 下载函数
func Download(APPID string, downloadPath string) error {
	// 尝试从多个源下载
	data, err := TrySources(APPID)
	if err != nil {
		return err
	}

	// 处理文件
	modifiedData := ProcessFile(data)

	// 下载 DepotKeys
	depotkeys, err := DownloadDepotkeys()
	if err != nil {
		fmt.Printf("下载 DepotKeys 失败: %v\n", err)
	} else {
		// 修补 DepotKey
		modifiedData = PatchDepotkey(APPID, modifiedData, depotkeys)
	}

	// 保存文件
	filename := APPID + ".lua"
	fullPath := filepath.Join(downloadPath, filename)

	// 使用配置的下载路径保存
	if err := SaveFile(downloadPath, filename, modifiedData); err != nil {
		return err
	}

	// 下载完成后添加DLC
	fmt.Println(Division)
	fmt.Println("开始添加无仓库的DLC...")
	if err := AddDLC(APPID, fullPath); err != nil {
		fmt.Printf("添加DLC失败: %v\n", err)
	} else {
		fmt.Println("DLC添加完成")
	}
	return nil
}

// 主程序
func main() {
	// 输出
	fmt.Println(Division)
	fmt.Println("ManifestHub CLI - 新一代密钥获取工具")
	fmt.Println(Division)
	fmt.Println("开发者:LANREN")
	fmt.Println("版本号:V1.1")

	// 加载配置
	config := LoadConfig()

	for {
		// 定义变量
		var UserAPPID string

		// 输出输入
		fmt.Println(Division)
		fmt.Print("请输入AppID:")
		fmt.Scanln(&UserAPPID)

		// 显示下载信息
		fmt.Println(Division)
		fmt.Printf("开始下载: %s.lua\n", UserAPPID)
		fmt.Println("尝试以下下载源:")
		for i, source := range Sources {
			fmt.Printf(" %d. %s\n", i+1, fmt.Sprintf(source, UserAPPID, UserAPPID))
		}
		fmt.Println(Division)

		// 调用下载函数
		startTime := time.Now()
		if err := Download(UserAPPID, config.DownloadPath); err != nil {
			fmt.Printf("下载失败: %v\n", err)
		}

		fmt.Println(Division)
		fmt.Printf("耗时: %.2f秒\n", time.Since(startTime).Seconds())
	}
}
