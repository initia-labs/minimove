package launchtools

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"os"
)

func NewManifestFromFile[manifest Manifest](path string) (*manifest, error) {
	bz, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to read file: %s", path))
	}

	ret := new(manifest)
	if err := json.Unmarshal(bz, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

type Manifest interface {
	FromFile(path string) (Manifest, error)
	Validate() error
}
