package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ImTheCurse/container-impl-with-vuln/archive"
	"github.com/ImTheCurse/container-impl-with-vuln/container"
	containerruntime "github.com/ImTheCurse/container-impl-with-vuln/runtime"
)

func main() {
	blueprint := flag.String("blueprint", "", "Path of the blueprint instructions to build the container")
	containerChild := flag.Bool("container-child", false, "internal container child mode")
	memoryInGigabytes := flag.Int("memory", math.MaxInt, "Limit the amout of memory in the container in Gigabytes")
	maxProcesses := flag.Int("max-proc", math.MaxInt, "Limit the amount of procesess that the container can create")
	maxCpus := flag.Float64("max-cpu", math.MaxFloat64, "Limit the amount of CPUs that the container can use")
	fsPath := flag.String("fs", "ubuntu-fs.tar", "Path of the filesystem to use as the rootfs")

	flag.Parse()

	if *containerChild {
		if err := containerruntime.RunContainerChild(); err != nil {
			panic(err)
		}
		return
	}

	containerFlags := &container.ContainerFlags{
		BlueprintPath: blueprint,
	}
	bp, err := containerFlags.OpenBluePrint()
	if err != nil {
		panic(err)
	}

	limits := &container.ContainerResourcesLimit{
		MaxMemory: uint(*memoryInGigabytes),
		MaxPid:    uint(*maxProcesses),
		MaxCpu:    *maxCpus,
	}

	rootfsPath, err := archive.ExtractFileSystem(*fsPath)
	if err != nil {
		panic(err)
	}
	fmt.Println("rootfsPath:", rootfsPath)
	defer func() {
		if err := cleanupExtractedRootfs(rootfsPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed cleaning extracted rootfs %q: %v\n", rootfsPath, err)
		}
	}()

	container := container.Container{
		RootfsPath: rootfsPath,
	}
	if err := container.BuildContainer(bp, limits); err != nil {
		panic(err)
	}
}

func cleanupExtractedRootfs(rootfsPath string) error {
	for _, mountpoint := range []string{"proc", "dev"} {
		target := filepath.Join(rootfsPath, mountpoint)
		if err := syscall.Unmount(target, syscall.MNT_DETACH); err != nil && err != syscall.EINVAL && err != syscall.ENOENT {
			return err
		}
	}

	extractDir := filepath.Dir(rootfsPath)
	if filepath.Base(rootfsPath) == "rootfs" {
		return os.RemoveAll(extractDir)
	}

	return os.RemoveAll(rootfsPath)
}
