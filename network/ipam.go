package network

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"myDocker/constant"
	"net"
	"os"
	"path"
	"strings"
)

type IPAM struct {
	SubnetAllocatorPath string             // 分配文件存放位置
	Subnets             *map[string]string // 网段和位图算法的数组 map, key 是网段， value 是分配的位图数组
}

const ipamDefaultAllocatorPath = "/var/lib/mydocker/network/ipam/subnet.json"

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

func (ipam *IPAM) load() error {
	ipam.SubnetAllocatorPath = ipamDefaultAllocatorPath
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		//没有文件就不要加载，且没有错
		return nil
	}
	//读取文件
	f, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return err
	}
	defer f.Close()
	var data []byte = make([]byte, 2000)
	n, err := f.Read(data)
	if err != nil {
		return errors.Wrap(err, "read subnet config file error")
	}
	err = json.Unmarshal(data[:n], ipam.Subnets)
	return errors.Wrap(err, "err dump allocation info")
}

// dump 存储网段地址分配信息
func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(ipamConfigFileDir, constant.Perm0644); err != nil {
			return err
		}
	}
	// 打开存储文件 O_TRUNC 表示如果存在则消空， os O_CREATE 表示如果不存在则创建
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, constant.Perm0644)
	if err != nil {
		return err
	}
	defer subnetConfigFile.Close()

	dataJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	_, err = subnetConfigFile.Write(dataJson)
	return err
}

/* Allocate 在网段中分配一个可用的 IP 地址*/
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	ipam.Subnets = &map[string]string{}

	err = ipam.load()
	if err != nil {
		return nil, errors.Wrap(err, "load subnet allocation info error")
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())
	one, size := subnet.Mask.Size()
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		// ／用“0”填满这个网段的配置，uint8(size - one ）表示这个网段中有多少个可用地址
		// size - one是子网掩码后面的网络位数，2^(size - one)表示网段中的可用IP数
		// 而2^(size - one)等价于1 << uint8(size - one)
		// 左移一位就是扩大两倍
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}
	for c := range (*ipam.Subnets)[subnet.String()] {
		// 找到数组中为“0”的项和数组序号，即可以分配的 IP
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// 设置这个为“0”的序号值为“1” 即标记这个IP已经分配过了
			// Go 的字符串，创建之后就不能修改 所以通过转换成 byte 数组，修改后再转换成字符串赋值
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			// 这里的 subnet.IP只是初始IP，比如对于网段192 168.0.0/16 ，这里就是192.168.0.0
			ip = subnet.IP
			/*
				还需要通过网段的IP与上面的偏移相加计算出分配的IP地址，由于IP地址是uint的一个数组，
				需要通过数组中的每一项加所需要的值，比如网段是172.16.0.0/12，数组序号是65555,
				那么在[172,16,0,0] 上依次加[uint8(65555 >> 24)、uint8(65555 >> 16)、
				uint8(65555 >> 8)、uint8(65555 >> 0)]， 即[0, 1, 0, 19]， 那么获得的IP就
				是172.17.0.19.
			*/
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			// ／由于此处IP是从1开始分配的（0被网关占了），所以最后再加1，最终得到分配的IP 172.17.0.20
			ip[3] += 1
			break
		}
	}
	// 最后调用dump将分配结果保存到文件中
	err = ipam.dump()
	if err != nil {
		log.Error("Allocate：dump ipam error", err)
	}
	return
}

/*解除ip使用*/
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}
	_, subnet, _ = net.ParseCIDR(subnet.String())
	err := ipam.load()
	if err != nil {
		return errors.Wrap(err, "load subnet allocation info error")
	}
	//跟分配一样的算法 根据ip算出位数图中对应索引的位置
	c := 0

	releaseIp := ipaddr.To4()
	releaseIp[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIp[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	fmt.Println(c)
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	err = ipam.dump()
	if err != nil {
		return errors.Wrap(err, "Allocate：dump ipam error")
	}
	return nil
}
