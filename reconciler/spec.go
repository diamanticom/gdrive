package reconciler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gdrive-org/gdrive/drive"
	"github.com/gdrive-org/gdrive/utils"
	"gopkg.in/yaml.v2"
)

func (s *Spec) Reconcile() error {
Loop:
	for i, file := range s.Files {
		now := time.Now()
		file.g = s.g

		if file.existsLocal() {
			ok, err := file.verifyMd5Local()
			if err != nil {
				return err
			}

			if ok {
				_, _ = fmt.Printf("%2d/%2d: reconciled: %30s [%v]\n",
					i+1, len(s.Files), file.Name, time.Since(now))
				continue Loop
			}
		}

		ok, err := file.verifyMd5Remote()
		if err != nil {
			return err
		}

		if !ok {
			return fmt.Errorf("md5 mismatch between remote %s and spec", file.Name)
		}

		if err := file.download(); err != nil {
			return err
		}

		_, _ = fmt.Printf("%2d/%2d: reconciled: %30s [%v]\n", i+1, len(s.Files), file.Name, time.Since(now))
	}
	return nil
}

func (s *Spec) SetDriver(g *drive.Drive) {
	s.g = g
}

func (s *Spec) Generate() error {
	bb := new(bytes.Buffer)
	bw := bufio.NewWriter(bb)

	owner, ok := os.LookupEnv(utils.AssetOwnerKey)
	var query string
	if !ok {
		query = utils.DefaultQuery
	} else {
		query = fmt.Sprintf("trashed = false and '%s' in owners", owner)
	}

	args := drive.ListFilesArgs{
		Out:      bw,
		MaxFiles: 1000,
		Query:    query,
		AbsPath:  true,
		JsonOut:  true,
	}

	if err := s.g.List(args); err != nil {
		return err
	}

	_ = bw.Flush()

	type listItem struct {
		CreatedTime time.Time `json:"createdTime"`
		ID          string    `json:"id"`
		Md5Checksum string    `json:"md5Checksum"`
		MimeType    string    `json:"mimeType"`
		Name        string    `json:"name"`
		Parents     []string  `json:"parents"`
		Size        string    `json:"size"`
	}

	type list struct {
		Items []*listItem `json:"Files"`
	}

	l := new(list)

	if err := json.Unmarshal(bb.Bytes(), l); err != nil {
		return err
	}

	s.Files = make([]*File, 0, len(l.Items))
	s.Kind = SpecKind
	s.ApiVersion = SpecApiVersionV1Beta1

Loop:
	for _, item := range l.Items {
		switch v := item.MimeType; v {
		case "application/vnd.google-apps.folder":
			continue Loop
		}

		// this is the check for the binary file
		if item.Md5Checksum == "" {
			continue Loop
		}

		file := new(File)
		file.Id = item.ID
		file.Md5 = item.Md5Checksum
		file.LocalPath, file.Name = filepath.Split(item.Name)

		s.Files = append(s.Files, file)
	}

	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	return nil
}
