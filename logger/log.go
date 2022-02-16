package logger

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func Init() {
	log.SetFormatter(&log.TextFormatter{})
	file, err := os.OpenFile("uvpn.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Error("failed to create log file!", err.Error())
	}
	log.SetOutput(file)
	//设置最低loglevel
	log.SetLevel(log.InfoLevel)
	log.SetReportCaller(true) // 报告错误文件和行号等信息
}
