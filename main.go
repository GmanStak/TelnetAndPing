package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Telnet 超时时间
const TELNET_TIMEOUT = 3 * time.Second

// Ping 超时时间
const PING_TIMEOUT = 3 * time.Second

// 页面模板
var tmpl = template.Must(template.ParseFiles("index.html"))

// 检测结果结构体
type ScanResult struct {
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	IsOpen bool   `json:"isOpen"`
	URI    string `json:"uri"`
}

type PingResult struct {
	IP          string `json:"ip"`
	IsReachable bool   `json:"isReachable"`
}

func main() {
	// 提供静态文件服务
	fs := http.FileServer(http.Dir("./"))
	http.Handle("/", fs)
	http.HandleFunc("/scan", scanHandler)
	http.HandleFunc("/scanPing", scanPingHandler)

	fmt.Println("服务器已启动，访问 http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

// 检测处理器
func scanHandler(w http.ResponseWriter, r *http.Request) {
	ipRange := r.URL.Query().Get("ipRange")
	portStr := r.URL.Query().Get("port")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "无效的端口号", http.StatusBadRequest)
		return
	}

	// 生成 IP 地址范围
	ipRangeStart, ipRangeEnd, err := parseIPRange(ipRange)
	if err != nil {
		http.Error(w, "无效的 IP 地址段格式", http.StatusBadRequest)
		return
	}
	ipRangeList := generateIPRange(ipRangeStart, ipRangeEnd)

	var results []ScanResult
	var wg sync.WaitGroup

	// 使用通道限制并发数量
	semaphore := make(chan struct{}, 100) // 允许同时运行 100 个 goroutine

	for _, ip := range ipRangeList {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			// 控制并发
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			isOpen := checkPort(ip, port)
			var result ScanResult
			result.IP = ip
			result.Port = port
			result.IsOpen = isOpen

			if isOpen {
				result.URI = fmt.Sprintf("http://%s:%d/metrics", ip, port)
			}

			results = append(results, result)
		}(ip)
	}

	wg.Wait()

	// 按 IP 地址排序结果
	sortResults(&results)

	// 发送 JSON 响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Ping检测处理器
func scanPingHandler(w http.ResponseWriter, r *http.Request) {
	ipRange := r.URL.Query().Get("ipRange")

	// 生成 IP 地址范围
	ipRangeStart, ipRangeEnd, err := parseIPRange(ipRange)
	if err != nil {
		http.Error(w, "无效的 IP 地址段格式", http.StatusBadRequest)
		return
	}
	ipRangeList := generateIPRange(ipRangeStart, ipRangeEnd)

	var results []PingResult
	var wg sync.WaitGroup

	// 使用通道限制并发数量
	semaphore := make(chan struct{}, 100) // 允许同时运行 100 个 goroutine

	for _, ip := range ipRangeList {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			// 控制并发
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			isReachable := checkPing(ip)
			var result PingResult
			result.IP = ip
			result.IsReachable = isReachable

			results = append(results, result)
		}(ip)
	}

	wg.Wait()

	// 按 IP 地址排序结果
	sortResultsPing(&results)

	// 发送 JSON 响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// 检查端口是否开放
func checkPort(ip string, port int) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, TELNET_TIMEOUT)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// 检查是否可以 Ping 通
func checkPing(ip string) bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows系统下使用"-n"参数
		cmd = exec.Command("ping", "-n", "1", "-w", "1000", ip)
	} else {
		// 类Unix系统下使用"-c"和"-W"参数
		cmd = exec.Command("ping", "-c", "1", "-W", "1", ip)
	}
	err := cmd.Run()
	return err == nil
}

// 生成 IP 地址范围
func generateIPRange(startIP, endIP string) []string {
	var ips []string

	start := ipToLong(startIP)
	end := ipToLong(endIP)

	for i := start; i <= end; i++ {
		ips = append(ips, longToIP(uint32(i)))
	}

	return ips
}

// 将 IP 转换为长整型
func ipToLong(ip string) uint32 {
	parts := strings.Split(ip, ".")
	var long uint32 = 0
	for _, part := range parts {
		long = (long << 8) + uint32(toInt(part))
	}
	return long
}

// 将长整型转换为 IP
func longToIP(long uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		(long>>24)&0xFF,
		(long>>16)&0xFF,
		(long>>8)&0xFF,
		long&0xFF)
}

// 字符串转整数
func toInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// 解析 IP 地址段
func parseIPRange(ipRange string) (string, string, error) {
	parts := strings.Split(ipRange, "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("无效的 IP 地址段格式")
	}
	return parts[0], parts[1], nil
}

// 按 IP 地址排序结果
func sortResults(results *[]ScanResult) {
	sort.Slice(*results, func(i, j int) bool {
		ip1 := (*results)[i].IP
		ip2 := (*results)[j].IP
		return ip1 < ip2
	})
}

// 按 IP 地址排序 Ping 结果
func sortResultsPing(results *[]PingResult) {
	sort.Slice(*results, func(i, j int) bool {
		ip1 := (*results)[i].IP
		ip2 := (*results)[j].IP
		return ip1 < ip2
	})
}
