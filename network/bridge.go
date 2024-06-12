package network

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"net"
)

type BridgeNetworkDriver struct{}

func (b *BridgeNetworkDriver) Name() string {
	return "bridge"
}

//
//Delete(name string) error
////连接容器网络端点到网络
//Connect(network *Network, endpoint *Endpoint) error
////从网络中移除容器网络端点
//Disconnect(network Network, endpoint *Endpoint) error

/* 创建网桥 */
func (b *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip

	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  b.Name(),
	}

	err := b.initBrigde(n)
	if err != nil {
		return nil, errors.Wrap(err, "bridge network creation failed")
	}
	return n, nil
}

// 删除网络
func (b *BridgeNetworkDriver) Delete(name string) error {
	br, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	/*删除设备*/
	return netlink.LinkDel(br)
}

/*是将 Endpoint 连接到当前指定网络*/
func (b *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	//brctl addif bxxx vXXX
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	//创建Veth接口配置
	la := netlink.NewLinkAttrs()
	// 由于 Linux 接口名的限制,取 endpointID 的前5位
	la.Name = endpoint.ID[:5]
	// 通过设置 Veth 接口 master 属性，设置这个Veth的一端挂载到网络对应的 Linux Bridge
	la.MasterIndex = br.Attrs().Index //->挂载bridge
	// 创建 Veth 对象，通过 PeerNarne 配置 Veth 另外 端的接口名
	// 配置 Veth 另外 端的名字 cif {endpoint ID 的前 位｝
	endpoint.Device = netlink.Veth{LinkAttrs: la, PeerName: "cif-" + endpoint.ID[:5]} //连接endpoint
	// 调用netlink的LinkAdd方法创建出这个Veth接口
	// 因为上面指定了link的MasterIndex是网络对应的Linux Bridge
	// 所以Veth的一端就已经挂载到了网络对应的LinuxBridge.上
	if err := netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error Add Endpoint Device: %v", err)
	}
	//调用netlink的LinkSetUp方法，设置Veth启动
	// 相当于ip link set xxx up命令
	if err := netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("error Set Up Endpoint Device: %v", err)
	}
	return nil
}

func (b *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	vethNme := endpoint.ID[:5]
	veth, err := netlink.LinkByName(vethNme)
	if err != nil {
		return err
	}
	if err := netlink.LinkSetNoMaster(veth); err != nil {
		return err
	}
	//err = netlink.LinkDel(veth)
	//if err != nil {
	//	return err
	//}
	return nil
}

func (b *BridgeNetworkDriver) initBrigde(n *Network) error {
	bridgeName := n.Name
	//创建bridge网络设备
	if err := createBridgeInterface(bridgeName); err != nil {
		return errors.Wrap(err, "bridge network creation failed")
	}

	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP
	// 设置 Bridge 设备地址和路由
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return errors.Wrapf(err, "Error set bridge ip: %s on bridge: %s", gatewayIP.String(), bridgeName)
	}

	// 3）启动 Bridge 设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return errors.Wrapf(err, "Failed to set %s up", bridgeName)
	}

	// 4）设置 iptables SNAT 规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return errors.Wrapf(err, "Failed to set up iptables for %s", bridgeName)
	}
	return nil
}
