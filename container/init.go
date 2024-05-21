package container

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strings"
)

func readUserCommand() []string {
	// 传进来的管道一端
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		log.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}
