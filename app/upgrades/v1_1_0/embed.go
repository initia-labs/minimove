package v1_1_0

import (
	"embed"
	"path"
)

//go:embed modules/*
var files embed.FS

// GetModuleBytes return upgrade contract module bytes array
func GetModuleBytes() ([][]byte, error) {
	modules, err := files.ReadDir("modules")
	if err != nil {
		return nil, err
	}

	moduleBytes := make([][]byte, len(modules))
	for i, module := range modules {
		bz, err := files.ReadFile(path.Join("modules", module.Name()))
		if err != nil {
			return nil, err
		}

		moduleBytes[i] = bz
	}

	return moduleBytes, nil
}

func GetModuleWithName(name string) ([]byte, error) {
	return files.ReadFile(path.Join("modules", name))
}
