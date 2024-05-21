package cgroups

import (
	"github.com/sirupsen/logrus"
	"myDocker/cgroups/subsystems"
)

type CgroupManager struct {
	path     string
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{path: path}
}

// 将进程的PID加入subsystem资源限制处理链数组中
func (cm *CgroupManager) Apply(pid int) error {
	for _, subsystem := range subsystems.SubsystemsIns {
		subsystem.Apply(cm.path, pid)
	}
	return nil
}

func (cm *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subsystem := range subsystems.SubsystemsIns {
		subsystem.Set(cm.path, res)
	}
	return nil
}

// 释放各个subsystem挂载的cgroup
func (cm *CgroupManager) Destroy() error {
	for _, subsystem := range subsystems.SubsystemsIns {
		if err := subsystem.Remove(cm.path); err != nil {
			logrus.Warnf("remove cgroup failed %v", err)
		}
	}
	return nil
}
