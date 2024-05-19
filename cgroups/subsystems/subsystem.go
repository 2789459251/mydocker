package subsystems

type ResourceConfig struct {
	//内存限制
	MemoryLimit string
	//cpu时间权重
	CpuShare string
	//cpu核心数
	CpuSet string
}

// 将cgroup抽象为path
type Subsystem interface {
	Name() string
	//资源限制,设置某个cgroup在这个subsystem的限制
	Set(path string, res *ResourceConfig) error
	//将进程添加到某个cgroup中
	Apply(path string, pid int) error
	Remove(path string) error
}

// 通过不同的subsystem初始化实例创建资源限制处理链数组
var (
	SubsystemsIns = []Subsystem{
		&CpusetSubSystem{},
		&MemorySubsystem{},
		&CpuSubSystem{},
	}
)
