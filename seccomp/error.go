package seccomp

import "fmt"

var SeccompProfileLoadFailed = fmt.Errorf("failed to load seccomp profile")
var SeccompFailedCreation = fmt.Errorf("failed to create seccomp filter")
