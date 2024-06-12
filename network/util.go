package network

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"myDocker/container"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

/*创建网络设备*/
func createBridgeInterface(bridgeName string) error {
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		//已经存在该名字的网络设备
		return err
	}
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	// 使用刚才创建的Link的属性创netlink Bridge对象
	br := &netlink.Bridge{LinkAttrs: la}
	// 调用 net link Linkadd 方法，创 Bridge 虚拟网络设备:ip link add xxxx
	if err := netlink.LinkAdd(br); err != nil {
		return errors.Wrapf(err, "create bridge %s error", bridgeName)
	}
	return nil
}

/*分配ip地址*/
func setInterfaceIP(bridge, ip string) error {
	//addr add "xxx.xx.xx.x" dev bridge
	retries := 2
	var iface netlink.Link
	var err error
	for ; retries > 0; retries-- {
		iface, err = netlink.LinkByName(bridge)
		if err == nil {
			break
		}
		log.Debugf("error retrieving new bridge netlink link [ %s ]... retrying", bridge)
		time.Sleep(2 * time.Second)
	}
	//两次尝试 获得网络设备失败
	if err != nil {
		return errors.Wrap(err, "abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot")
	}
	//子网掩码
	ipNet, err := netlink.ParseIPNet(ip)
	if err != nil {
		return err
	}
	addr := &netlink.Addr{IPNet: ipNet}
	return netlink.AddrAdd(iface, addr)
}

/*启动网络设备*/
func setInterfaceUP(bridge string) error {
	link, err := netlink.LinkByName(bridge)
	if err != nil {
		return errors.Wrapf(err, "error retrieving a link named [ %s ]:", link.Attrs().Name)
	}
	//ip link set XX up
	err = netlink.LinkSetUp(link)
	if err != nil {
		return errors.Wrapf(err, "error enabling bridge [ %s ]:", bridge)
	}
	return nil
}

/*
通过直接执行 iptables 命令，创建 SNAT 规则，
只要是从这个网桥上出来的包，都会对其做源 IP 的转换，
保证了容器经过宿主机访问到宿主机外部网络请求的包转换成机器的 IP,从而能正确的送达和接收
//
*/
func setupIPTables(bridge string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridge)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	// 执行该命令
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return err
}

// enterContainerNetNS 将容器的网络端点加入到容器的网络空间中
// 并锁定当前程序所执行的线程，使当前线程进入到容器的网络空间
// 返回值是一个函数指针，执行这个返回函数才会退出容器的网络空间，回归到宿主机的网络空间
func enterContainerNetNS(enLink *netlink.Link, info *container.Info) func() {
	// 找到容器的Net Namespace
	// /proc/[pid]/ns/net 打开这个文件的文件描述符就可以来操作Net Namespace
	// 而ContainerInfo中的PID,即容器在宿主机上映射的进程ID
	// 它对应的/proc/[pid]/ns/net就是容器内部的Net Namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", info.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	// 锁定当前程序所执行的线程，如果不锁定操作系统线程的话
	// Go语言的goroutine可能会被调度到别的线程上去
	// 就不能保证一直在所需要的网络空间中了
	// 所以先调用runtime.LockOSThread()锁定当前程序执行的线程
	runtime.LockOSThread()

	// 修改网络端点Veth的另外一端，将其移动到容器的Net Namespace 中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("error set link netns , %v", err)
	}

	// 获取当前的网络namespace
	origns, err := netns.Get()
	if err != nil {
		log.Errorf("error get current netns, %v", err)
	}

	// 调用 netns.Set方法，将当前进程加入容器的Net Namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("error set netns, %v", err)
	}
	// 返回之前Net Namespace的函数
	// 在容器的网络空间中执行完容器配置之后调用此函数就可以将程序恢复到原生的Net Namespace
	return func() {
		// 恢复到上面获取到的之前的 Net Namespace
		netns.Set(origns)
		origns.Close()
		// 取消对当附程序的线程锁定
		runtime.UnlockOSThread()
		f.Close()
	}
}

// configEndpointIpAddressAndRoute 配置容器网络端点的地址和路由
func configEndpointIpAddressAndRoute(ep *Endpoint, info *container.Info) error {
	// 根据名字找到对应Veth设备
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}
	// 将容器的网络端点加入到容器的网络空间中
	// 并使这个函数下面的操作都在这个网络空间中进行
	// 执行完函数后，恢复为默认的网络空间，具体实现下面再做介绍

	defer enterContainerNetNS(&peerLink, info)()
	// 获取到容器的IP地址及网段，用于配置容器内部接口地址
	// 比如容器IP是192.168.1.2， 而网络的网段是192.168.1.0/24
	// 那么这里产出的IP字符串就是192.168.1.2/24，用于容器内Veth端点配置

	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress
	// 设置容器内Veth端点的IP
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}
	// 启动容器内的Veth端点
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}
	// Net Namespace 中默认本地地址 127 的网卡是关闭状态的
	// 启动它以保证容器访问自己的请求
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}
	// 设置容器内的外部请求都通过容器内的Veth端点访问
	// 0.0.0.0/0的网段，表示所有的IP地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	// 构建要添加的路由数据，包括网络设备、网关IP及目的网段
	// 相当于route add -net 0.0.0.0/0 gw (Bridge网桥地址) dev （容器内的Veth端点设备)

	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}
	// 调用netlink的RouteAdd,添加路由到容器的网络空间
	// RouteAdd 函数相当于route add 命令
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil
}

// configPortMapping 配置端口映射
func configPortMapping(ep *Endpoint) error {
	var err error
	// 遍历容器端口映射列表
	for _, pm := range ep.PortMapping {
		// 分割成宿主机的端口和容器的端口
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("port mapping format error, %v", pm)
			continue
		}
		// 由于iptables没有Go语言版本的实现，所以采用exec.Command的方式直接调用命令配置
		// 在iptables的PREROUTING中添加DNAT规则
		// 将宿主机的端口请求转发到容器的地址和端口上
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		log.Infoln("配置端口映射 cmd:", cmd.String())
		// 执行iptables命令,添加端口映射转发规则
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return err
}
