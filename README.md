## UVPN

此项目是处理uvpn的消费者端，consumer 下的 consumer.go打包放到uvpn服务器后台跑就可以了，日志文件会放在 uvpn.log

producer是生产者的测试文件，可以用来生成mq消息，用来测试。生产在用的是放在UUAP服务中的。

### 开发时跑消费者的方法
```shell
go run consumer.go 
```

### 开发时跑生产者
```shell
go run producer.go
```

### 打包方法

1. mac中打linux可执行文件
```shell
cd uvpn/consumer
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o uvpn
```

2. windows打linux可执行文件
```shell
set GOARCH=amd64
set GOOS=linux
go build -o uvpn
```


### 放在centos7服务器直接跑

```shell
# 将二进制可执行文件通过jumpserver上传到uvpn服务器，然后到/opt/uvpn/src目录下
# 将配置文件上传到/opt/uvpn/conf目录下，仅需要第一次上传，后面更新即可；或者可以通过-config指令指定配置文件绝对路径
cd /opt/uvpn/src
sudo mv /home/zhenyun/uvpn .
chmod 777 ./uvpn

cd /opt/uvpn/conf
sudo mv /home/zhenyun/conf.yaml .
```

配置文件格式

conf.yaml

```shell
system:
  Dev: true
  CCDFilePath: /etc/openvpn/ccd
  DevCCDFilePath: /Users/randolph/goodjob/uvpn/ccd

redis:
  Addr: x.x.x.x:6379
  Password: ""
  DB: 5
  PoolSize: 12000
  DialTimeout: 60s
  ReadTimeout: 500ms
  WriteTimeout: 500ms

ldapCfg:
  ConnUrl:       ldap://192.168.x.x:389
  BaseDn:        DC=x,DC=com
  AdminAccount:  CN=Admin,CN=Users,DC=x,DC=com
  Password:      xxxxxxxxxxxxx
  SslEncryption: False
  Timeout:       60

rocketMQ:
  Addr: 192.168.x.x
  Port: 9876
  TopicName: UVPN
```

**注意生产服务器上配置文件的Dev参数一定要设置为`false`!这样处理ccd文件的目录才是正确的～**

- 跑起来程序

```shell
sudo ./uvpn &
```

如果指定配置文件路径则为：
```shell
sudo ./uvpn -config /opt/uvpn/testConf.yaml &
```

- 正确执行二进制文件后，日志文件`uvpn.log`将生成在同目录下;
- mq的日志会不断出现在当前页面，可以contrl+c后关闭此tab页面，新开tab页面操作服务器

### TODO

1. 完善反馈消息 【待优化】
2. 要借助redis为新的UVPN用户初始化权限；这一步做完负担就只有审核了 【ok】