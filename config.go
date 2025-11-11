package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
