package container

import "errors"

var BlueprintMissingPathError error = errors.New("Missing blueprint file path")
var InvalidBlueprintError error = errors.New("Blueprint file provided is invalid. use the --help flag for help")
var MissingRunCommandsError error = errors.New("Missing run commands in json file")
var MissingWorkingDirectoryError error = errors.New("Missing working directory in json file")
var MissingCopyError error = errors.New("Missing copy commands in json file")
var CmdBuildFailedError error = errors.New("Command build failed")
var CmdRunFailedError error = errors.New("Command run failed")
var RootChangeFailedError error = errors.New("File root change failed")
var ResourceLimitError error = errors.New("Resource limit setup failed")
var InvalidFilePairError error = errors.New("Invalid copy structure, copy is in the format of <Source> <Target>")
var TargetDirectoryCreateError error = errors.New("Target directory creation failed")
var TargetPathCleanupError error = errors.New("Target path cleanup failed")
