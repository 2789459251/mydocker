package network

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
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

// setupIPTables 设置 iptables 对应 bridge MASQUERADE 规则
//func setupIPTables(bridgeName string, subnet *net.IPNet) error {
//	// 拼接命令
//	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
//	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
//	// 执行该命令
//	output, err := cmd.Output()
//	if err != nil {
//		log.Errorf("iptables Output, %v", output)
//	}
//	return err
//}
