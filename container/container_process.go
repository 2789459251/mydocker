package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"myDocker/constant"
	"myDocker/utils"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func setUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Get current directory err:%v", err)
	}
	logrus.Info("Current location is %s", pwd)
	pivotRoot(pwd)

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NODEV | syscall.MS_NOSUID
	//proc 是一个虚拟文件系统，提供了关于系统内核状态和进程信息的接口
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	//tmpfs 是一个内存文件系统，用于临时存储数据
	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")

}

// NewParentProcess 构建 command 用于启动一个新进程
/*
这里是父进程，也就是当前进程执行的内容。
1.这里的/proc/se1f/exe调用中，/proc/self/ 指的是当前运行进程自己的环境，exec 其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
2.后面的args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化进程的一些环境和资源
3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离新创建的进程和外部环境。
4.如果用户指定了-it参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool, volume, containnerId, ImageName string) (*exec.Cmd, *os.File) {
	// 创建匿名管道用于传递参数，将readPipe作为子进程的ExtraFiles，子进程从readPipe中读取参数
	// 父进程中则通过writePipe将参数写入管道
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
		return nil, nil
	}
	//创建一个新的exec.Cmd对象，调用当前执行的程序本身，先执行initCommand初始化环境、资源
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		/*后台运行就重定向输出写入日志文件*/
		dir := fmt.Sprintf(InfoLocFormat, containnerId)
		if err := os.MkdirAll(dir, 0622); err != nil {
			logrus.Errorf("NewParentProcess MkdirAll %s err:%v", dir, err)
			return nil, nil
		}
		setLogPath := dir + LogFile
		setLogFile, err := os.Create(GetLogPath(setLogPath, containnerId))
		if err != nil {
			logrus.Errorf("NewParentProcess Create %s err:%v", setLogPath, err)
		}
		cmd.Stdout = setLogFile
		cmd.Stderr = setLogFile
	}
	//cmd.Dir = "/home/zsy/busybox"
	cmd.ExtraFiles = []*os.File{readPipe}
	//mntURL := "/home/zsy/merged/"
	//rootURL := "/home/zsy/"
	NewWorkSpace(containnerId, ImageName, volume)
	cmd.Dir = utils.GetMerged(containnerId)
	return cmd, writePipe
}

// 容器 执行的第一个进程，使用mount挂载proc文件系统，方便查看当前进程的资源情况
func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if len(cmdArray) == 0 || cmdArray == nil {
		return fmt.Errorf("Run container get user command error ,cmdArray is nil")
	}
	// container/init.go#RunContainerInitProcess 方法
	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示声明你要这个新的mount namespace独立。
	// 即 mount proc 之前先把所有挂载点的传播类型改为 private，避免本 namespace 中的挂载事件外泄。
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	// 设置默认的挂载标志，这些标志用于挂载 proc 文件系统
	// MS_NOEXEC: 阻止在挂载的文件系统上执行任何程序
	// MS_NODEV: 阻止在挂载的文件系统上访问任何设备文件
	// MS_NOSUID: 阻止挂载的文件系统上的 SUID 和 SGID 设置
	//defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NODEV | syscall.MS_NOSUID
	//syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	setUpMount()

	path, err := exec.LookPath(cmdArray[0])

	if err != nil {
		logrus.Errorf("Exec LookPath error %v", err)
		return err
	}
	logrus.Info("Find path %s", path)

	//调用内核的execve系统函数，执行程序，并覆盖当前进程的堆栈信息，使我们容器的PID为1的进程是用户的程序；
	//todo 第二个是命令的参数,怎么是这个写法
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		logrus.Error(err)
	}
	return nil
}

/*
创建新的挂载点：
首先，创建一个挂载点，通常是在新的根目录下创建一个.pivot_root目录。
重新挂载根目录：使用syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, "")命令，
将当前的根目录挂载到自身。这里的MS_BIND标志表示这是一个绑定挂载（bind mount），
它允许将一个目录挂载到另一个目录，而MS_REC标志表示递归挂载，即挂载指定目录及其所有子目录。
分离文件系统：通过将根目录挂载到自身，我们实际上是在创建一个新的挂载点，这个挂载点与原始的根目录在文件系统树上是独立的。
这意味着，当执行pivot_root时，我们可以将原始的根目录移动到.pivot_root目录下，而不会干扰到新的根目录。

执行pivot_root：此时，执行pivot_root(root, pivotDir)命令，将当前的工作目录切换到新的根目录，并将旧的根目录移动到.pivot_root目录下。

卸载临时目录：一旦pivot_root成功执行，旧的根目录就被移动到了.pivot_root目录下，此时可以卸载这个临时目录
*/
func pivotRoot(root string) error {
	//使当前的root与即将创建的root不在一个文件系统下，将root重新mount(bind mount换一个挂载点挂载)
	//root -> root递归地应用到子目录
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount rootfs to itself error %v", err)
	}
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, constant.Perm0755); err != nil {
		return err
	}

	//创建新的文件系统
	//整个系统切换到root,当前进程的old root文件移动到pivotDir
	//将当前的工作目录切换到新的根目录，并将旧的根目录移动到.pivot_root目录下。
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivotDir %v", err)
	}

	//更新当前的工作目录为新的根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %v", err)
	}

	//将之前用于 pivot_root 的临时目录 pivotDir 卸载（取消挂载）
	pivotDir = filepath.Join("/", ".pivot_root")
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root error %v", err)
	}

	return os.Remove(pivotDir)
}

func GenerateContainerID() string {
	return randStringBytes(IDLength)
}

/* 获取容器ID */
func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
