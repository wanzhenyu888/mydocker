package main

import (
	"fmt"

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
			Name: "ti",
			Usage: "enable tty",
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
		log.Infof("run come on")
		cmd := context.Args().Get(0)
		tty := context.Bool("ti")
		Run(tty, cmd)
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
		cmd := context.Args().Get(0)
		log.Infof("command %s", cmd)
		err := container.RunContainerInitProcess(cmd, nil)
		return err
	},
}
