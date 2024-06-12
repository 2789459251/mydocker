package network

import (
	"errors"
	"myDocker/container"
)

func ConnectContain(network string, containerInfo *container.Info) error {
	networks, err := loadNetwork()
	if err != nil {
		return err
	}
	net, ok := networks[network]
	if !ok {
		return errors.New("network not exist")
	}

	//分配容器ip地址
	ip, err := ipAllocator.Allocate(net.IpRange)
	if err != nil {
		return errors.New("allocate ip error")
	}
	//创建网络端点
	ep := &Endpoint{
		ID:          containerInfo.Id,
		IPAddress:   ip,
		PortMapping: containerInfo.PortMapping,
		Network:     net,
	}

	// 调用网络驱动挂载和配置网络端点
	err = drivers[net.Driver].Connect(net, ep)
	if err != nil {
		return err
	}

	// 到容器的namespace配置容器网络设备IP地址
	if err = configEndpointIpAddressAndRoute(ep, containerInfo); err != nil {
		return err
	}
	// 配置端口映射信息，例如 mydocker run -p 8080:80
	return configPortMapping(ep)
}
