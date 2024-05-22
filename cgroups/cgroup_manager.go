package cgroups

import (
	"github.com/sirupsen/logrus"
	"myDocker/cgroups/subsystems"
)

type CgroupManager struct {
	//cgroup在hierarchy中的路径，相当于创建cgroup相对于root cgroup目录的路径
	path     string
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{path: path}
}

// 将进程的PID加入subsystem资源限制处理链数组中
func (cm *CgroupManager) Apply(pid int) error {
	for _, subsystem := range subsystems.SubsystemsIns {
		err := subsystem.Apply(cm.path, pid, cm.Resource)
		if err != nil {
			logrus.Errorf("apply subsystem:%s err:%s", subsystem.Name(), err)
		}
	}
	return nil
}

func (cm *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subsystem := range subsystems.SubsystemsIns {
		err := subsystem.Set(cm.path, res)
		if err != nil {
			logrus.Errorf("apply subsystem:%s err:%s", subsystem.Name(), err)
		}
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
