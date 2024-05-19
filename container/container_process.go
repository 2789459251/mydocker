package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

func NewParentProcess(tty bool, command string) *exec.Cmd {
	args := []string{"init", command}
	//创建一个新的exec.Cmd对象，调用当前执行的程序本身，先执行initCommand初始化环境、资源
	cmd := exec.Command("/proc/self/exe", args...)
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
	return cmd
}

// 容器 执行的第一个进程，使用mount挂载proc文件系统，方便查看当前进程的资源情况
func RunContainerInitProcess(command string, args []string) error {
	logrus.Infof("command:%s", command)

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NODEV | syscall.MS_NOSUID
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	argv := []string{command}
	//调用内核的execve系统函数，执行程序，并覆盖当前进程的堆栈信息，使我们容器的PID为1的进程是用户的程序
	if err := syscall.Exec(command, argv, os.Environ()); err != nil {
		logrus.Error(err)
	}
	return nil
}
