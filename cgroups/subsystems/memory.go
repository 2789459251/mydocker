package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

// type MemorySubsystem struct{}
//
// func (s *MemorySubsystem) Name() string { return "memory" }
//
//	func (s *MemorySubsystem) Set(cgroupPath string, res *ResourceConfig) error {
//		// 获得当前subsystem在虚拟文件系统中的路径
//		if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
//			if res.MemoryLimit != "" {
//				// 设置cgroup的内存限制->memory_limit_in_bytes文件
//				if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "memory_limit_in_bytes"), []byte(res.MemoryLimit), 0644); err != nil {
//					return fmt.Errorf("set cgroup memory limit fail: %s", err)
//				}
//			}
//			return nil
//		} else {
//			//获取实际cgroup路径出错
//			return err
//		}
//
// }
//
//	func (s *MemorySubsystem) Apply(cgroupPath string, pid int) error {
//		if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
//			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
//				return fmt.Errorf("set cgroup proc fail: %s", err)
//			}
//			return nil
//		} else {
//			return fmt.Errorf("get cgroup %s path fail: %s", cgroupPath, err)
//		}
//	}
//
//	func (s *MemorySubsystem) Remove(path string) error {
//		if subsysCgroupPath, err := GetCgroupPath(s.Name(), path, false); err == nil {
//			return os.RemoveAll(subsysCgroupPath)
//		} else {
//			return err
//		}
//	}
type MemorySubsystem struct {
}

func (s *MemorySubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {
		fmt.Println("路径：", subsysCgroupPath)
		if res.MemoryLimit != "" {
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup memory fail %v", err)
			}
		}
		return nil
	} else {
		return err
	}

}

func (s *MemorySubsystem) Remove(cgroupPath string) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		fmt.Println("路径：", subsysCgroupPath)
		return os.RemoveAll(subsysCgroupPath)
	} else {
		return err
	}
}

func (s *MemorySubsystem) Apply(cgroupPath string, pid int) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		fmt.Println("路径：", subsysCgroupPath)
		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("set cgroup proc fail %v", err)
		}
		return nil
	} else {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
}

func (s *MemorySubsystem) Name() string {
	return "memory"
}
