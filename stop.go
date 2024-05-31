package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"myDocker/container"
	"os"
	"path"
	"strconv"
	"syscall"
)

func stopContainer(containerID string) {
	info, err := getContainerInfoById(containerID)
	if err != nil {
		log.Errorf(" stopContainer get container info by id error: %v", err)
		return
	}
	//注意路径
	dir := fmt.Sprintf(container.InfoLocFormat, containerID)
	configPath := path.Join(dir, container.ConfigName)

	pid := info.Pid
	PID, err := strconv.Atoi(pid)
	if err != nil {
		log.Errorf(" stopContainer convert PID to int error: %v", err)
		return
	}
	err = syscall.Kill(PID, syscall.SIGTERM)
	if err != nil {
		log.Errorf(" stopContainer kill error: %v", err)
		return
	}
	info.Status = container.STOP
	info.Pid = " "
	newContentByte, err := json.Marshal(info)
	if err != nil {
		log.Errorf(" stopContainer marshal error: %v", err)
		return
	}
	//dir := fmt.Sprintf(container.InfoLocFormat, containerID)
	//configPath := path.Join(dir, containerID)
	//fmt.Println(configPath)
	//if err_ := os.WriteFile(configPath, newContentByte, constant.Perm0622); err_ != nil {
	//	log.Errorf("Write file %s error:%v", configPath, err_)
	//	return
	//}
	//

	err_ := os.Remove(configPath)
	if err_ != nil {
		log.Errorf(" stopContainer remove config file error: %v", err)
		return
	}
	f, err := os.Create(configPath)
	if err != nil {
		log.Errorf("Create file error: %v", err)
		return
	}
	defer f.Close()

	_, err = f.Write(newContentByte)
	if err != nil {
		log.Errorf("Write file error: %v", err)
		return
	}
	return
}

func getContainerInfoById(containerId string) (*container.Info, error) {
	dir := fmt.Sprintf(container.InfoLocFormat, containerId)
	filePath := path.Join(dir, container.ConfigName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("getContainerInfoById read file %s error %v", filePath, err)
	}
	info := new(container.Info)
	if err = json.Unmarshal(content, info); err != nil {
		return nil, fmt.Errorf("json unmarshal error %v", err)
	}
	return info, nil
}
