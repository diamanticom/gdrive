package reconciler

import (
	"fmt"
	"io/ioutil"

	"github.com/gdrive-org/gdrive/drive"
	"gopkg.in/yaml.v2"
)

func New(fileName string, g *drive.Drive) (Reconciler, error) {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	type kind struct {
		Kind       string
		ApiVersion string
	}

	k := new(kind)
	if err := yaml.Unmarshal(b, k); err != nil {
		return nil, err
	}

	if k.Kind != SpecKind {
		return nil, fmt.Errorf("spec kind is not valid. expected:%s", SpecKind)
	}

	switch v := k.ApiVersion; v {
	case SpecApiVersionV1Beta1:
	default:
		return nil, fmt.Errorf("spec api version is not valid. expected:%s", SpecApiVersionV1Beta1)
	}

	switch v := k.ApiVersion; v {
	case SpecApiVersionV1Beta1:
		spec := new(Spec)
		if err := yaml.Unmarshal(b, spec); err != nil {
			return nil, err
		}
		spec.g = g
		return spec, nil
	default:
		return nil, fmt.Errorf("requested api is not supported")
	}
}
