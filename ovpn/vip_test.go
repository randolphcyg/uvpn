package ovpn

import (
	"fmt"
	"testing"
)

// 功能测试 ip转整数
func TestVipToNum(t *testing.T) {
	fmt.Println("========[功能测试]========")
	num, err := VipToNum("10.121.3.155")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("IP转换为整数:%v\n", num)
}

// 功能测试 整数转ip
func TestNumToVip(t *testing.T) {
	ip, err := NumToVip(932)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("整数转换为IP:%v\n", ip)
	fmt.Println("========[功能测试]========")
}

// 边界测试 ip转整数
func TestVipToNumBoundary(t *testing.T) {
	fmt.Println("========[边界测试]========")
	var vips = [...]string{"10.121.0.0", "10.120.255.255", "10.121.255.253", "10.121.255.254", "10.121.255.255", "10.122.0.0"}
	for index, vip := range vips {
		num, err := VipToNum(vip)
		fmt.Printf("%v.IP %v 转换为整数:%v\n", index, vip, num)
		if err != nil {
			fmt.Println(err)
		}

	}

}

func TestNumToVipBoundary(t *testing.T) {
	var nums = [...]uint32{0, 1, 158, 567, 3005, 65533, 65534, 65535, 65536, 789789}
	for index, num := range nums {
		ip, err := NumToVip(num)
		fmt.Printf("%v.整数 %v 转换为IP:%v\n", index, num, ip)
		if err != nil {
			fmt.Println(err)
		}
	}
	fmt.Println("========[边界测试]========")
}

// 功能测试 ip转整数
func TestTemp(t *testing.T) {
	fmt.Println("！！！！！！！！！！！！！！！！！")
	num, err := VipToNum("10.11.3.164")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("IP转换为整数:%v\n", num)
}
