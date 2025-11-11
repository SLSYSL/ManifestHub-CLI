package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"
)

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
	fmt.Println("版本号:V1.2")

	// 加载配置
	config := LoadConfig()

	for {
		// 输出输入
		OriginUserAPPID, err := GetAppID()
		if err != nil {
			// 如果是 EOF（比如输入流关闭或用户退出），优雅退出程序
			if err == io.EOF {
				fmt.Println("\n输入已关闭，程序退出")
				return
			}
			fmt.Printf("获取AppID失败: %v\n", err)
			continue
		}

		// 显示下载信息
		UserAPPID := strconv.Itoa(OriginUserAPPID)
		fmt.Println(Division)
		fmt.Printf("开始下载: %s.lua\n", UserAPPID)
		fmt.Println("尝试以下下载源:")
		for i, source := range Sources {
			fmt.Printf(" %d. %s\n", i+1, fmt.Sprintf(source, UserAPPID, UserAPPID))
		}
		fmt.Printf(" 5. %s\n", fmt.Sprintf(zipSource, UserAPPID))
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
