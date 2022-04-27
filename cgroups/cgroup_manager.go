package cgroups

import (
	"github.com/wanzhenyu888/mydocker/cgroups/subsystems"

	"github.com/Sirupsen/logrus"
)

// CgroupManager 将资源限制的配置，以及将进程移动到cgroup中的操作
// 交给各个subsystem去处理
type CgroupManager struct {
	// cgroup在heirarchy中的路径，相当于创建的cgroup目录
	// 相对于各root cgroup目录的路径
	Path string
	// 资源配置
	Resource *subsystems.ResourceConfig
}

// CgroupManager初始化
func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

// 将进程PID加入到每个cgroup中
func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemIns {
		subSysIns.Apply(c.Path, pid)
	}
	return nil
}

// 设置各个subsystem挂载中的cgroup资源限制
func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range subsystems.SubsystemIns {
		subSysIns.Set(c.Path, res)
	}
	return nil
}

// 释放各个subsystem挂载中的cgroup
func (c *CgroupManager) Destroy() error {
	for _, subSysIns := range subsystems.SubsystemIns {
		if err := subSysIns.Remove(c.Path); err != nil {
			logrus.Warnf("remove cgroup fail %v", err)
		}
	}
	return nil
}
