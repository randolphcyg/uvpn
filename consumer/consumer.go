package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/go-ldap/ldap/v3"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"mq/cache"
	"mq/conf"
	"mq/logger"
	"mq/ovpn"
	"mq/utils"
	"mq/uuap"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	RouterTmp = `push "route %s %s"` // ccd添加路由规则模版
	lock      sync.Mutex
)

const (
	InfoGenerateCCDFile4User = "无此用户ccd文件，为用户创建ccd文件并分配初始权限"
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

func Consumer() {
	c, _ := rocketmq.NewPushConsumer(
		consumer.WithGroupName("uvpn"),
		consumer.WithNsResolver(primitive.NewPassthroughResolver([]string{conf.Conf.RocketMQ.Addr + ":" + conf.Conf.RocketMQ.Port})),
	)

	// 设置订阅消息的tag
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: conf.Conf.RocketMQ.TopicName, // 根据topic名称定义筛选表达式
	}

	err := c.Subscribe(conf.Conf.RocketMQ.TopicName, selector, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for i := range msgs {
			err := HandleUVPN(msgs[i])
			if err != nil {
				log.Error(err)
			}

		}

		return consumer.ConsumeSuccess, nil
	})
	if err != nil {
		log.Info(err.Error())
	}

	//开始消费
	err = c.Start()
	if err != nil {
		log.Info(err.Error())
		os.Exit(-1)

	}

	// 同步阻塞
	chWait := make(chan struct{})
	<-chWait

	err = c.Shutdown()
	if err != nil {
		log.Info("shutdown Consumer error: %s", err.Error())
	}
}

// HandleUVPN 处理权限文件
func HandleUVPN(msg *primitive.MessageExt) (err error) {
	var order UVPNAuthority
	json.Unmarshal(msg.Body, &order)

	log.Info(fmt.Sprintf("[1]MQ消息: 主题[%s] 工单名[%s] 消息Id[%s] OffsetMsgId[%s] 存储时间[%s]",
		msg.Topic, order.SpName, msg.MsgId, msg.OffsetMsgId,
		time.Unix(msg.StoreTimestamp/1000, 0).Format("2006-01-02 15:04:05")))
	fmt.Println("################################")
	// 查询LDAP用户，如果有这个人，则取其sam名称
	res, err := uuap.FetchUser(&uuap.LdapConns, &uuap.LdapAttributes{
		Num:         order.Eid,
		DisplayName: order.DisplayName,
	})
	if err != nil {
		return
	}
	// 如果查不到人
	if res == nil {
		log.Error("查无此人！", &uuap.LdapAttributes{
			Num:         order.Eid,
			DisplayName: order.DisplayName,
		})
		return errors.New("查无此人！")
	}

	// 如果ldap用户存在 但ccd文件不存在，则到redis取最新的VIP
	var ccdPath string
	if conf.Conf.System.Dev {
		ccdPath = conf.Conf.System.DevCCDFilePath
	} else {
		ccdPath = conf.Conf.System.CCDFilePath
	}
	isUserCCDFileExist := utils.IsFileExist(ccdPath + "/" + res.GetAttributeValue("sAMAccountName"))
	// 如果发现ccd文件不存在，则新建ccd文件并写入基础权限 加锁
	if !isUserCCDFileExist {
		err = GenerateCCD4User(ccdPath + "/" + res.GetAttributeValue("sAMAccountName"))
		if err != nil {
			return
		}
		log.Info(InfoGenerateCCDFile4User)
	}

	// 将取到的LDAP名称同名的ovpn的ccd文件做修改
	content := ""
	for idx, cidr := range order.UVPNDestIps {
		if idx == len(order.UVPNDestIps)-1 {
			content += CIDR2OVPNRouterClause(cidr.DestIp)
		} else {
			content += CIDR2OVPNRouterClause(cidr.DestIp) + "\n"
		}

	}
	// 将权限更新到配置文件
	err = utils.AddRoute4User(ccdPath+"/"+res.GetAttributeValue("sAMAccountName"), content)
	if err != nil {
		return
	} else {
		log.Info(fmt.Sprintf("[2]用户[%s] 账号[%s]", res.GetAttributeValue("displayName"), res.GetAttributeValue("sAMAccountName")))
		log.Info(fmt.Sprintf("[3]详细新增路由: %s", content))
	}
	return
}

