package main

import (
	"flag"
	"math"

	"github.com/ImTheCurse/container-impl-with-vuln/container"
	containerruntime "github.com/ImTheCurse/container-impl-with-vuln/runtime"
)

func main() {
	blueprint := flag.String("blueprint", "", "Path of the blueprint instructions to build the container")
	containerChild := flag.Bool("container-child", false, "internal container child mode")
	memoryInGigabytes := flag.Int("memory", math.MaxInt, "Limit the amout of memory in the container in Gigabytes")
	maxProcesses := flag.Int("max-proc", math.MaxInt, "Limit the amount of procesess that the container can create")
	maxCpus := flag.Float64("max-cpu", math.MaxFloat64, "Limit the amount of CPUs that the container can use")

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

	container := container.Container{}
	if err := container.BuildContainer(bp, limits); err != nil {
		panic(err)
	}
}
