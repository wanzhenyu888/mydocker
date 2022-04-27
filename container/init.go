package container

import (
	"os"
	"syscall"
	
	"github.com/Sirupsen/logrus"
)

// 这里的init函数是在容器内部执行的，也就是说，代码执行到这里后，
// 容器所在的进程其实已经创建出来了，这是本容器执行的第一个进程。
// 使用mount先去挂载proc文件系统，以便后面通过ps等系统命令去查看当前进程资源的情况
func RunContainerInitProcess(command string, args []string) error {
	logrus.Infof("command %s", command)

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	argv := []string{command}
	if err := syscall.Exec(command, argv, os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}