// GenerateCCD4User 为用户生成ccd文件
func GenerateCCD4User(ccdFilePath string) (err error) {
	// 获取ovpn当前可分配vip对应的int值
	lock.Lock()
	currentVip, err := cache.Get("OVPNVIP")
	if err != nil {
		lock.Unlock()
		return
	}
	lock.Unlock()

	temp, err := cache.Get("OVPNTEMP")
	if err != nil {
		return
	}

	num, err := strconv.ParseUint(currentVip, 10, 32)
	if err != nil {
		return
	}

	vip, err := ovpn.NumToVip(uint32(num))
	if err != nil {
		return
	}

	err = utils.GenerateCCD(ccdFilePath, fmt.Sprintf(temp, vip))
	if err != nil {
		return
	}

	// 成功生成ccd文件后，刷新缓存中的当前vip
	err = AddVip(currentVip)
	if err != nil { // 如果更新缓存的vip失败，则删除ccd文件
		err = os.Remove(ccdFilePath)
		return
	}
	return
}

// AddVip 为用户生成ccd文件成功后，缓存中的vip加1
func AddVip(oldVipStr string) (err error) {
	oldVip, err := strconv.Atoi(oldVipStr)
	if err != nil {
		return err
	}
	lock.Lock()
	err = cache.Set("OVPNVIP", strconv.Itoa(oldVip+1))
	if err != nil {
		lock.Unlock()
		return
	}
	lock.Unlock()
	return
}

// CIDR2OVPNRouterClause CIDR2OVPNRouter 将mq中的地址转换为ovpn的路由语句
func CIDR2OVPNRouterClause(src string) (res string) {
	dnsIp, err := utils.ResolveIP(src) // 将域名解析
	if err != nil {
		// 如果是CIDR则生成子网掩码
		_, ipv4Net, _ := net.ParseCIDR(src)
		res += fmt.Sprintf(RouterTmp, ipv4Net.IP, utils.Ipv4MaskString(ipv4Net.Mask))
	} else {
		res += fmt.Sprintf(RouterTmp, dnsIp, "255.255.255.255")
	}
	return
}

// ScanUVPNUserCCD 扫描所有ldap用户，去匹配ccd文件，不存在对应用户的ccd就可以删掉了--删除操作尽量手动删除 防止放在循环中因为意外清理掉了所有用户文件
func ScanUVPNUserCCD() {
	fmt.Println("##################")
	var ldapUserKeys []interface{} // ldap用户 账号名
	var ldapUser = map[string]*ldap.Entry{}
	users := uuap.FetchLdapUsers(&uuap.LdapAttributes{})
	for _, user := range users {
		ldapUserKeys = append(ldapUserKeys, user.GetAttributeValue("sAMAccountName"))
		ldapUser[user.GetAttributeValue("sAMAccountName")] = user
	}

	count := 0
	count2 := 0
	rd, _ := ioutil.ReadDir(conf.Conf.System.DevCCDFilePath)
	for _, file := range rd {
		// ccd文件存在ldap用户账号名映射的 保留
		if utils.IsInSlice(file.Name(), ldapUserKeys) {
			count++
			firstLine, err := utils.ExtractViPFromCCD(conf.Conf.System.DevCCDFilePath + "/" + file.Name())
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(count, ldapUser[file.Name()].GetAttributeValue("employeeNumber"), file.Name(), ldapUser[file.Name()].GetAttributeValue("displayName"), firstLine)

		} else { // ccd文件名不存在ldap用户账号名映射的 报告出来，手动处理
			count2++
			fmt.Println("+++++++++++++ccd文件没有ldap用户映射的有：", count2, file.Name(), "如果是机器账号可能是因为没有设置mail字段，在查询时被过滤掉了+++++++++++++++++")
		}

	}

}

func main() {
	path := flag.String("config", "", "指定配置文件地址")
	flag.Parse()
	conf.ConfPath = *path
	conf.Conf, _ = conf.Init(conf.ConfPath)

	if conf.Conf.System.CCDFilePath == "" {
		panic("uvpn ccd 路径不可为空！")
	}
	if conf.Conf.LdapCfg.ConnUrl == "" || conf.Conf.LdapCfg.AdminAccount == "" || conf.Conf.LdapCfg.BaseDn == "" {
		panic("LDAP连接信息不可以为空！")
	}

	// 初始化日志
	logger.Init()

	// 初始化LDAP连接池
	if err := uuap.Init(conf.Conf); err != nil {
		panic(err)
	}

	// 初始化缓存
	if err := cache.Init(&conf.Conf.Redis); err != nil {
		panic(err)
	}

	//ScanUVPNUserCCD()

	// 消费者
	Consumer()

	//ScanUVPNUserCCD()
	//err := GenerateCCD4User(conf.Conf.System.DevCCDFilePath + "/" + "test")
	//if err != nil {
	//	log.Error(err)
	//}

}
