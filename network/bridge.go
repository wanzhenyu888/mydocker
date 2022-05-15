package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type BridgeNetWorkDriver struct{}

func (d *BridgeNetWorkDriver) Name() string {
	return "bridge"
}

func (d *BridgeNetWorkDriver) Create(subnet, name string) (*Network, error) {
	// 通过net包的net.ParseCIDR方法，取到网段的字符串中的网关IP地址和网络IP段
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip

	// 初始化网络对象
	network := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  d.Name(),
	}
	err := d.initBridge(network)
	if err != nil {
		log.Errorf("error init bridge: %v", err)
	}

	return network, err
}

func (d *BridgeNetWorkDriver) Delete(network Network) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

// 连接一个网络和网络端点
func (d *BridgeNetWorkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	// 通过接口名获取到Linux Bridge接口的对象和接口属性
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	// 创建Veth接口的配置
	la := netlink.NewLinkAttrs()
	// 由于Linux接口名的限制，名字取endpoint ID的前5位
	la.Name = endpoint.ID[:5]
	// 通过设置Veth接口的master属性，设置这个Veth的一端挂载到网络对应的Linux Bridge上
	la.MasterIndex = br.Attrs().Index

	// 创建Veth对象，通过PeerName配置Veth另外一端的接口名
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	// 调用netlink的LinkAdd方法创建出这个Veth接口
	// 因为上面指定了link的MasterIndex是网络对应的Linux Bridge
	// 所以Veth的一端就已经挂载到了网络对应的Linux Bridge上
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Add Endpoint Device: %v", err)
	}

	// 调用netlink的LinkSetUp方法，设置Veth启动
	// 相当于ip link set xxx up命令
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Add Endpoint Device: %v", err)
	}

	return nil
}

func (d *BridgeNetWorkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

// 初始化Bridge设备
func (d *BridgeNetWorkDriver) initBridge(network *Network) error {
	// 1. 创建网桥；如果网桥存在则不创建
	bridgeName := network.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("Error add bridge: %s, Error: %v", bridgeName, err)
	}

	// 2. 设置网桥设备的地址和路由
	gatewayIP := *network.IpRange
	gatewayIP.IP = network.IpRange.IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("Error assigning address: %s on bridge: %s with an error of: %v", gatewayIP, bridgeName, err)
	}

	// 3. 启动Bridge设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("Error set bridge up: %s, Error: %v", bridgeName, err)
	}

	// 4. 设置iptables的SNAT规则
	if err := setupIPTables(bridgeName, network.IpRange); err != nil {
		return fmt.Errorf("Error setting iptables for %s: %v", bridgeName, err)
	}

	return nil
}

// 创建网桥
func createBridgeInterface(bridgeName string) error {
	// 检查是否已经存在同名设备
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// 初始化一个netlink的Link基础对象，Link的名字即Bridge虚拟设备的名字
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	// 使用刚才创建的Link属性创建netlink的Bridge对象
	br := &netlink.Bridge{la}
	// netlink的Linkadd方法用来创建虚拟网络设备，相当于ip link add xxx
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("Bridge creation failed for bridge %s: %v", bridgeName, err)
	}
	return nil
}

// 设置Bridge设备的地址和路由
func setInterfaceIP(name, rawIP string) error {
	// 通过netlink的LinkByName方法找到需要设置的网络接口
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("error get interface: %v", err)
	}

	// 由于netlink.ParseIPNet是对net.ParseCIDR的一个封装
	// 因此可以将net.ParseCIDR的返回值中的IP和net整合
	// 返回值中的ipNet既包含了网段的信息，192.168.0.0/24，也包含了原始的ip 192.168.0.1
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}

	// 通过netlink.AddrAdd给网络接口配置地址，相当于ip addr add xxx的命令
	// 同时如果配置了地址所在网段的信息，例如192.168.0.0/24
	// 还会配置路由表192.168.0.0/24转发到这个testbridge的网络接口上
	addr := &netlink.Addr{ipNet, "", 0, 0, nil}
	return netlink.AddrAdd(iface, addr)
}

// 启动Bridge设备，设置网络接口为UP状态
func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link name [ %s ]: %v", iface.Attrs().Name, err)
	}

	// 通过“netlink”的“LinkSetUp”方法设置接口状态为“UP”状态
	// 等价于ip link set xxx up命令
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("Error enabling interface for %s: %v", interfaceName, err)
	}
	return nil
}

// 设置iptables对应的bridge的MASQUERADE规则
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	// 由于Golang没有直接操控iptables操作的库，所以需要通过命令的方式来配置
	// 创建iptables的命令
	// iptables -t nat -A POSTROUTING -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
	iptablesCmd := fmt.Sprintf("-t nat -A ROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	// 执行iptables命令配置SNAT规则
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return nil
}
