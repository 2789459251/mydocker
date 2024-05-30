package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"myDocker/container"
	"os"
)

/*从日志文件中读取数据*/
func logContainner(containerID string) {
	logFileLocation := fmt.Sprintf(container.InfoLocFormat, containerID) + container.LogFile
	logPath := container.GetLogPath(logFileLocation, containerID)
	file, err := os.Open(logPath)
	if err != nil {
		log.Errorf("Log container open file %s error %v", logFileLocation, err)
		return
	}
	//读取文件
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("Log container read file %s error %v", logFileLocation, err)
		return
	}
	//终端输出
	_, err = fmt.Fprint(os.Stdout, string(content))
	if err != nil {
		log.Errorf("Log container Fprint  error %v", err)
		return
	}

}
