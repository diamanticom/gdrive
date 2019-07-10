package reconciler

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/gdrive-org/gdrive/drive"
	"github.com/google/uuid"
)

func (f *File) existsLocal() bool {
	info, err := os.Stat(filepath.Join(f.LocalPath, f.Name))
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (f *File) verifyMd5Local() (bool, error) {
	b, err := ioutil.ReadFile(filepath.Join(f.LocalPath, f.Name))
	if err != nil {
		return false, err
	}

	md5Sum := md5.Sum(b)
	s := hex.EncodeToString(md5Sum[:])

	return s == f.Md5, nil
}

func (f *File) verifyMd5Remote() (bool, error) {
	bb := new(bytes.Buffer)
	bw := bufio.NewWriter(bb)

	args := drive.FileInfoArgs{
		Out:     bw,
		Id:      f.Id,
		JsonOut: true,
	}

	if err := f.g.Info(args); err != nil {
		return false, err
	}

	_ = bw.Flush()

	out := make(map[string]string)

	if err := json.Unmarshal(bb.Bytes(), &out); err != nil {
		return false, err
	}

	// update remote name
	f.remoteName = out["name"]

	return out["md5Checksum"] == f.Md5, nil
}

func (f *File) download() error {
	if f.Name == "" {
		return fmt.Errorf("name cannot be empty, please check spec")
	}

	if f.g == nil {
		return fmt.Errorf("remote driver is invalid and not initialized")
	}

	if f.remoteName == "" {
		if md5Verified, err := f.verifyMd5Remote(); err != nil {
			return err
		} else {
			if !md5Verified {
				return fmt.Errorf("md5 checksum of remote object does not match spec")
			}
		}
	}

	tmpFolder := filepath.Join("/tmp", uuid.New().String())
	if err := os.Mkdir(tmpFolder, 0755); err != nil {
		return err
	}
	defer os.Remove(tmpFolder)

	bb := new(bytes.Buffer)
	bw := bufio.NewWriter(bb)

	args := drive.DownloadArgs{
		Out:       bw,
		Progress:  os.Stderr,
		Id:        f.Id,
		Path:      tmpFolder,
		Force:     true,
		Skip:      false,
		Recursive: false,
		Delete:    false,
		Stdout:    false,
		Timeout:   time.Second * 120,
		JsonOut:   true,
	}

	if err := f.g.Download(args); err != nil {
		return err
	}

	_ = bw.Flush()

	fTmp := &File{
		Name:      f.remoteName,
		LocalPath: tmpFolder,
		Md5:       f.Md5,
	}

	if !fTmp.existsLocal() {
		return fmt.Errorf("downloaded file %s is missing", filepath.Join(fTmp.LocalPath, fTmp.Name))
	}

	if md5Verified, err := fTmp.verifyMd5Local(); err != nil {
		return err
	} else {
		if !md5Verified {
			return fmt.Errorf("md5sum of downloaded file not valid")
		}
	}

	// check if local path exists, if not create one
	if _, err := os.Stat(f.LocalPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if err := os.MkdirAll(f.LocalPath, 0755); err != nil {
			return err
		}
	}

	b, err := ioutil.ReadFile(filepath.Join(fTmp.LocalPath, fTmp.Name))
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(f.LocalPath, f.Name), b, 0755)
}
