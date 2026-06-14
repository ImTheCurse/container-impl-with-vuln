package runtime

import (
	"errors"
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

	procMounted := false
	devMounted := false
	cleanup := func() {
		if devMounted {
			_ = syscall.Unmount("/dev", 0)
		}
		if procMounted {
			_ = syscall.Unmount("/proc", 0)
		}
	}

	setupCompleted := false
	defer func() {
		if !setupCompleted {
			cleanup()
		}
	}()

	if err := os.MkdirAll("/proc", 0o555); err != nil {
		return nil, fmt.Errorf("%w: %v", ProcMountError, err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return nil, fmt.Errorf("%w: %v", ProcMountError, err)
	}
	procMounted = true

	if err := setupMountDev(); err != nil {
		return nil, err
	}
	devMounted = true

	if err := os.Chdir(cfg.workDir); err != nil {
		return nil, fmt.Errorf("%w: %v", DirChangeError, err)
	}

	setupCompleted = true
	return cleanup, nil
}

func setupMountDev() error {
	if err := os.MkdirAll("/dev", 0o755); err != nil {
		return fmt.Errorf("%w: %v", DevMountError, err)
	}

	if err := syscall.Mount("tmpfs", "/dev", "tmpfs", uintptr(syscall.MS_NOSUID|syscall.MS_STRICTATIME), "mode=755,size=16m"); err != nil {
		return fmt.Errorf("%w: %v", DevMountError, err)
	}

	mounted := true
	defer func() {
		if mounted {
			_ = syscall.Unmount("/dev", 0)
		}
	}()

	type devNode struct {
		path  string
		mode  uint32
		major uint32
		minor uint32
	}

	devNodes := []devNode{
		{path: "/dev/null", mode: 0o666, major: 1, minor: 3},
		{path: "/dev/zero", mode: 0o666, major: 1, minor: 5},
		{path: "/dev/full", mode: 0o666, major: 1, minor: 7},
		{path: "/dev/random", mode: 0o666, major: 1, minor: 8},
		{path: "/dev/urandom", mode: 0o666, major: 1, minor: 9},
		{path: "/dev/tty", mode: 0o666, major: 5, minor: 0},
	}

	for _, node := range devNodes {
		dev := int(linuxMakedev(node.major, node.minor))
		mode := syscall.S_IFCHR | node.mode
		if err := syscall.Mknod(node.path, mode, dev); err != nil && !errors.Is(err, syscall.EEXIST) {
			return fmt.Errorf("%w: mknod %s failed: %v", DevMountError, node.path, err)
		}
	}

	mounted = false
	return nil
}

func linuxMakedev(major, minor uint32) uint64 {
	return (uint64(major&0xfff) << 8) |
		(uint64(minor & 0xff)) |
		(uint64(minor&^uint32(0xff)) << 12)
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
