package main

import (
	log "github.com/sirupsen/logrus"
	"myDocker/container"
)

func removeContainer(containerId string, force bool) {
	ContainerInfo, err := getContainerInfoById(containerId)
	if err != nil {
		log.Errorf("Get container %s info error %v", containerId, err)
		return
	}
	/*判断容器状态，选择相应的操作*/
	switch ContainerInfo.Status {
	case container.STOP:
		//dirPath := fmt.Sprintf(container.InfoLocFormat, containerId)
		if err := container.DeleteContainerInfo(containerId); err != nil {
			log.Errorf("Remove container %s error %v", containerId, err)
			return
		}
		container.DeleteWorkSpace(containerId, ContainerInfo.Volume)
	case container.RUNNING:
		if force {
			stopContainer(containerId)
			removeContainer(containerId, force)
		} else {
			log.Errorf("can not remove a running container %s ", containerId)
			return
		}
	default:
		log.Errorf("can not remove container,invalid status %s ", ContainerInfo.Status)
		return
	}
}
