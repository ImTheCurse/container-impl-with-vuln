package container

import "fmt"

func (bp *ContainerBlueprint) String() string {
	if bp == nil {
		return "ContainerBlueprint<nil>"
	}
	runCommands := ""
	if bp.BuildCommands != nil {
		runCommands = fmt.Sprintf("%v", *bp.BuildCommands)
	}
	filesToCopyFromHost := ""
	if bp.FilesToCopyFromHost != nil {
		filesToCopyFromHost = fmt.Sprintf("%v", *bp.FilesToCopyFromHost)
	}
	wrkDir := ""
	if bp.WrkDir != nil {
		wrkDir = fmt.Sprintf("%v", *bp.WrkDir)
	}
	return fmt.Sprintf("ContainerBlueprint{RunCommands:%s,FilesToCopyFromHost:%s,WrkDir:%s}", runCommands, filesToCopyFromHost, wrkDir)
}

func (flags *ContainerFlags) String() string {
	if flags == nil {
		return "ContainerFlags<nil>"
	}
	blueprintPath := ""
	if flags.BlueprintPath != nil {
		blueprintPath = fmt.Sprintf("%q", *flags.BlueprintPath)
	}

	return fmt.Sprintf("ContainerFlags{BlueprintPath:%s}", blueprintPath)
}
