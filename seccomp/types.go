package seccomp

import "github.com/opencontainers/runtime-spec/specs-go"

// LocalProfile matches a simplified seccomp profile structure,
// using the host's native capabilities.
// for a more complete version of this, you should also compile the BPF using the
// different architectures supported by the host.
type LocalProfile struct {
	DefaultAction   specs.LinuxSeccompAction `json:"defaultAction"`
	DefaultErrnoRet *uint                    `json:"defaultErrnoRet,omitempty"`
	Syscalls        []specs.LinuxSyscall     `json:"syscalls"`
}
