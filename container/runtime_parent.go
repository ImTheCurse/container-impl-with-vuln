package container

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
)

func (cont *Container) BuildContainer(bp *ContainerBlueprint, limits *ContainerResourcesLimit) (retErr error) {
	if bp == nil {
		return InvalidBlueprintError
	}

	rootfsPath, err := filepath.Abs("rootfs")
	if err != nil {
		return fmt.Errorf("%w: %v", RootChangeFailedError, err)
	}

	startRead, startWrite, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("%w: %v", CmdBuildFailedError, err)
	}
	defer startWrite.Close()

	cmd := cont.newChildCommand(bp, rootfsPath, startRead)
	if err := cmd.Start(); err != nil {
		_ = startRead.Close()
		return fmt.Errorf("%w: %v", CmdRunFailedError, err)
	}
	_ = startRead.Close()

	cleanupCgroup, err := attachResourceLimits(cmd, limits)
	if err != nil {
		return err
	}

	defer func() {
		if err := cleanupCgroup(); err != nil && retErr == nil {
			retErr = err
		}
	}()

	if err := releaseChildStartSignal(startWrite, cmd); err != nil {
		return err
	}

	return waitForCommandWithSignals(cmd)
}

func (cont *Container) newChildCommand(bp *ContainerBlueprint, rootfsPath string, startRead *os.File) *exec.Cmd {
	script := buildContainerScript(*bp.BuildCommands)
	cmd := exec.Command("/proc/self/exe", childModeFlag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startPipeFD := 3 + len(cmd.ExtraFiles)
	cmd.ExtraFiles = append(cmd.ExtraFiles, startRead)
	cmd.Env = append(os.Environ(),
		childScriptEnv+"="+script,
		childWorkDirEnv+"="+*bp.WrkDir,
		childHostnameEnv+"="+defaultContainerHostname,
		childRootfsPathEnv+"="+rootfsPath,
		childStartPipeFDEnv+"="+strconv.Itoa(startPipeFD),
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	return cmd
}

func attachResourceLimits(cmd *exec.Cmd, limits *ContainerResourcesLimit) (func() error, error) {
	cleanupCgroup := func() error { return nil }
	if limits == nil {
		return cleanupCgroup, nil
	}

	cleanup, err := limits.ApplyWithCgroups(cmd.Process.Pid)
	if err != nil {
		killAndWait(cmd)
		return nil, err
	}
	return cleanup, nil
}

func releaseChildStartSignal(startWrite *os.File, cmd *exec.Cmd) error {
	if _, err := startWrite.Write([]byte{1}); err != nil {
		killAndWait(cmd)
		return fmt.Errorf("%w: %v", CmdRunFailedError, err)
	}
	_ = startWrite.Close()
	return nil
}

func waitForCommandWithSignals(cmd *exec.Cmd) error {
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	select {
	case waitErr := <-waitCh:
		if waitErr != nil {
			return fmt.Errorf("%w: %v", CmdRunFailedError, waitErr)
		}
		return nil
	case sig := <-sigCh:
		_ = cmd.Process.Kill()
		waitErr := <-waitCh
		if waitErr != nil {
			return fmt.Errorf("%w: interrupted by signal %v: %v", CmdRunFailedError, sig, waitErr)
		}
		return fmt.Errorf("%w: interrupted by signal %v", CmdRunFailedError, sig)
	}
}

func killAndWait(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}
