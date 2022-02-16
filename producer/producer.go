package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	log "github.com/sirupsen/logrus"
	"mq/utils"
	"net"
	"os"
	"strings"
)

var (
	nameSrvAddr = "192.168.5.119"
	nameSrvPort = "9876"
)

type UVPNAuthority struct {
	SpName      string       `mapstructure:"spName"`
	Userid      string       `mapstructure:"userid"`
	Eid         string       `mapstructure:"工号"`
	DisplayName string       `mapstructure:"姓名"`
	UVPNDestIps []UVPNDestIp `mapstructure:"UVPN权限"`
}

// UVPNDestIp UVPN目标权限
type UVPNDestIp struct {
	DestIp string `mapstructure:"目标IP"`
}

// RemoveRepeatedElement 通过map键的唯一性去重
func RemoveRepeatedElement(s []UVPNDestIp) []UVPNDestIp {
	result := make([]UVPNDestIp, 0)
	m := make(map[UVPNDestIp]bool) //map的值不重要
	for _, v := range s {
		if _, ok := m[v]; !ok {
			result = append(result, v)
			m[v] = true
		}
	}
	return result
}

func main() {
	var order = UVPNAuthority{
		SpName:      "UVPN权限",
		Userid:      "1987",
		Eid:         "1987",
		DisplayName: "王二小",
		UVPNDestIps: []UVPNDestIp{
			{DestIp: "10.16.3.0/24"},
			{DestIp: "192.168.5.9"},
			{DestIp: "192.168.5.9"},
			{DestIp: "192.168.5.8"},
			{DestIp: "122.112.146.97"},
			{DestIp: "122.112.146.97"},
			{DestIp: "12.4.3"}, // 这个填写少了 但是错误没判断出来 转换成了12.4.0.3
		},
	}

	// 复制工单
	tmpOrder := order
	tmpOrder.UVPNDestIps = []UVPNDestIp{}

	for _, item := range order.UVPNDestIps {
		ip := net.ParseIP(item.DestIp)
		if ip == nil { // 如果不是IP地址，判断是否是域名
			dnsIp, err := utils.ResolveIP(item.DestIp) // 将域名解析
			if err != nil {
				if dnsIp == "" && !strings.Contains(item.DestIp, "/") {
					log.Warning("[域名解析不出来，告知用户错误了]", item.DestIp)
				} else {
					cidr, ci, _ := net.ParseCIDR(item.DestIp)
					log.Info("[CIDR]", cidr, ci)
					tmpOrder.UVPNDestIps = append(tmpOrder.UVPNDestIps, UVPNDestIp{ci.String()})
				}
			} else {
				log.Info("[域名IP]", dnsIp)
				tmpOrder.UVPNDestIps = append(tmpOrder.UVPNDestIps, UVPNDestIp{dnsIp})
			}
		} else {
			if utils.IsPublicIP(ip) {
				log.Warning("[公网IP] 企业微信告知用户，只处理内网地址", ip)
			} else {
				log.Info("[正常IP]", ip)
				tmpOrder.UVPNDestIps = append(tmpOrder.UVPNDestIps, UVPNDestIp{ip.String()})
			}
		}
	}

	tmpOrder.UVPNDestIps = RemoveRepeatedElement(tmpOrder.UVPNDestIps)

	p, _ := rocketmq.NewProducer(
		producer.WithNsResolver(primitive.NewPassthroughResolver([]string{nameSrvAddr + ":" + nameSrvPort})),
		//指定重试次数
		producer.WithRetry(2),
	)
	// 启动producer
	err := p.Start()
	if err != nil {
		fmt.Printf("start producer error: %s", err.Error())
		os.Exit(1)
	}
	topic := "UVPN"
	data, err := json.Marshal(tmpOrder)
	msg := &primitive.Message{
		Topic: topic,
		Body:  data,
	}
	//指定标签
	msg.WithTag("UVPN")
	msg.WithKeys([]string{"UVPN", "权限"})
	msg.WithProperty("属性", "权限")
	//参数是延时级别，共有16个级别 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h
	msg.WithDelayTimeLevel(3)

	res, err := p.SendSync(context.Background(), msg)

	if err != nil {
		fmt.Printf("send message error: %s\n", err)
	} else {
		fmt.Printf("send message success: result=%s\n", res.String())
		fmt.Println(res.MsgID)
	}
	//关闭
	err = p.Shutdown()
	if err != nil {
		fmt.Printf("shutdown producer error: %s", err.Error())
	}
}
