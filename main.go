package main

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"myDocker/cgroups"
	"myDocker/cgroups/subsystems"
	"myDocker/container"
	"myDocker/utils"
	"os"
	"os/exec"
	"strings"
)

const usage = `mydocker is a simple container runtime implementation.
			   The purpose of this project is to learn how docker works and how to write a docker by ourselves
			   Enjoy it, just for fun.`

const (
	EnvExecPid = "mydocker_pid"
	EnvExecCmd = "mydocker_cmd"
)

func main() {
	//创建一个cli实例
	app := cli.NewApp()
	app.Name = "mydocker"
	app.Usage = usage

	//定义cli应用命令列表
	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		commitCommand,
		listCommand,
		logCommand,
		execCommand,
		stopCommand,
		removeCommand,
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

/*删除容器*/
var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove a container",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "f",
			Usage: "force remove the container",
		},
	},
	Action: func(c *cli.Context) error {
		if len(c.Args()) < 1 {
			return cli.NewExitError("missing container name", 1)
		}
		containerName := c.Args().Get(0)
		force := c.Bool("f")
		removeContainer(containerName, force)
		return nil
	},
}

/*停止容器*/
var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop container",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
	},
	Action: func(c *cli.Context) error {
		if len(c.Args()) < 1 {
			return fmt.Errorf("container name is required")
		}
		containerName := c.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

/*进入容器*/
var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		// 如果环境变量存在，说明C代码已经运行过了，即setns系统调用已经执行了，这里就直接返回，避免重复执行
		if os.Getenv(EnvExecPid) != "" {
			log.Infof("pid callback pid %v", os.Getgid())
			return nil
		}
		// 格式：mydocker exec 容器名字 命令，因此至少会有两个参数
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}
		containerName := context.Args().Get(0)
		// 将除了容器名之外的参数作为命令部分
		var commandArray []string
		for _, arg := range context.Args().Tail() {
			commandArray = append(commandArray, arg)
		}
		ExecContainer(containerName, commandArray)
		return nil
	},
}

/* ps命令 */
var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(c *cli.Context) error {
		ListContainers()
		return nil
	},
}

/* 运行指令 */
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
			Name:  "name",
			Usage: "container name"},
		/*后台启动*/
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
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
		cli.StringFlag{
			Name:  "v",
			Usage: "volume,e.g.: -v hostpath:containerpath",
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
		detach := context.Bool("d")
		containerName := context.String("name")
		if tty && detach {
			/*不能同时输出日志并且后态启动*/
			return fmt.Errorf("it and paramter can not both provided")
		}
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("mem"),
			CpuSet:      context.String("cpuset"),
			CpuCfsQuota: context.Int("cpu"),
		}
		log.Info("resConf:", resConf)
		volume := context.String("v")

		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]
		Run(tty, cmdArray, resConf, volume, containerName, imageName)
		return nil
	},
}

/*初始化命令*/
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(context *cli.Context) error {
		log.Infof("init come on")
		err := container.RunContainerInitProcess()
		return err
	},
}

/*打包镜像命令*/
var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "Commit container process image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing image name and container name")
		}
		containerID := context.Args().Get(0)
		imageName := context.Args().Get(1)
		return commitContainer(containerID, imageName)
	},
}

/*获取日志命令*/
var logCommand = cli.Command{
	Name:  "logs",
	Usage: "Show container logs",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		logContainner(containerName)
		return nil
	},
}

// Run 执行具体 command
/*
	这里的Start方法是真正开始执行由NewParentProcess构建好的command的调用，它首先会clone出来一个namespace隔离的
进程，然后在子进程中，调用/proc/self/exe,也就是调用自己，发送init参数，调用我们写的init方法，
去初始化容器的一些资源。
*/
func Run(tty bool, comArray []string, res *subsystems.ResourceConfig, volume, containerName, imageName string) {
	containerId := container.GenerateContainerID()
	//新建进程
	parent, writePipe := container.NewParentProcess(tty, volume, containerId, imageName)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Errorf("Run parent.Start err:%v", err)
	}

	/* 记录容器创建信息 */
	err := container.RecordContainerInfo(parent.Process.Pid, comArray, containerName, containerId, volume)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return
	}
	// 创建cgroup manager, 并通过调用set和apply设置资源限制并使限制在容器上生效
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Resource = res
	_ = cgroupManager.Set(res)
	_ = cgroupManager.Apply(parent.Process.Pid)

	// 在子进程创建后才能通过pipe来发送参数
	sendInitCommand(comArray, writePipe)

	if tty { // 如果是tty，那么父进程等待，就是前台运行，否则就是跳过，实现后台运行
		_ = parent.Wait()
		container.DeleteWorkSpace(containerId, volume)
		container.DeleteContainerInfo(containerId)
	}
}

// sendInitCommand 通过writePipe将指令发送给子进程
func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}

var ErrImageAlreadyExists = errors.New("Image Already Exists")

/*将容器打包成镜像*/
func commitContainer(containerId, imageName string) error {
	//mntPath := "/home/zsy/merged"
	//imageTar := "/home/zsy/" + imageName + ".tar"
	mntPath := utils.GetMerged(containerId)
	imageTar := utils.GetImage(imageName)
	exists, err := utils.PathExists(imageTar)
	if err != nil {
		return errors.WithMessagef(err, "check is image [%s/%s] exist failed", imageName, imageTar)
	}
	if exists {
		return ErrImageAlreadyExists
	}

	log.Info("commitCommand imageTar:", imageTar)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntPath, ".").CombinedOutput(); err != nil {
		return errors.WithMessagef(err, "tar folder %s  failed", mntPath)
	}
	return nil
}
