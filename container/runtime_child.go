package container

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

type childRuntimeConfig struct {
	script      string
	workDir     string
	hostname    string
	rootfsPath  string
	startPipeFD int
}

func RunContainerChild() error {
	fmt.Printf("Running child process id: %v\n", os.Getpid())

	cfg, err := loadChildRuntimeConfig()
	if err != nil {
		return err
	}

	if err := waitForParentStartSignal(cfg.startPipeFD); err != nil {
		return err
	}

	cleanup, err := setupChildRuntime(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	return runPayloadScript(cfg.script)
}

func loadChildRuntimeConfig() (*childRuntimeConfig, error) {
	script := os.Getenv(childScriptEnv)
	if script == "" {
		return nil, fmt.Errorf("%w: %s", MissingChildConfigError, childScriptEnv)
	}

	workDir := os.Getenv(childWorkDirEnv)
	if workDir == "" {
		return nil, fmt.Errorf("%w: %s", MissingChildConfigError, childWorkDirEnv)
	}

	startPipeFDValue := os.Getenv(childStartPipeFDEnv)
	if startPipeFDValue == "" {
		return nil, fmt.Errorf("%w: %s", MissingChildConfigError, childStartPipeFDEnv)
	}
	startPipeFD, err := strconv.Atoi(startPipeFDValue)
	if err != nil || startPipeFD < 0 {
		return nil, fmt.Errorf("%w: %s=%q", MissingChildConfigError, childStartPipeFDEnv, startPipeFDValue)
	}

	hostname := os.Getenv(childHostnameEnv)
	if hostname == "" {
		hostname = defaultContainerHostname
	}

	rootfsPath := os.Getenv(childRootfsPathEnv)
	if rootfsPath == "" {
		return nil, fmt.Errorf("%w: %s", MissingChildConfigError, childRootfsPathEnv)
	}

	return &childRuntimeConfig{
		script:      script,
		workDir:     workDir,
		hostname:    hostname,
		rootfsPath:  rootfsPath,
		startPipeFD: startPipeFD,
	}, nil
}

func waitForParentStartSignal(startPipeFD int) error {
	startPipe := os.NewFile(uintptr(startPipeFD), "container-start-pipe")
	if startPipe == nil {
		return fmt.Errorf("%w: %s", MissingChildConfigError, childStartPipeFDEnv)
	}
	defer startPipe.Close()

	if _, err := startPipe.Read(make([]byte, 1)); err != nil {
		return fmt.Errorf("%w: start-sync read failed: %v", CmdRunFailedError, err)
	}
	return nil
}

func setupChildRuntime(cfg *childRuntimeConfig) (func(), error) {
	if err := syscall.Sethostname([]byte(cfg.hostname)); err != nil {
		return nil, fmt.Errorf("%w: %v", HostnameChangeError, err)
	}

	if err := syscall.Chroot(cfg.rootfsPath); err != nil {
		return nil, fmt.Errorf("%w: %v", RootChangeFailedError, err)
	}
	if err := os.Chdir("/"); err != nil {
		return nil, fmt.Errorf("%w: %v", DirChangeError, err)
	}
	if err := os.MkdirAll("/proc", 0555); err != nil {
		return nil, fmt.Errorf("%w: %v", ProcMountError, err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return nil, fmt.Errorf("%w: %v", ProcMountError, err)
	}

	if err := os.Chdir(cfg.workDir); err != nil {
		_ = syscall.Unmount("/proc", 0)
		return nil, fmt.Errorf("%w: %v", DirChangeError, err)
	}

	return func() {
		_ = syscall.Unmount("/proc", 0)
	}, nil
}

func runPayloadScript(script string) error {
	payload := exec.Command("/bin/bash", "-c", script)
	payload.Stdin = os.Stdin
	payload.Stdout = os.Stdout
	payload.Stderr = os.Stderr

	if err := payload.Run(); err != nil {
		return fmt.Errorf("%w: %v", CmdRunFailedError, err)
	}
	return nil
}
