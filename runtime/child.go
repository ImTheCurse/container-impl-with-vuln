package runtime

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
	script := os.Getenv(ChildScriptEnv)
	if script == "" {
		return nil, fmt.Errorf("%w: %s", MissingChildConfigError, ChildScriptEnv)
	}

	workDir := os.Getenv(ChildWorkDirEnv)
	if workDir == "" {
		return nil, fmt.Errorf("%w: %s", MissingChildConfigError, ChildWorkDirEnv)
	}

	startPipeFDValue := os.Getenv(ChildStartPipeFDEnv)
	if startPipeFDValue == "" {
		return nil, fmt.Errorf("%w: %s", MissingChildConfigError, ChildStartPipeFDEnv)
	}
	startPipeFD, err := strconv.Atoi(startPipeFDValue)
	if err != nil || startPipeFD < 0 {
		return nil, fmt.Errorf("%w: %s=%q", MissingChildConfigError, ChildStartPipeFDEnv, startPipeFDValue)
	}

	hostname := os.Getenv(ChildHostnameEnv)
	if hostname == "" {
		hostname = DefaultContainerHostname
	}

	rootfsPath := os.Getenv(ChildRootfsPathEnv)
	if rootfsPath == "" {
		return nil, fmt.Errorf("%w: %s", MissingChildConfigError, ChildRootfsPathEnv)
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
		return fmt.Errorf("%w: %s", MissingChildConfigError, ChildStartPipeFDEnv)
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
	if err := configureStaticDNS(); err != nil {
		return nil, err
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

func configureStaticDNS() error {
	if info, err := os.Lstat("/etc/resolv.conf"); err == nil {
		// checks if /etc/resolv.conf is a symbolic link
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: /etc/resolv.conf is a symlink : %v", DNSConfigError, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("%w: failed checking /etc/resolv.conf: %v", DNSConfigError, err)
	}

	const resolvConf = "nameserver 1.1.1.1\nnameserver 8.8.8.8\n"
	if err := os.WriteFile("/etc/resolv.conf", []byte(resolvConf), 0o644); err != nil {
		return fmt.Errorf("%w: failed writing /etc/resolv.conf: %v", DNSConfigError, err)
	}
	return nil
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
