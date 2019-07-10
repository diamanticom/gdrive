package reconciler

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gdrive-org/gdrive/cli"
	"github.com/gdrive-org/gdrive/drive"
	"github.com/gdrive-org/gdrive/utils"
	"github.com/google/uuid"
)

var defaultConfigDir = utils.GetDefaultConfigDir()

func newDrive() *drive.Drive {
	args := make(map[string]interface{})
	args["configDir"] = defaultConfigDir
	args["refreshToken"] = ""
	args["accessToken"] = ""
	args["serviceAccount"] = ""
	args["jsonOut"] = true
	return utils.NewDrive(cli.Arguments(args))
}

func TestFile_Exists(t *testing.T) {
	id := uuid.New().String()

	f := &File{
		LocalPath: filepath.Join("/tmp", id),
		Name:      "test.txt",
	}

	if err := os.Mkdir(f.LocalPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(f.LocalPath)

	if f.existsLocal() {
		t.Fatal("file should not exist")
	}

	if err := ioutil.WriteFile(filepath.Join(f.LocalPath, f.Name), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if !f.existsLocal() {
		t.Fatal("file should exist")
	}
}

func TestFile_VerifyMd5Local(t *testing.T) {
	id := uuid.New().String()

	f := &File{
		LocalPath: filepath.Join("/tmp", id),
		Name:      "test.txt",
		Md5:       "098f6bcd4621d373cade4e832627b4f6",
	}

	if err := os.Mkdir(f.LocalPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(f.LocalPath)

	if err := ioutil.WriteFile(filepath.Join(f.LocalPath, f.Name),
		[]byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	md5Verified, err := f.verifyMd5Local()
	if err != nil {
		t.Fatal(err)
	}

	if !md5Verified {
		t.Fatal("not a valid md5")
	}
}

func TestFile_Download(t *testing.T) {
	g := newDrive()

	id := uuid.New().String()

	f := &File{
		LocalPath: filepath.Join("/tmp", id),
		Name:      "test.txt",
		Id:        "1EmoLLfGZZ2ho6MLKf027EKiy3N59QQ6f",
		Md5:       "e19c1283c925b3206685ff522acfe3e6",
		g:         g,
	}

	if err := os.Mkdir(f.LocalPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(f.LocalPath)

	if f.existsLocal() {
		t.Fatal("local file exists, cannot test download")
	}

	if err := f.download(); err != nil {
		t.Fatal(err)
	}

	if !f.existsLocal() {
		t.Fatal("file does not exist, it should have been downloaded")
	}
}

func TestFile_Download_MustFail_InvalidId(t *testing.T) {
	g := newDrive()

	f := &File{
		Id: "not-a-valid-gdrive-id",
		g:  g,
	}

	if err := f.download(); err == nil {
		t.Fatal("call to invalid id should have failed")
	}
}

func TestFile_Download_MustFail_InvalidName(t *testing.T) {
	f := &File{
		LocalPath: "/tmp",
	}

	if err := f.download(); err == nil {
		t.Fatal("download should fail on empty name")
	}
}

func TestFile_Download_MustFail_InvalidDriver(t *testing.T) {
	f := &File{
		Name: "test.txt",
	}

	if err := f.download(); err == nil {
		t.Fatal("download should fail on invalid driver")
	}
}

func TestFile_Md5Remote(t *testing.T) {
	g := newDrive()

	f := &File{
		Id:  "1EmoLLfGZZ2ho6MLKf027EKiy3N59QQ6f",
		Md5: "e19c1283c925b3206685ff522acfe3e6",
		g:   g,
	}

	md5Verified, err := f.verifyMd5Remote()
	if err != nil {
		t.Fatal(err)
	}

	if !md5Verified {
		t.Fatal("md5 not verified for remote file")
	}
}

func TestFile_Md5Remote_MustFail(t *testing.T) {
	g := newDrive()

	f := &File{
		Id:  "not-a-valid-gdrive-id",
		Md5: "e19c1283c925b3206685ff522acfe3e6",
		g:   g,
	}

	_, err := f.verifyMd5Remote()
	if err == nil {
		t.Fatal("call to invalid id should have failed")
	}
}
