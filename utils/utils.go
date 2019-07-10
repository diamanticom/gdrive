package utils

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gdrive-org/gdrive/auth"
	"github.com/gdrive-org/gdrive/cli"
	"github.com/gdrive-org/gdrive/drive"
)

func NewDrive(args cli.Arguments) *drive.Drive {
	oauth, err := getOauthClient(args)
	if err != nil {
		ExitF("Failed getting oauth client: %s", err.Error())
	}

	client, err := drive.New(oauth)
	if err != nil {
		ExitF("Failed getting drive: %s", err.Error())
	}

	return client
}

func getOauthClient(args cli.Arguments) (*http.Client, error) {
	if args.String("refreshToken") != "" && args.String("accessToken") != "" {
		ExitF("Access token not needed when refresh token is provided")
	}

	if args.String("refreshToken") != "" {
		return auth.NewRefreshTokenClient(ClientId, ClientSecret, args.String("refreshToken")), nil
	}

	if args.String("accessToken") != "" {
		return auth.NewAccessTokenClient(ClientId, ClientSecret, args.String("accessToken")), nil
	}

	configDir := getConfigDir(args)

	if args.String("serviceAccount") != "" {
		serviceAccountPath := ConfigFilePath(configDir, args.String("serviceAccount"))
		serviceAccountClient, err := auth.NewServiceAccountClient(serviceAccountPath)
		if err != nil {
			return nil, err
		}
		return serviceAccountClient, nil
	}

	tokenPath := ConfigFilePath(configDir, TokenFilename)
	return auth.NewFileSourceClient(ClientId, ClientSecret, tokenPath, authCodePrompt)
}

func GetDefaultConfigDir() string {
	return filepath.Join(Homedir(), ".gdrive")
}

func ConfigFilePath(basePath, name string) string {
	return filepath.Join(basePath, name)
}

func Homedir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("APPDATA")
	}
	return os.Getenv("HOME")
}

func Equal(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func ExitF(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Println("")
	os.Exit(1)
}

func CheckErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func WriteJson(path string, data interface{}) error {
	tmpFile := path + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	err = json.NewEncoder(f).Encode(data)
	f.Close()
	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	return os.Rename(tmpFile, path)
}

func Md5sum(path string) string {
	h := md5.New()
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func getConfigDir(args cli.Arguments) string {
	// Use dir from environment var if present
	if os.Getenv("GDRIVE_CONFIG_DIR") != "" {
		return os.Getenv("GDRIVE_CONFIG_DIR")
	}
	return args.String("configDir")
}

func authCodePrompt(url string) func() string {
	return func() string {
		fmt.Println("Authentication needed")
		fmt.Println("Go to the following url in your browser:")
		fmt.Printf("%s\n\n", url)
		fmt.Print("Enter verification code: ")

		var code string
		if _, err := fmt.Scan(&code); err != nil {
			fmt.Printf("Failed reading code: %s", err.Error())
		}
		return code
	}
}
