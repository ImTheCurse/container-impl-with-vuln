package container

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func (bp *ContainerBlueprint) CopyFiles() error {
	for _, pathPair := range *bp.FilesToCopyFromHost {
		mapping, err := parseSrcTargetFile(pathPair)
		if err != nil {
			return err
		}

		fin, err := os.Open(mapping.src)
		if err != nil {
			return err
		}
		defer fin.Close()

		targetInImage := mapping.target
		if !filepath.IsAbs(targetInImage) {
			targetInImage = filepath.Join(*bp.WrkDir, targetInImage)
		}
		targetInImage = filepath.Clean(targetInImage)

		if !filepath.IsAbs(targetInImage) {
			targetInImage = string(os.PathSeparator) + targetInImage
		}

		targetPath := filepath.Join("rootfs", strings.TrimPrefix(targetInImage, string(os.PathSeparator)))
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return TargetDirectoryCreateError
		}
		if info, err := os.Stat(targetPath); err == nil {
			if info.IsDir() {
				if err := os.RemoveAll(targetPath); err != nil {
					return TargetPathCleanupError
				}
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		fout, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		defer fout.Close()
		_, err = io.Copy(fout, fin)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseSrcTargetFile(filePair string) (ContainerFileMapping, error) {
	pair := strings.Fields(filePair)
	if len(pair) != 2 {
		return ContainerFileMapping{}, InvalidFilePairError
	}
	return ContainerFileMapping{src: pair[0], target: pair[1]}, nil
}
