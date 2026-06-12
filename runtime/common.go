package runtime

import "strings"

const (
	ChildModeFlag            = "--container-child"
	ChildScriptEnv           = "CONTAINER_SCRIPT"
	ChildWorkDirEnv          = "CONTAINER_WORKDIR"
	ChildHostnameEnv         = "CONTAINER_HOSTNAME"
	ChildRootfsPathEnv       = "CONTAINER_ROOTFS_PATH"
	ChildStartPipeFDEnv      = "CONTAINER_START_PIPE_FD"
	DefaultContainerHostname = "container"
)

func BuildContainerScript(commands []string) string {
	return "set -ex\n" + strings.Join(commands, "\n")
}
