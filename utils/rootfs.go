package utils

import "fmt"

const (
	ImagePath       = "/var/lib/mydocker/image/"
	RootPath        = "/var/lib/mydocker/overlay2/"
	lowerDirFormat  = RootPath + "%s/lower"
	upperDirFormat  = RootPath + "%s/upper"
	workDirFormat   = RootPath + "%s/work"
	mergedDirFormat = RootPath + "%s/merged"
	overlayFSFormat = "lowerdir=%s,upperdir=%s,workdir=%s"
)

func GetMerged(ContainerId string) string {
	return fmt.Sprintf(mergedDirFormat, ContainerId)
}
func GetImage(ImageName string) string {
	return fmt.Sprintf("%s%s.tar", ImagePath, ImageName)
}
func GetLower(ContainerId string) string {
	return fmt.Sprintf(lowerDirFormat, ContainerId)
}
func GetUpper(containerId string) string {
	return fmt.Sprintf(upperDirFormat, containerId)
}
func GetWorker(ContainerId string) string {
	return fmt.Sprintf(workDirFormat, ContainerId)
}
func GetOverlayFSDirs(lowerDir string, upperDir string, workDir string) string {
	return fmt.Sprintf(overlayFSFormat, lowerDir, upperDir, workDir)
}
func GetRoot(ContainerId string) string {
	return RootPath + ContainerId
}
