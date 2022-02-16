package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
)

// GetAllFile 获取目录下所有文件
func GetAllFile(path string, s []string) ([]string, error) {
	rd, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("read dir fail:", err)
		return s, err
	}

	for _, fi := range rd {
		if !fi.IsDir() {
			fullName := path + "/" + fi.Name()
			s = append(s, fullName)
		}
	}
	return s, nil
}

// ReadFile 读文件输出在终端
func ReadFile(path string) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("文件打开失败", err)
	}
	//及时关闭file句柄
	defer file.Close()
	//读原来文件的内容，并且显示在终端
	reader := bufio.NewReader(file)
	line := 0
	for {
		str, err := reader.ReadString('\n')
		line++
		fmt.Print(line, str)
		if err == io.EOF {
			break
		}
	}
}

// ExtractViPFromCCD 从ccd文件提取出用户的VIP
func ExtractViPFromCCD(path string) (vip string, err error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("文件打开失败", err)
	}
	//及时关闭file句柄
	defer file.Close()
	//读原来文件的内容，并且显示在终端
	reader := bufio.NewReader(file)
	line := 0
	for {
		str, err := reader.ReadString('\n')
		line++
		if strings.Contains(str, "ifconfig-push") {
			if str == "" {
				return "", errors.New("________ipconfpush语句错误______")
			}
			// VIP所在行用空格分割后提取
			strSplit := strings.Split(str, " ")
			res := strings.Replace(strSplit[1], " ", "", -1)
			return res, nil
		}
		if err == io.EOF {
			return "", errors.New("未找到VIP")
		}
	}
	return
}

// IsFileExist 判断文件是否存在
func IsFileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}

// AddRoute4User 向文件写入内容
func AddRoute4User(path string, content string) (err error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("文件打开失败", err)
	}
	//及时关闭file句柄
	defer file.Close()

	// 阻塞模式下，加排他锁
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		log.Info("add exclusive lock in block failed", err)
	}
	// 这里进行业务逻辑
	//写入文件时，使用带缓存的 *Writer
	write := bufio.NewWriter(file)
	write.WriteString("\n" + content + "\n")

	//Flush将缓存的文件真正写入到文件中
	write.Flush()

	// 解锁
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		log.Info("unlock exclusive lock failed", err)
	}
	return
}

func GenerateCCD(path string, content string) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("文件打开失败", err)
	}
	//及时关闭file句柄
	defer file.Close()

	// 阻塞模式下，加排他锁
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		log.Info("add exclusive lock in block failed", err)
	}
	// 这里进行业务逻辑
	//写入文件时，使用带缓存的 *Writer
	write := bufio.NewWriter(file)
	write.WriteString("\n" + content + "\n")

	//Flush将缓存的文件真正写入到文件中
	write.Flush()

	// 解锁
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		log.Info("unlock exclusive lock failed", err)
	}
	return
}

// IsInSlice 判断是否已在切片中
func IsInSlice(ele interface{}, s []interface{}) bool {
	for _, item := range s {
		if ele == item {
			return true
		}
	}
	return false
}

/*
* 计算公网IP地址，后续可以提示用户
 */

type IPInfo struct {
	Code int `json:"code"`
	Data IP  `json:"data`
}

type IP struct {
	Country   string `json:"country"`
	CountryId string `json:"country_id"`
	Area      string `json:"area"`
	AreaId    string `json:"area_id"`
	Region    string `json:"region"`
	RegionId  string `json:"region_id"`
	City      string `json:"city"`
	CityId    string `json:"city_id"`
	Isp       string `json:"isp"`
}

func TabaoAPI(ip string) *IPInfo {
	url := "http://ip.taobao.com/service/getIpInfo.php?ip="
	url += ip

	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var result IPInfo
	if err := json.Unmarshal(out, &result); err != nil {
		return nil
	}

	return &result
}

func Ipv4MaskString(m []byte) string {
	if len(m) != 4 {
		panic("ipv4Mask: len must be 4 bytes")
	}
	return fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
}

// ResolveIP 将域名解析成IP
func ResolveIP(domain string) (ip string, err error) {
	addr, err := net.ResolveIPAddr("ip", domain)
	if err != nil {
		return
	}
	ip = addr.String()
	return
}

// IsPublicIP 判断是否为公网IP
func IsPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := IP.To4(); ip4 != nil {
		switch true {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}

// GetLanIp 获取执行此代码的服务的局域网IP
func GetLanIp() (string, error) {
	conn, err := net.Dial("udp", "google.com:80")
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	defer conn.Close()
	return strings.Split(conn.LocalAddr().String(), ":")[0], nil
}
