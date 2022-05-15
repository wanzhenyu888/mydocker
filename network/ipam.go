package network

import (
	"encoding/json"
	"net"
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// IP地址分配信息存放文件地址
const ipamDefaultAllocatorPath = "/var/run/mydocker/network/ipam/subnet.json"

// 存放IP地址分配信息
type IPAM struct {
	// 分配文件存放位置
	SubnetAllocatorPath string
	// 网段和位图算法的数组map，key是网段，value是分配的位图数组
	Subnets *map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// 加载网段地址分配信息
func (ipam *IPAM) load() error {
	// 通过os.Stat函数检查存储文件状态，如果不存在，则说明之前没有分配，则不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}

	// 打开并读取存储文件
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}

	// 将文件中的内容反序列化出IP的配置信息
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
		return err
	}

	return nil
}

// 存储网段地址分配信息
func (ipam *IPAM) dump() error {
	// 检查存储文件所在文件夹是否存在，如果不存在则创建
	// path.Split函数能够分割目录和文件
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			// 创建文件夹，os.MkdirAll相当于mkdir -p <dir>命令
			os.MkdirAll(ipamConfigFileDir, 0o644)
		} else {
			return err
		}
	}

	// 打开存储文件，os.O_TRUNC表示如果存在则清空，os.CREATE表示如果不存在则创建
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|
		os.O_WRONLY|os.O_CREATE, 0o644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}

	// 序列化ipam对象到json串
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}

	// 将序列化后的json串写入到配置文件中
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return err
	}

	return nil
}

// 在网段中分配一个可用的IP地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	// 存放网段中地址分配信息的数组
	ipam.Subnets = &map[string]string{}

	// 从文件中加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("Error load allocation info, %v", err)
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	// net.IPNet.Mask.Size()函数会返回网段前面的固定位的长度和网段的子网掩码的总长度
	one, size := subnet.Mask.Size()

	// 如果之前没有分配过这个网段，则初始化网段的分配配置
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		// 用“0”填满这个网段的配置，1 << uint8(size - one)表示这个网段中有多少个可用地址
		// “size - one”是子网掩码后面的网络位数，2^(size - one)表示网段中可用IP数
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}

	// 遍历网段的位图数组
	for c := range (*ipam.Subnets)[subnet.String()] {
		// 找到数组中位“0”的项和数组序号，即可以分配的IP
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// 设置这个为“0”的序号值为“1”，即分配这个IP
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			// Go的字符串创建后就不能修改，所以通过转换成byte数组，修改后再转换成字符串赋值
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			// 这里的IP为初始IP，比如对于网段192.168.0.0/16，这里就是192.168.0.0
			ip = subnet.IP

			// 通过网段的IP与上面的偏移相加计算出分配的IP地址，由于IP地址是uint的一个数组，
			// 需要通过数组中的每一项加所需要的值，比如网段172.16.0.0/12，数组序号是65555，
			// 那么在[172,16,0,0]上依次加[uint8(65555 >> 24)、uint8(65555 >> 16)、
			// uint8(65555 >> 8)、uint8(65555 >> 0)]，即[0, 1, 0, 19]，那么获得的IP是
			// 172.17.0.19
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			// 由于此处IP是从1开始分配的，所以最后再加1，最终得到分配的IP为172.17.0.20
			ip[3] += 1
			break
		}
	}

	// 通过调用dump()将分配结果保存到文件中
	ipam.dump()
	return
}

// 在一个网段中释放一个IP
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	// 从文件中加载网段的分配信息
	err := ipam.load()
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
	}

	// 计算IP地址在网段位图数组中的索引位置
	c := 0
	// 将IP地址转换成4个字节的表示方式
	releaseIP := ipaddr.To4()
	// 由于IP是从1开始分配的，所以转换成索引应减1
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		// 与分配IP相反，释放IP获得索引的方式是IP地址的每一位相减之后分别
		// 左移将对应的数值加到索引上
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	// 将分配的位图数组中索引位置的值置为0
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	// 保存释放掉IP之后的网段IP分配信息
	ipam.dump()
	return nil
}
