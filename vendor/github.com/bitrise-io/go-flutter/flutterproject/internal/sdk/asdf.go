package sdk

import (
	"bufio"
	"io"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const asdfConfigRelPath = ".tool-versions"

type ASDFVersionReader struct {
	fileOpener FileOpener
}

func NewASDFVersionReader(fileOpener FileOpener) ASDFVersionReader {
	return ASDFVersionReader{
		fileOpener: fileOpener,
	}
}

func (r ASDFVersionReader) ReadSDKVersions(projectRootDir string) (*semver.Version, error) {
	asdfConfigPth := filepath.Join(projectRootDir, asdfConfigRelPath)
	f, err := r.fileOpener.OpenReaderIfExists(asdfConfigPth)
	if err != nil {
		return nil, err
	}

	if f == nil {
		return nil, nil
	}

	versionStr, err := parseASDFFlutterVersion(f)
	if err != nil {
		return nil, err
	}
	if versionStr == "" {
		return nil, nil
	}

	return semver.NewVersion(versionStr)
}

func parseASDFFlutterVersion(asdfConfigReader io.Reader) (string, error) {
	scanner := bufio.NewScanner(asdfConfigReader)
	scanner.Split(bufio.ScanLines)
	versionStr := ""
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "flutter ") {
			versionStr = strings.TrimPrefix(line, "flutter ")
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return versionStr, nil
}
