package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func (flags *ContainerFlags) Validate() error {
	if flags.BlueprintPath == nil {
		return BlueprintMissingPathError
	}
	return nil
}
func (bp *ContainerBlueprint) validate() error {
	if bp.BuildCommands == nil {
		return MissingRunCommandsError
	}
	if bp.FilesToCopyFromHost == nil {
		return MissingCopyError
	}
	if bp.WrkDir == nil {
		return MissingWorkingDirectoryError
	}
	return nil
}

func (flags *ContainerFlags) OpenBluePrint() (*ContainerBlueprint, error) {
	err := flags.Validate()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(*flags.BlueprintPath)
	if err != nil {
		return nil, err
	}

	blueprint := ContainerBlueprint{}
	err = json.Unmarshal(data, &blueprint)
	if err != nil {
		return nil, err
	}
	err = blueprint.validate()
	if err != nil {
		return nil, err
	}
	return &blueprint, nil
}

func (cont *Container) WithWorkingDirectory(workDir string) *Container {
	cont.cmd.Dir = workDir
	return cont
}

const (
	childModeFlag            = "--container-child"
	childScriptEnv           = "CONTAINER_SCRIPT"
	childWorkDirEnv          = "CONTAINER_WORKDIR"
	childHostnameEnv         = "CONTAINER_HOSTNAME"
	childRootfsPathEnv       = "CONTAINER_ROOTFS_PATH"
	defaultContainerHostname = "container"
)

func (cont *Container) BuildContainer(bp *ContainerBlueprint) error {
	if bp == nil {
		return InvalidBlueprintError
	}

	rootfsPath, err := filepath.Abs("rootfs")
	if err != nil {
		return fmt.Errorf("%w: %v", RootChangeFailedError, err)
	}

	script := "set -ex\n" + strings.Join(*bp.BuildCommands, "\n")
	cmd := exec.Command("/proc/self/exe", childModeFlag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		childScriptEnv+"="+script,
		childWorkDirEnv+"="+*bp.WrkDir,
		childHostnameEnv+"="+defaultContainerHostname,
		childRootfsPathEnv+"="+rootfsPath,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", CmdRunFailedError, err)
	}
	return nil
}

func RunContainerChild() error {
	script := os.Getenv(childScriptEnv)
	if script == "" {
		return fmt.Errorf("%w: %s", MissingChildConfigError, childScriptEnv)
	}

	workDir := os.Getenv(childWorkDirEnv)
	if workDir == "" {
		return fmt.Errorf("%w: %s", MissingChildConfigError, childWorkDirEnv)
	}

	hostname := os.Getenv(childHostnameEnv)
	if hostname == "" {
		hostname = defaultContainerHostname
	}
	if err := syscall.Sethostname([]byte(hostname)); err != nil {
		return fmt.Errorf("%w: %v", HostnameChangeError, err)
	}

	rootfsPath := os.Getenv(childRootfsPathEnv)
	if rootfsPath == "" {
		return fmt.Errorf("%w: %s", MissingChildConfigError, childRootfsPathEnv)
	}
	if err := syscall.Chroot(rootfsPath); err != nil {
		return fmt.Errorf("%w: %v", RootChangeFailedError, err)
	}
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("%w: %v", DirChangeError, err)
	}
	if err := os.Chdir(workDir); err != nil {
		return fmt.Errorf("%w: %v", DirChangeError, err)
	}

	payload := exec.Command("/bin/bash", "-c", script)
	payload.Stdin = os.Stdin
	payload.Stdout = os.Stdout
	payload.Stderr = os.Stderr

	if err := payload.Run(); err != nil {
		return fmt.Errorf("%w: %v", CmdRunFailedError, err)
	}
	return nil
}

func (cont *Container) Run() error {
	err := cont.cmd.Run()
	if err != nil {
		return CmdRunFailedError
	}
	return nil
}
