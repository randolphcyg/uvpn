package conf

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"mq/uuap"
)

var (
	Conf     = &uuap.Config{} // 全局配置文件结构体
	ConfPath string           // 全局配置文件的路径
)

// Init 初始化配置
func Init(path string) (*uuap.Config, error) {
	cfgFile, err := LoadConfig(path)
	if err != nil {
		log.Fatalf("LoadConfig: %v", err)
	}
	Conf, err = ParseConfig(cfgFile)
	if err != nil {
		log.Fatalf("ParseConfig: %v", err)
	}

	return Conf, nil
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*viper.Viper, error) {
	v := viper.New()
	if path != "" {
		v.SetConfigFile(path) // 如果指定了配置文件,则解析指定的配置文件
	} else {
		v.AddConfigPath("../conf/")
		v.SetConfigName("conf")
	}
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, errors.New("config file not found")
		}
		return nil, err
	}
	go watchConfig(v)

	return v, nil
}

// ParseConfig 解析配置文件
func ParseConfig(v *viper.Viper) (c *uuap.Config, err error) {
	if err = v.Unmarshal(&c); err != nil {
		return nil, err
	}
	return c, nil
}

// watchConfig 监控文件热更新
func watchConfig(v *viper.Viper) {
	v.WatchConfig()
	v.OnConfigChange(func(event fsnotify.Event) {
		fmt.Printf("Detect conf change: %s \n", event.String())
		Conf, err := Init(ConfPath)
		if err != nil {
			fmt.Printf("Reload cfg error, %s", err)
		}
		fmt.Println("重载后的配置文件：", Conf)
	})
}
