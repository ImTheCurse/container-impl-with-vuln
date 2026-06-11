package container

import "os/exec"

type ContainerConfigCommand uint

const (
	CopyFileCommand ContainerConfigCommand = iota
	RunCommand
	WrkDirCommand
)

type ContainerFlags struct {
	BlueprintPath *string
}

type ContainerBlueprint struct {
	// Commands to run after container isolation, the commands run inside the container.
	BuildCommands *[]string `json:"buildCommands,omitempty"`

	// Files to copy from the host to the container.
	FilesToCopyFromHost *[]string `json:"copy,omitempty"`

	// Working directory inside the container
	WrkDir *string `json:"workingDirectory,omitempty"`
}

type Container struct {
	// Command to run inside the container.
	cmd *exec.Cmd
	// Error of the command, remove it?
	err error
}

type ContainerResourcesLimit struct {
	MaxMemory uint
	MaxPid    uint
	MaxCpu    float64
}
