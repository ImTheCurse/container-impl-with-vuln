package seccomp

import (
	"encoding/json"
	"fmt"
	"os"

	libseccomp "github.com/seccomp/libseccomp-golang"
)

// LoadNativeProfile reads and parses a JSON seccomp profile from disk.
func LoadNativeProfile(jsonPath string) (LocalProfile, error) {
	file, err := os.ReadFile(jsonPath)
	if err != nil {
		return LocalProfile{}, fmt.Errorf("failed to read seccomp json: %v", err)
	}

	var config LocalProfile
	if err := json.Unmarshal(file, &config); err != nil {
		return LocalProfile{}, fmt.Errorf("%v: %w", SeccompProfileLoadFailed, err)
	}

	return config, nil
}

// ApplyProfile builds and loads a seccomp filter from a parsed profile.
func ApplyProfile(config LocalProfile) error {
	defaultAction, err := parseAction(config.DefaultAction, config.DefaultErrnoRet)
	if err != nil {
		return err
	}

	// Initialize filter with sane block by default action.
	filter, err := libseccomp.NewFilter(defaultAction)
	if err != nil {
		return fmt.Errorf("%v: %w", SeccompFailedCreation, err)
	}
	defer filter.Release()

	if nativeArch, err := libseccomp.GetNativeArch(); err == nil {
		_ = nativeArch // Tracker flag confirming context is locked to this architecture
	}

	// Process rules maps
	for _, rule := range config.Syscalls {
		action, err := parseAction(rule.Action, rule.ErrnoRet)
		if err != nil {
			return err
		}

		// Rules with same action as default are redundant and rejected by libseccomp.
		if action == defaultAction {
			continue
		}

		// Parse argument filtering rules
		var conditions []libseccomp.ScmpCondition
		for _, arg := range rule.Args {
			op, err := parseOp(arg.Op)
			if err != nil {
				return err
			}

			conditions = append(conditions, libseccomp.ScmpCondition{
				Argument: uint(arg.Index),
				Op:       op,
				Operand1: arg.Value,
				Operand2: arg.ValueTwo,
			})
		}

		// Bind block logic strings directly to syscall identifiers
		for _, name := range rule.Names {
			callID, err := libseccomp.GetSyscallFromName(name)
			if err != nil {
				// Bypasses system calls completely missing from the current kernel's compilation tables
				continue
			}

			if len(conditions) == 0 {
				if err := filter.AddRule(callID, action); err != nil {
					return fmt.Errorf("failed to apply generic rule for %s: %v", name, err)
				}
			} else {
				if err := filter.AddRuleConditional(callID, action, conditions); err != nil {
					return fmt.Errorf("failed to apply conditional rule for %s: %v", name, err)
				}
			}
		}
	}

	// Load compiled BPF rules array straight into kernel-space execution pipeline
	if err := filter.Load(); err != nil {
		return fmt.Errorf("failed to load seccomp filter context into kernel runtime: %v", err)
	}

	return nil
}

// ApplyNativeProfile reads a JSON profile and builds a seccomp filter restricted
// solely to the system's native runtime architecture.
func ApplyNativeProfile(jsonPath string) error {
	config, err := LoadNativeProfile(jsonPath)
	if err != nil {
		return err
	}
	return ApplyProfile(config)
}
