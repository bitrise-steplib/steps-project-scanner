package sdk

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const fvmConfigRelPath = ".fvm/fvm_config.json"

type FVMVersionReader struct {
	fileOpener FileOpener
}

func NewFVMVersionReader(fileOpener FileOpener) FVMVersionReader {
	return FVMVersionReader{
		fileOpener: fileOpener,
	}
}

func (r FVMVersionReader) ReadSDKVersion(projectRootDir string) (*semver.Version, string, error) {
	fvmConfigPth := filepath.Join(projectRootDir, fvmConfigRelPath)
	f, err := r.fileOpener.OpenReaderIfExists(fvmConfigPth)
	if err != nil {
		return nil, "", err
	}

	if f == nil {
		return nil, "", nil
	}

	versionStr, channel, err := parseFVMFlutterVersion(f)
	if err != nil {
		return nil, "", err
	}
	if versionStr == "" {
		return nil, "", nil
	}

	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, "", err
	}

	return version, channel, nil
}

func parseFVMFlutterVersion(fvmConfigReader io.Reader) (string, string, error) {
	type fvmConfig struct {
		FlutterSdkVersion string `json:"flutterSdkVersion"`
	}

	var config fvmConfig
	d := json.NewDecoder(fvmConfigReader)
	if err := d.Decode(&config); err != nil {
		return "", "", err
	}

	version := config.FlutterSdkVersion
	channel := ""
	s := strings.Split(config.FlutterSdkVersion, "@")
	if len(s) > 1 {
		version = s[0]
		channel = strings.Join(s[1:], "@")
	}

	return version, channel, nil
}
