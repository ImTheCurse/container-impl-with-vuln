package seccomp

import (
	"fmt"
	"math"

	"github.com/opencontainers/runtime-spec/specs-go"
	libseccomp "github.com/seccomp/libseccomp-golang"
)

// parseAction converts OCI specification actions to matching native libseccomp structural rules.
func parseAction(action specs.LinuxSeccompAction, errnoRet *uint) (libseccomp.ScmpAction, error) {
	var parsed libseccomp.ScmpAction

	switch action {
	case specs.ActAllow:
		parsed = libseccomp.ActAllow
	case specs.ActErrno:
		parsed = libseccomp.ActErrno
	case specs.ActKill:
		parsed = libseccomp.ActKillThread
	case specs.ActTrap:
		parsed = libseccomp.ActTrap
	case specs.ActLog:
		parsed = libseccomp.ActLog
	default:
		return libseccomp.ActInvalid, fmt.Errorf("unsupported action string variant parsed: %s", action)
	}

	if action == specs.ActErrno && errnoRet != nil {
		if *errnoRet > math.MaxInt16 {
			return libseccomp.ActInvalid, fmt.Errorf("invalid errnoRet %d: exceeds max supported value %d", *errnoRet, math.MaxInt16)
		}
		parsed = parsed.SetReturnCode(int16(*errnoRet))
	}

	return parsed, nil
}

// parseOp transforms OCI parameter query comparisons down into libseccomp bit operations.
func parseOp(op specs.LinuxSeccompOperator) (libseccomp.ScmpCompareOp, error) {
	switch op {
	case specs.OpEqualTo:
		return libseccomp.CompareEqual, nil
	case specs.OpNotEqual:
		return libseccomp.CompareNotEqual, nil
	case specs.OpGreaterThan:
		return libseccomp.CompareGreater, nil
	case specs.OpGreaterEqual:
		return libseccomp.CompareGreaterEqual, nil
	case specs.OpLessThan:
		return libseccomp.CompareLess, nil
	case specs.OpLessEqual:
		return libseccomp.CompareLessOrEqual, nil
	case specs.OpMaskedEqual:
		return libseccomp.CompareMaskedEqual, nil
	default:
		return libseccomp.CompareInvalid, fmt.Errorf("unsupported comparative condition sequence parsing: %s", op)
	}
}
