package main

import (
	"net/http"
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

// 解析 SteamUI 的响应结构
type LoadGamesResponse struct {
	Games []Game `json:"games"` // API返回的游戏列表字段
}

// 单个游戏的核心信息结构体
type Game struct {
	AppID int    `json:"appid"`
	Name  string `json:"name"`
}

// 下载源
var Sources = []string{
	"https://raw.githubusercontent.com/SteamAutoCracks/ManifestHub/%s/%s.lua", // 原始源
	"https://cdn.jsdelivr.net/gh/SteamAutoCracks/ManifestHub@%s/%s.lua",       // jsDelivr CDN
	"https://gcore.jsdelivr.net/gh/SteamAutoCracks/ManifestHub@%s/%s.lua",     // G-Core CDN
	"https://fastly.jsdelivr.net/gh/SteamAutoCracks/ManifestHub@%s/%s.lua",    // Fastly CDN
}

// 新增zip源
const zipSource = "https://walftech.com/proxy.php?url=https://steamgames554.s3.us-east-1.amazonaws.com/%s.zip"

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
	Timeout: 5 * time.Second, // 5秒超时
}
