package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

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

	// 所有常规源都失败, 尝试zip源
	fmt.Printf("所有 %d 个源尝试失败: %v\n", totalSources, lastError)
	fmt.Printf("正在尝试 Walftech 源: %s\n", fmt.Sprintf(zipSource, APPID))
	return tryZipSource(APPID)
}

// 尝试从zip源下载
func tryZipSource(APPID string) ([]byte, error) {
	zipURL := fmt.Sprintf(zipSource, APPID)
	expectedFileName := APPID + ".lua"
	maxRetries := 2 // 最多重试2次

	// 增加重试机制, 应对临时网络波动
	for retry := 0; retry <= maxRetries; retry++ {
		if retry > 0 {
			fmt.Printf("第 %d 次重试 Walftech 源...\n", retry)
		}

		// 先尝试 HEAD 获取 Content-Length，以便计算合适的超时并展示进度
		var contentLen int64 = 0
		headReq, herr := http.NewRequest("HEAD", zipURL, nil)
		if herr == nil {
			hctx, hcancel := context.WithTimeout(context.Background(), 5*time.Second)
			headReq = headReq.WithContext(hctx)
			hresp, herr2 := http.DefaultClient.Do(headReq)
			if herr2 == nil && hresp != nil {
				if hresp.StatusCode == http.StatusOK {
					if cl := hresp.Header.Get("Content-Length"); cl != "" {
						if v, err := strconv.ParseInt(cl, 10, 64); err == nil {
							contentLen = v
						}
					}
				}
				hresp.Body.Close()
			}
			hcancel()
		}

		// 根据 Content-Length 自动计算超时（最低30s），若无长度则使用较保守的默认超时
		var timeout time.Duration
		if contentLen > 0 {
			// 假设最低下载速度 32KB/s，超时时间 = 文件大小 / 32KB + 10s buffer
			estSec := contentLen / 32768
			timeout = time.Duration(estSec)*time.Second + 10*time.Second
			if timeout < 30*time.Second {
				timeout = 30 * time.Second
			}
		} else {
			// 未知大小时使用较长的默认超时（例如 90s），可覆盖重试逻辑
			timeout = 90 * time.Second
		}

		// 创建 GET 请求并使用可取消的上下文（不使用整体固定 deadline）
		req, err := http.NewRequest("GET", zipURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("请求失败: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			cancel()
			return nil, fmt.Errorf("状态码错误 %d(URL: %s)", resp.StatusCode, zipURL)
		}

		// 使用分块读取并显示进度（若有 Content-Length 则显示百分比），同时把数据写入内存缓冲
		start := time.Now()
		var buf bytes.Buffer
		const chunkSize = 32 * 1024
		tmp := make([]byte, chunkSize)
		var total int64 = 0
		// 自适应“空闲超时”监测：当短时间内无进展（无字节读取）会取消请求并重试
		lastProgress := time.Now()
		var idleTimeout time.Duration
		if contentLen > 0 {
			// 假设最低持续速度 8KB/s，空闲超时 = 文件大小/8KB + 60s buffer，最小60s
			estSec := contentLen / (8 * 1024)
			idleTimeout = time.Duration(estSec)*time.Second + 60*time.Second
			if idleTimeout < 60*time.Second {
				idleTimeout = 60 * time.Second
			}
		} else {
			idleTimeout = 120 * time.Second
		}

		doneCh := make(chan struct{})
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if time.Since(lastProgress) > idleTimeout {
						fmt.Printf("\n检测到长时间无进展(%v)，取消下载并重试...\n", idleTimeout)
						cancel()
						return
					}
				case <-ctx.Done():
					return
				case <-doneCh:
					return
				}
			}
		}()
		readErr := error(nil)
		for {
			n, rerr := resp.Body.Read(tmp)
			if n > 0 {
				if _, werr := buf.Write(tmp[:n]); werr != nil {
					resp.Body.Close()
					close(doneCh)
					cancel()
					return nil, fmt.Errorf("写入缓冲失败: %v", werr)
				}
				total += int64(n)
				// 更新最后进度时间，避免被 watchdog 误判为停止
				lastProgress = time.Now()

				// 计算并打印进度/速度（KB/s）
				elapsed := time.Since(start)
				if elapsed <= 0 {
					elapsed = 1 * time.Millisecond
				}
				speedKB := float64(total) / 1024.0 / elapsed.Seconds()
				if contentLen > 0 {
					pct := float64(total) / float64(contentLen) * 100.0
					fmt.Printf("\r下载中: %.2f%% (%d/%d bytes)  %.2f KB/s", pct, total, contentLen, speedKB)
				} else {
					fmt.Printf("\r下载中: %d bytes  %.2f KB/s", total, speedKB)
				}
			}
			if rerr == io.EOF {
				// 正常结束
				break
			}
			if rerr != nil {
				// 非 EOF 错误（例如超时），记录并中断读取，后续重试时丢弃部分数据
				readErr = rerr
				break
			}
		}
		resp.Body.Close()
		cancel()
		fmt.Println()

		// 如果读取过程中发生非 EOF 错误，则丢弃本次部分数据并重试
		if readErr != nil {
			resp.Body.Close()
			cancel()
			fmt.Printf("\n读取ZIP数据失败: %v, 将重试...\n", readErr)
			continue
		}

		// 如果没有读取到数据则继续重试
		if buf.Len() == 0 {
			continue
		}

		zipData := buf.Bytes()
		// 简单校验 ZIP magic header（以避免解析 HTML/错误页面）
		if len(zipData) < 4 || !bytes.HasPrefix(zipData, []byte("PK")) {
			return nil, fmt.Errorf("解析ZIP失败: 返回内容不是有效的ZIP文件(URL: %s)", zipURL)
		}

		// 解析ZIP
		zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
		if err != nil {
			return nil, fmt.Errorf("解析ZIP失败: %v", err)
		}

		// 查找目标文件(支持嵌套目录, 只匹配文件名)
		for _, file := range zipReader.File {
			// 用Base函数忽略路径, 只比较文件名(如"subdir/123.lua"也能匹配"123.lua")
			if filepath.Base(file.Name) == expectedFileName {
				rc, err := file.Open()
				if err != nil {
					return nil, fmt.Errorf("打开ZIP内文件失败: %v", err)
				}

				// 读取文件内容(这里不需要defer, 读取后直接关闭)
				data, err := io.ReadAll(rc)
				rc.Close() // 立即关闭, 避免资源占用
				if err != nil {
					return nil, fmt.Errorf("读取ZIP内文件失败: %v", err)
				}

				fmt.Printf("成功从 Walftech 源提取文件: %s(大小: %d字节)\n", expectedFileName, len(data))
				fmt.Println(Division)
				return data, nil
			}
		}

		// 如果没找到文件, 无需重试(内容问题, 重试也没用)
		break
	}

	return nil, fmt.Errorf("ZIP中未找到目标文件: %s(URL: %s)", expectedFileName, zipURL)
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
