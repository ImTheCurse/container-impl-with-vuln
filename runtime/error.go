package runtime

import "errors"

var MissingChildConfigError error = errors.New("Missing child process config")
var CmdRunFailedError error = errors.New("Command run failed")
var RootChangeFailedError error = errors.New("File root change failed")
var DirChangeError error = errors.New("Directory change failed")
var HostnameChangeError error = errors.New("Hostname change failed")
var ProcMountError error = errors.New("Proc mount failed")
var DNSConfigError error = errors.New("DNS configuration failed")
