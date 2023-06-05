package sdk

import (
	"encoding/json"
	"io"
	"path/filepath"

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

func (r FVMVersionReader) ReadSDKVersion(projectRootDir string) (*semver.Version, error) {
	fvmConfigPth := filepath.Join(projectRootDir, fvmConfigRelPath)
	f, err := r.fileOpener.OpenReaderIfExists(fvmConfigPth)
	if err != nil {
		return nil, err
	}

	if f == nil {
		return nil, nil
	}

	versionStr, err := parseFVMFlutterVersion(f)
	if err != nil {
		return nil, err
	}
	if versionStr == "" {
		return nil, nil
	}

	return semver.NewVersion(versionStr)
}

func parseFVMFlutterVersion(fvmConfigReader io.Reader) (string, error) {
	type fvmConfig struct {
		FlutterSdkVersion string `json:"flutterSdkVersion"`
	}

	var config fvmConfig
	d := json.NewDecoder(fvmConfigReader)
	if err := d.Decode(&config); err != nil {
		return "", err
	}

	return config.FlutterSdkVersion, nil
}
