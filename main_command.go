package main

import (
	"fmt"

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
		tty := context.Bool("ti")
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuShare:    context.String("cpuset"),
			CpuSet:      context.String("cpushare"),
		}
		// 把volume参数传给Run函数
		volume := context.String("v")
		Run(tty, cmdArray, resConf, volume)
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
var commitCommand = cli.Command {
	Name: "commit",
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container name")
		}
		imageNmae := context.Args().Get(0)
		commitContainer(imageNmae)
		return nil
	},
}
