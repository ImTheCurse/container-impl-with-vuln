package main

import (
	"flag"

	"github.com/ImTheCurse/container-impl-with-vuln/container"
)

func main() {
	blueprint := flag.String("blueprint", "", "Path of the blueprint instructions to build the container")
	containerChild := flag.Bool("container-child", false, "internal container child mode")

	flag.Parse()

	if *containerChild {
		if err := container.RunContainerChild(); err != nil {
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

	container := container.Container{}
	if err := container.BuildContainer(bp); err != nil {
		panic(err)
	}
}
