package drive

import (
	"fmt"
	"path/filepath"

	"google.golang.org/api/drive/v3"
)

func (g *Drive) newPathfinder() *remotePathfinder {
	return &remotePathfinder{
		service: g.service.Files,
		files:   make(map[string]*drive.File),
	}
}

type remotePathfinder struct {
	service *drive.FilesService
	files   map[string]*drive.File
}

func (rmt *remotePathfinder) absPath(f *drive.File) (string, error) {
	name := f.Name

	if len(f.Parents) == 0 {
		return name, nil
	}

	var path []string

	for {
		parent, err := rmt.getParent(f.Parents[0])
		if err != nil {
			return "", err
		}

		// Stop when we find the root dir
		if len(parent.Parents) == 0 {
			break
		}

		path = append([]string{parent.Name}, path...)
		f = parent
	}

	path = append(path, name)
	return filepath.Join(path...), nil
}

func (rmt *remotePathfinder) getParent(id string) (*drive.File, error) {
	// Check cache
	if f, ok := rmt.files[id]; ok {
		return f, nil
	}

	// Fetch file from drive
	f, err := rmt.service.Get(id).Fields("id", "name", "parents").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %s", err)
	}

	// Save in cache
	rmt.files[f.Id] = f

	return f, nil
}
