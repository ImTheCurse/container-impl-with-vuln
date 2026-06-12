package runtime

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

type ChildCommandConfig struct {
	Script     string
	WorkDir    string
	Hostname   string
	RootfsPath string
	StartRead  *os.File
}

func NewChildCommand(cfg ChildCommandConfig) *exec.Cmd {
	hostname := cfg.Hostname
	if hostname == "" {
		hostname = DefaultContainerHostname
	}

	cmd := exec.Command("/proc/self/exe", ChildModeFlag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startPipeFD := 3 + len(cmd.ExtraFiles)
	cmd.ExtraFiles = append(cmd.ExtraFiles, cfg.StartRead)
	cmd.Env = append(os.Environ(),
		ChildScriptEnv+"="+cfg.Script,
		ChildWorkDirEnv+"="+cfg.WorkDir,
		ChildHostnameEnv+"="+hostname,
		ChildRootfsPathEnv+"="+cfg.RootfsPath,
		ChildStartPipeFDEnv+"="+strconv.Itoa(startPipeFD),
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	return cmd
}
