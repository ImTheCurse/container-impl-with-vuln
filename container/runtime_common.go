package container

import "strings"

const (
	childModeFlag            = "--container-child"
	childScriptEnv           = "CONTAINER_SCRIPT"
	childWorkDirEnv          = "CONTAINER_WORKDIR"
	childHostnameEnv         = "CONTAINER_HOSTNAME"
	childRootfsPathEnv       = "CONTAINER_ROOTFS_PATH"
	childStartPipeFDEnv      = "CONTAINER_START_PIPE_FD"
	defaultContainerHostname = "container"
)

func buildContainerScript(commands []string) string {
	return "set -ex\n" + strings.Join(commands, "\n")
}
