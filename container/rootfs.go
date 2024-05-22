package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"myDocker/constant"
	"myDocker/utils"
	"os"
	"os/exec"
	"path"
	"strings"
)

func NewWorkSpace(rootPath string, mntURL, volume string) {
	createLower(rootPath)
	createDirs(rootPath)
	mountOverlayFS(rootPath, mntURL)
	if volume != "" {
		mntPath := path.Join(rootPath, "merged")
		hostPath, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("extract volume failed err %v", err)
			return
		}
		mountVolume(mntPath, hostPath, containerPath)
	}
}

func mountVolume(mntPath string, hostPath string, containerPath string) {
	if err := os.Mkdir(hostPath, constant.Perm0777); err != nil {
		log.Infof("mkdir parent dir %s error. %v", hostPath, err)
	}
	containerPathInHost := path.Join(mntPath, containerPath)
	if err := os.Mkdir(containerPathInHost, constant.Perm0777); err != nil {
		log.Infof("mkdir container dir %s error. %v", containerPathInHost, err)
	}
	cmd := exec.Command("mount", "-o", "bind", hostPath, containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Infof("mount cvolume failed , error: %v", err)
	}
}

func volumeExtract(volume string) (sourcePath, destinationPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid volume [%s], must split by `:`", volume)
	}
	sourcePath, destinationPath = parts[0], parts[1]
	if sourcePath == "" || destinationPath == "" {
		return "", "", fmt.Errorf("invalid volume [%s], path can't be empty", volume)
	}

	return sourcePath, destinationPath, nil
}

func createLower(rootPath string) {
	busyboxURL := rootPath + "busybox/"
	busyboxTarURL := rootPath + "busybox.tar"

	/* 检查镜像文件已经存在 */
	// 检查是否已经存在busybox文件夹
	exist, err := utils.PathExists(busyboxURL)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", busyboxURL, err)
	}
	// 不存在则创建目录并将busybox.tar解压到busybox文件夹中
	if !exist {
		if err := os.Mkdir(busyboxURL, constant.Perm0777); err != nil {
			log.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("Untar dir %s error %v", busyboxURL, err)
		}
	}

}

// createDirs 创建overlayfs需要的的upper、worker目录
func createDirs(rootURL string) {
	upperURL := rootURL + "upper/"
	if err := os.Mkdir(upperURL, constant.Perm0777); err != nil {
		log.Errorf("mkdir dir %s error. %v", upperURL, err)
	}
	workURL := rootURL + "work/"
	if err := os.Mkdir(workURL, constant.Perm0777); err != nil {
		log.Errorf("mkdir dir %s error. %v", workURL, err)
	}
}

// mountOverlayFS 挂载overlayfs
func mountOverlayFS(rootURL string, mntURL string) {
	// mount -t overlay overlay -o lowerdir=lower1:lower2:lower3,upperdir=upper,workdir=work merged
	// 创建对应的挂载目录
	if err := os.Mkdir(mntURL, constant.Perm0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", mntURL, err)
	}
	// 拼接参数
	// e.g. lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/merged
	dirs := "lowerdir=" + rootURL + "busybox" + ",upperdir=" + rootURL + "upper" + ",workdir=" + rootURL + "work"
	// dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"

	/*执行挂载到mnt目录*/
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

/*
	docker会在删除容器时，删除掉读写层，容器初始化的init layer ；留下镜像的所有内容，本容器在删除容器时，会删除upper层、merge层、work目录

在容器退出时会删除容器：
步骤：解除挂载、删除文件
*/
func DeleteWorkSpace(rootURL, volume string) {
	mntURL := rootURL + "merged/"
	if volume != "" {
		_, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("extract volume failed err %v", err)
			return
		}
		umountVolume(mntURL, containerPath)
	}

	umountOverlayFS(mntURL)
	deleteDirs(rootURL)
}

func umountVolume(mntURL string, containerPath string) {
	containerPathInHost := path.Join(mntURL, containerPath)
	cmd := exec.Command("umount", containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount volume failed err %v", err)
	}
}

func umountOverlayFS(mntURL string) {
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("remove dir %s error :%v", mntURL, err)
	}
}

func deleteDirs(rootURL string) {
	writeURL := rootURL + "upper/"
	if err := os.RemoveAll(writeURL); err != nil {
		log.Errorf("remove dir %s error :%v", writeURL, err)
	}
	workURL := rootURL + "work"
	if err := os.RemoveAll(workURL); err != nil {
		log.Errorf("remove dir %s error :%v", workURL, err)
	}
}
