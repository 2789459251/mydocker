package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

// NewParentProcess 构建 command 用于启动一个新进程
/*
这里是父进程，也就是当前进程执行的内容。
1.这里的/proc/se1f/exe调用中，/proc/self/ 指的是当前运行进程自己的环境，exec 其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
2.后面的args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化进程的一些环境和资源
3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离新创建的进程和外部环境。
4.如果用户指定了-it参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
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
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	return cmd, writePipe
}

func NewParentProcess_(tty bool) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logrus.Errorf("new pipe error %v", err)
		return nil, nil
	}
	//args := []string{"init", command}
	//创建一个新的exec.Cmd对象，调用当前执行的程序本身，先执行initCommand初始化环境、资源
	cmd := exec.Command("/proc/self/exe", "init")
	//系统调用clone参数fork新进程，创建隔离的新容器进程
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET,
	}
	if tty {
		// 如果启用了tty，设置标准输入输出错误
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	// 在这里传入管道读取端的句柄
	cmd.ExtraFiles = []*os.File{readPipe} // ->带着这个句柄去创建子进程 ->readpipe成为了第四个文件描述符
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
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NODEV | syscall.MS_NOSUID
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

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

func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
