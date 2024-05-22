package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"myDocker/cgroups"
	"myDocker/cgroups/subsystems"
	"myDocker/container"
	"os"
	"strings"
)

const usage = `mydocker is a simple container runtime implementation.
			   The purpose of this project is to learn how docker works and how to write a docker by ourselves
			   Enjoy it, just for fun.`

func main() {
	//创建一个cli实例
	app := cli.NewApp()
	app.Name = "mydocker"
	app.Usage = usage

	//定义cli应用命令列表
	app.Commands = []cli.Command{
		initCommand,
		runCommand,
	}

	//在docker启动之前执行的钩子函数，设置docker日志打印
	app.Before = func(c *cli.Context) error {
		log.SetFormatter(&log.JSONFormatter{}) //日志格式
		log.SetOutput(os.Stdout)               //将日志输出到标准输出
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit
			mydocker run -it [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			// 简单起见，这里把 -i 和 -t 参数合并成一个
			Name:  "it",
			Usage: "enable tty",
		},
		cli.StringFlag{
			Name:  "mem", // 限制进程内存使用量，为了避免和 stress 命令的 -m 参数冲突 这里使用 -mem,到时候可以看下解决冲突的方法
			Usage: "memory limit,e.g.: -mem 100m",
		},
		cli.StringFlag{
			Name:  "cpu",
			Usage: "cpu quota,e.g.: -cpu 100", // 限制进程 cpu 使用率
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit,e.g.: -cpuset 2,4", // 限制进程 cpu 使用率
		},
	},
	/*
		这里是run命令执行的真正函数。
		1.判断参数是否包含command
		2.获取用户指定的command
		3.调用Run function去准备启动容器:
	*/
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}

		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}

		tty := context.Bool("it")
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("mem"),
			CpuSet:      context.String("cpuset"),
			CpuCfsQuota: context.Int("cpu"),
		}
		log.Info("resConf:", resConf)

		Run(tty, cmdArray, resConf)
		return nil
	},
}
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(context *cli.Context) error {
		log.Infof("init come on")
		err := container.RunContainerInitProcess()
		return err
	},
}

// Run 执行具体 command
/*
	这里的Start方法是真正开始执行由NewParentProcess构建好的command的调用，它首先会clone出来一个namespace隔离的
进程，然后在子进程中，调用/proc/self/exe,也就是调用自己，发送init参数，调用我们写的init方法，
去初始化容器的一些资源。
*/
func Run(tty bool, comArray []string, res *subsystems.ResourceConfig) {
	//新建进程
	parent, writePipe := container.NewParentProcess(tty)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Errorf("Run parent.Start err:%v", err)
	}
	// 创建cgroup manager, 并通过调用set和apply设置资源限制并使限制在容器上生效
	cgroupManager := cgroups.NewCgroupManager("mydocker2")

	cgroupManager.Resource = res
	_ = cgroupManager.Set(res)
	_ = cgroupManager.Apply(parent.Process.Pid)
	//defer cgroupManager.Destroy()

	// 在子进程创建后才能通过pipe来发送参数
	sendInitCommand(comArray, writePipe)
	_ = parent.Wait()
}

// sendInitCommand 通过writePipe将指令发送给子进程
func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}
