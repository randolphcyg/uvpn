/*
支持 OpenVPN 虚拟IP池的动态给定：但因 subnet 拓扑要求, OpenVPN 的虚拟IP池只能是 255.255.0.0(/16) 或更高;
支持 CIDR 与 uint64 整数的互相转换,以便从缓存中读取大整数并将其转换为当前可分配虚拟IP地址;
*/
package ovpn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// OpenVPN 用户虚拟IP网段CIDR表示形式
// 该网段最大与最小可用虚拟IP 默认最小IP留给VPN服务端
var ovpnVipPool = "10.11.0.0/16"
var ovpnVipPoolCidr *net.IPNet

// 虚拟IP池网段最小/最大IP 最小/最大可分配虚拟IP
var minIp, maxIp, minVip, maxVip string
var minIpNum, maxIpNum, minVipNum, maxVipNum, ipLen, vipLen uint32

func init() {
	_, ovpnVipPoolCidr, _ = net.ParseCIDR(ovpnVipPool)
	maskLen, _ := ovpnVipPoolCidr.Mask.Size()
	if maskLen < 16 {
		fmt.Println(errors.New("因 subnet 拓扑要求, OpenVPN 的虚拟IP池只能是 255.255.0.0(/16) 或更高;"))
	}
	// 虚拟IP池网段最大最小IP
	minIp = AssignVip(ovpnVipPoolCidr, 0)
	minIpNum, _ = VipToNum(minIp)
	ipLen = 2 << (31 - maskLen)
	maxIp = AssignVip(ovpnVipPoolCidr, ipLen)
	maxIpNum, _ = VipToNum(maxIp)
	// 虚拟IP池网段最大最小可用虚拟IP
	minVip = AssignVip(ovpnVipPoolCidr, 2)
	minVipNum, _ = VipToNum(minVip)
	vipLen = ipLen - 3
	maxVip = AssignVip(ovpnVipPoolCidr, vipLen)
	maxVipNum, _ = VipToNum(maxVip)
}

// VipToNum 虚拟IP转换为整数
func VipToNum(vip string) (num uint32, err error) {
	ip := net.ParseIP(vip)
	if !ovpnVipPoolCidr.Contains(ip) {
		return 0, errors.New("该IP不在ovpn虚拟IP池内！")
	}
	num = binary.BigEndian.Uint32(ip.To4())
	num -= minIpNum
	return
}

// NumToVip 整数转换为虚拟IP
func NumToVip(num uint32) (vip string, err error) {
	if num < 2 || num > vipLen {
		return "", errors.New("该整数对应IP不在用户可分配虚拟IP范围!")
	}
	vip = AssignVip(ovpnVipPoolCidr, num)
	return
}

// AssignVip 分配虚拟IP计算方法
func AssignVip(cidr *net.IPNet, num uint32) string {
	ip := make(net.IP, len(cidr.IP.To4()))
	last := binary.BigEndian.Uint32(cidr.IP.To4()) | num
	binary.BigEndian.PutUint32(ip, last)
	return ip.String()
}
