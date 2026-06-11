package container

import (
	"encoding/json"
	"os"
)

func (flags *ContainerFlags) Validate() error {
	if flags.BlueprintPath == nil {
		return BlueprintMissingPathError
	}
	return nil
}

func (bp *ContainerBlueprint) validate() error {
	if bp.BuildCommands == nil {
		return MissingRunCommandsError
	}
	if bp.FilesToCopyFromHost == nil {
		return MissingCopyError
	}
	if bp.WrkDir == nil {
		return MissingWorkingDirectoryError
	}
	return nil
}

func (flags *ContainerFlags) OpenBluePrint() (*ContainerBlueprint, error) {
	err := flags.Validate()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(*flags.BlueprintPath)
	if err != nil {
		return nil, err
	}

	blueprint := ContainerBlueprint{}
	err = json.Unmarshal(data, &blueprint)
	if err != nil {
		return nil, err
	}
	err = blueprint.validate()
	if err != nil {
		return nil, err
	}
	return &blueprint, nil
}
