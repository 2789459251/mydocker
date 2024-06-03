package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"myDocker/container"
	"os"
	"os/exec"
	"path"
	"strings"
	//导入“C”包
	_ "myDocker/nsenter"
)

func ExecContainer(containerId string, comArray []string) {
	// 根据传进来的容器名获取对应的PID
	pid, err := getPidByContainerId(containerId)
	if err != nil {
		log.Errorf("Exec container getContainerPidByName %s error %v", containerId, err)
		return
	}

	//简单 fork 出了一个进程
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 把命令拼接成字符串，便于传递
	cmdStr := strings.Join(comArray, " ")
	log.Infof("container pid：%s command：%s", pid, cmdStr)
	//设置环境变量
	_ = os.Setenv(EnvExecPid, pid)
	_ = os.Setenv(EnvExecCmd, cmdStr)
	//获得原来的环境变量
	// 把指定PID进程的环境变量传递给新启动的进程，实现通过exec命令也能查询到容器的环境变量
	env := getEnvsByPid(pid)
	cmd.Env = append(os.Environ(), env...)
	if err = cmd.Run(); err != nil {
		log.Errorf("Exec container %s error %v", containerId, err)
	}
}
func getPidByContainerId(containerId string) (string, error) {
	dir := fmt.Sprintf(container.InfoLocFormat, containerId)
	dirPath := path.Join(dir, container.ConfigName)
	jsonbyte, err := os.ReadFile(dirPath)
	if err != nil {
		return "", fmt.Errorf("getPidByContainerId read file %s err:%v", string(jsonbyte), err)
	}
	info := &container.Info{}
	err = json.Unmarshal(jsonbyte, info)
	if err != nil {
		return "", fmt.Errorf("getPidByContainerId  json unmarshal %s err:%v", string(jsonbyte), err)
	}
	return info.Pid, nil
}

func getEnvsByPid(pid string) []string {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("getEnvsByPid read file %s err:%v", path, err)
		return nil
	}
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}
