package network

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"myDocker/constant"
	"net"
	"os"
	"path"
	"path/filepath"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/lib/mydocker/network/network/"
	drivers            = map[string]Driver{}
)

/*命令来创建网络*/
func CreateNetwork(driver, subnet, name string) error {
	_, cidr, _ := net.ParseCIDR(subnet)

	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip
	network, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	fmt.Println(network)
	return network.dump(defaultNetworkPath)
}

/*ListNetwork命令来展示所有网络*/
func ListNetwork() {
	// ListNetwork 打印出当前全部 Network 信息
	networks, err := loadNetwork()
	if err != nil {
		logrus.Errorf("load network from file failed,detail: %v", err)
		return
	}
	// 通过tabwriter库把信息打印到屏幕上
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, net := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			net.Name,
			net.IpRange.String(),
			net.Driver,
		)
	}
	if err = w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}

}

func DeleteNetwork(networkName string) error {
	networks, err := loadNetwork()
	if err != nil {
		return err
	}
	net, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("network %s not found", networkName)
	}
	//释放网关ip
	err = ipAllocator.Release(net.IpRange, &net.IpRange.IP)
	if err != nil {
		return err
	}
	// 调用网络驱动删除网络创建的设备与配置 后面会以 Bridge 驱动删除网络为例子介绍如何实现网络驱动删除网络
	if err = drivers[net.Driver].Delete(net.Name); err != nil {
		return errors.Wrap(err, "remove Network DriverError failed")
	}

	return net.Remove(defaultNetworkPath)
}

/*将net信息持久化到文件中*/
//func (net *Network) dump(dumpPath string) error {
//	if _, err := os.Stat(dumpPath); err != nil {
//		if !os.IsNotExist(err) {
//			return err
//		}
//		if err := os.MkdirAll(dumpPath, constant.Perm0644); err != nil {
//			return err
//		}
//	}
//	netPath := path.Join(dumpPath, net.Name)
//	//if err := os.MkdirAll(netPath, constant.Perm0644); err != nil {
//	//	return err
//	//}
//
//	//后面的参数分别代表如果表存在删除表，只写，如果不存在就创建
//	netFile, err := os.OpenFile(netPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, constant.Perm0644)
//	if err != nil {
//		return errors.Wrapf(err, "open file %s failed", dumpPath)
//	}
//	defer netFile.Close()
//
//	netJson, err := json.Marshal(net)
//	if err != nil {
//		return errors.Wrapf(err, "json marshal failed")
//	}
//	_, err = netFile.Write(netJson)
//	if err != nil {
//		return errors.Wrap(err, "write file failed")
//	}
//	return nil
//}
func (net *Network) dump(dumpPath string) error {
	// 检查保存的目录是否存在，不存在则创建
	if _, err := os.Stat(dumpPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(dumpPath, constant.Perm0644); err != nil {
			return errors.Wrapf(err, "create network dump path %s failed", dumpPath)
		}
	}
	// 保存的文件名是网络的名字
	netPath := path.Join(dumpPath, net.Name)

	netFile, err := os.OpenFile(netPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, constant.Perm0644)
	if err != nil {
		// 确保使用正确的变量名和路径变量
		fmt.Println(netPath)                                     // 假设这里需要打印 netPath
		return errors.Wrapf(err, "open file %s failed", netPath) // 使用 netPath 而不是 dumpPath
	}

	netJson, err := json.Marshal(net)
	if err != nil {
		return errors.Wrapf(err, "Marshal %v failed", net)
	}

	_, err = netFile.Write(netJson)
	return errors.Wrapf(err, "write %s failed", netJson)
}

/*读取文件到net中*/
func (net *Network) load(dumpPath string) error {
	netFile, err := os.Open(dumpPath)
	if err != nil {
		return errors.Wrapf(err, "open file %s failed", dumpPath)
	}
	defer netFile.Close()

	netJson := make([]byte, 2000)
	n, err := netFile.Read(netJson)
	if err != nil {
		return errors.Wrapf(err, "read file %s failed", dumpPath)
	}
	err = json.Unmarshal(netJson[:n], &net)
	return nil
}

/*删除net的配置文件*/
func (net *Network) Remove(dumpPath string) error {
	// 检查网络对应的配置文件状态，如果文件己经不存在就直接返回
	fullPath := path.Join(dumpPath, net.Name)
	if _, err := os.Stat(fullPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	// 否则删除这个网络对应的配置文件
	return os.Remove(fullPath)
}

/*加载所有的network*/
func loadNetwork() (map[string]*Network, error) {
	networks := map[string]*Network{}

	//1.从指定的根目录开始。
	//2.调用回调函数处理根目录。
	//3.如果根目录中包含子目录，Walk 将递归地进入每个子目录。
	//4.对于每个子目录，重复步骤 2 和 3，直到所有子目录都被访问。
	err := filepath.Walk(defaultNetworkPath, func(netPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		netName := path.Base(netPath)
		n := &Network{
			Name: netName,
		}
		err = n.load(netPath)
		if err != nil {
			logrus.Errorf("error load network: %s", err)
		}
		networks[netName] = n
		return nil
	})
	return networks, err
}
