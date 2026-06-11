package container

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	cgroupRoot  = "/sys/fs/cgroup"
	cpuPeriodUS = 100000
)

func (limit *ContainerResourcesLimit) hasAnyLimit() bool {
	if limit == nil {
		return false
	}

	hasMem := limit.MaxMemory != math.MaxInt
	hasPid := limit.MaxPid != math.MaxInt
	hasCPU := limit.MaxCpu != math.MaxFloat64
	return hasMem || hasPid || hasCPU
}

// ApplyWithCgroups applies configured limits using Linux cgroups v2 and adds the
// process to the created cgroup. It returns a cleanup function that removes the
// cgroup directory after process exit.
func (limit *ContainerResourcesLimit) ApplyWithCgroups(pid int) (func() error, error) {
	if !limit.hasAnyLimit() {
		return func() error { return nil }, nil
	}

	if _, err := os.Stat(filepath.Join(cgroupRoot, "cgroup.controllers")); err != nil {
		return nil, fmt.Errorf("%w: cgroups v2 not available: %v", ResourceLimitError, err)
	}

	groupName := fmt.Sprintf("container-%d-%d", pid, time.Now().UnixNano())
	groupPath := filepath.Join(cgroupRoot, groupName)

	if err := os.Mkdir(groupPath, 0o755); err != nil {
		return nil, fmt.Errorf("%w: failed creating cgroup %q: %v", ResourceLimitError, groupPath, err)
	}

	cleanup := func() error {
		if err := os.Remove(groupPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("%w: failed cleaning cgroup %q: %v", ResourceLimitError, groupPath, err)
		}
		return nil
	}

	if limit.MaxMemory > 0 && limit.MaxMemory != math.MaxInt {
		memoryLimitBytes := int64(limit.MaxMemory) * 1024 * 1024 * 1024
		if memoryLimitBytes <= 0 {
			_ = cleanup()
			return nil, fmt.Errorf("%w: invalid memory limit %dGiB", ResourceLimitError, limit.MaxMemory)
		}
		if err := os.WriteFile(filepath.Join(groupPath, "memory.max"), []byte(strconv.FormatInt(memoryLimitBytes, 10)), 0o644); err != nil {
			_ = cleanup()
			return nil, fmt.Errorf("%w: setting memory.max failed: %v", ResourceLimitError, err)
		}
	}

	if limit.MaxPid > 0 && limit.MaxPid != math.MaxInt {
		if err := os.WriteFile(filepath.Join(groupPath, "pids.max"), []byte(strconv.Itoa(int(limit.MaxPid))), 0o644); err != nil {
			_ = cleanup()
			return nil, fmt.Errorf("%w: setting pids.max failed: %v", ResourceLimitError, err)
		}
	}

	if limit.MaxCpu > 0 && limit.MaxCpu != math.MaxFloat64 {
		cpuQuota := int64(limit.MaxCpu * cpuPeriodUS)
		if cpuQuota < 1 {
			_ = cleanup()
			return nil, fmt.Errorf("%w: invalid cpu limit %f", ResourceLimitError, limit.MaxCpu)
		}
		cpuMax := strings.Join([]string{strconv.FormatInt(cpuQuota, 10), strconv.Itoa(cpuPeriodUS)}, " ")
		if err := os.WriteFile(filepath.Join(groupPath, "cpu.max"), []byte(cpuMax), 0o644); err != nil {
			_ = cleanup()
			return nil, fmt.Errorf("%w: setting cpu.max failed: %v", ResourceLimitError, err)
		}
	}

	if err := os.WriteFile(filepath.Join(groupPath, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o644); err != nil {
		_ = cleanup()
		return nil, fmt.Errorf("%w: adding pid %d to cgroup failed: %v", ResourceLimitError, pid, err)
	}

	return cleanup, nil
}
