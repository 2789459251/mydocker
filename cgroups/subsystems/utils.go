package subsystems

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

func FindCgroupMountpoint(subsystem string) string {
	//找出当前进程相关的mount信息
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		txt := scanner.Text()
		filelds := strings.Split(txt, " ")
		// 遍历每一条cgroup信息 找到对应的subsystem的挂载目录
		for _, opt := range strings.Split(filelds[len(filelds)-1], ",") {
			if opt == subsystem {
				return filelds[4]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err.Error()
	}
	return ""
}

// 找到对应的subsystem挂载的hieraechy相对路径的cgroup在虚拟文件的系统路径
func GetCgroupPath(subsystem string, cgrouPath string, autoCreate bool) (string, error) {
	cgroupRoot := FindCgroupMountpoint(subsystem)
	var err error
	if _, err = os.Stat(path.Join(cgroupRoot, cgrouPath)); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err = os.Mkdir(path.Join(cgroupRoot, cgrouPath), 0755); err != nil {
				return "", fmt.Errorf("failed to create cgroup mount point %v", err)
			}
			return path.Join(cgroupRoot, cgrouPath), nil
		}
	}
	return "", fmt.Errorf("cgroup path error %v", err)
}
