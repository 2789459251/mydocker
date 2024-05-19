package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"myDocker/container"
	"os"
)

const usage = ` mydocker is a simple container runtime implementation.
				The purpose of this project is to learn how docker works and how to write a docker by myself.
				Enjoy it,just for fun`

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
		log.SetFormatter(&log.JSONFormatter{}) //日至格式
		log.SetOutput(os.Stdout)               //将日志输出到标准输出
		return nil
	}

	//运行cli应用，传入命令参数
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// 定义 ‘run’ 命令
var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namesoace and cgroups limit
			mydocker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti", // 定义一个布尔类型的flag，用于启用tty
			Usage: "enable tty",
		},
	},
	Action: func(c *cli.Context) error {
		//检查至少有一个参数（容器命令）
		if len(c.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		//获取第一个参数作为容器命令
		cmd := c.Args().Get(0)
		// 获取 'ti' flag的值
		tty := c.Bool("ti")
		Run(tty, cmd)
		return nil
	},
}

var initCommand = cli.Command{
	Name: "init",
	Usage: `Init container process run user's process in container. 
		Do not call it outside.`,
	Action: func(c *cli.Context) error {
		log.Infof("init come on")
		//获取第一个参数命令
		cmd := c.Args().Get(0)
		//记录命令日志
		log.Infof("command: %s", cmd)
		// 调用容器的初始化进程运行函数
		err := container.RunContainerInitProcess(cmd, nil)
		return err
	},
}

func Run(tty bool, command string) {
	parent := container.NewParentProcess(tty, command)
	// 开始前面创建好的commond调用，clone出隔离进程，在子进程中调用自己，发送init参数
	if err := parent.Start(); err != nil {
		log.Fatal(err)
	}
	parent.Wait()
	os.Exit(-1)
}
