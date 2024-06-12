package network

import (
	"github.com/vishvananda/netlink"
	"net"
)

/*网络*/
type Network struct {
	Name    string     //网络名
	IpRange *net.IPNet //地址段
	Driver  string     //网络驱动名
}

/*容器网络端点*/
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"portmapping"`
	Network     *Network
}

/* 网络驱动 */
type Driver interface {
	//驱动名
	Name() string
	//创建网络
	Create(subnet string, name string) (*Network, error)
	//删除网络
	Delete(name string) error
	//连接容器网络端点到网络
	Connect(network *Network, endpoint *Endpoint) error
	//从网络中移除容器网络端点
	Disconnect(network Network, endpoint *Endpoint) error
}

/*IPAM 用于网络IP地址的分配和释放*/
type IPAMer interface {
	Allocate(subnet *net.IPNet) (ip net.IP, err error) // 从指定的 subnet 网段中分配 IP 地址
	Release(subnet *net.IPNet, ipaddr *net.IP) error   // 从指定的 subnet 网段中释放掉指定的 IP 地址。
}
