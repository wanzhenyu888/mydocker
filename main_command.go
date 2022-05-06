package main

import (
	"fmt"
	"os"

	"github.com/wanzhenyu888/mydocker/cgroups/subsystems"
	"github.com/wanzhenyu888/mydocker/container"

	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

// 这里定义了runCommand的Flags，其作用类似于运行命令时使用--来指定参数
var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups 
			limit mydocker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "set environment",
		},
	},
	/*
	 * 这里是run命令执行的真正函数
	 * 1. 判断参数是否包含command
	 * 2. 获取用户指定的command
	 * 3. 调用Run function去准备启动容器
	 */
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}

		// get image name
		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]

		tty := context.Bool("ti")
		detach := context.Bool("d")

		// 这里的tty和detach不能共存
		if tty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuShare:    context.String("cpuset"),
			CpuSet:      context.String("cpushare"),
		}
		log.Infof("tty %v", tty)
		volume := context.String("v")
		containerName := context.String("name")
		envSlice := context.StringSlice("e")

		Run(tty, cmdArray, resConf, volume, containerName, imageName, envSlice)
		return nil
	},
}

// 这里，定义了initCommand的具体操作，此操作为内部方法，禁止外部调用
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	/*
	 1. 获取传递过来的command参数
	 2. 执行容器初始化操作
	*/
	Action: func(context *cli.Context) error {
		log.Infof("init come on")
		err := container.RunContainerInitProcess()
		return err
	},
}

// commitCommand 用于容器退出时，把运行状态容器的内容
// 存储成镜像保存起来
var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := context.Args().Get(0)
		imageNmae := context.Args().Get(1)
		commitContainer(containerName, imageNmae)
		return nil
	},
}

// docker ps 展示正在运行的容器信息
var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {
		ListContainers()
		return nil
	},
}

// logCommand 日志
var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Please input your container name")
		}
		containerName := context.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}

// execCommand 进入后台进程
var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		// This is for callback
		if os.Getenv(ENV_EXEC_PID) != "" {
			log.Infof("%s", os.Getenv(ENV_EXEC_PID))
			log.Infof("pid callback pid %d", os.Getgid())
			return nil
		}

		// 命令格式是mydocker exec 容器名 命令
		if len(context.Args()) < 2 {
			return fmt.Errorf("Missing container name or command")
		}
		containerName := context.Args().Get(0)
		var commandArray []string
		// 将除了容器名之外的参数当作需要执行的命令处理
		for _, arg := range context.Args().Tail() {
			commandArray = append(commandArray, arg)
		}
		ExecContainer(containerName, commandArray)
		return nil
	},
}

var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}

		containerName := context.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove unused containers",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		containerName := context.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
}
