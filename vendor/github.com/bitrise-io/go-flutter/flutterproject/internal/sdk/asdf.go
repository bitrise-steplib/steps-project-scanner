package sdk

import (
	"bufio"
	"io"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const (
	stableChannel = "stable"
	betaChannel   = "beta"
	masterChannel = "dev"
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

func (r ASDFVersionReader) ReadSDKVersions(projectRootDir string) (*semver.Version, string, error) {
	asdfConfigPth := filepath.Join(projectRootDir, asdfConfigRelPath)
	f, err := r.fileOpener.OpenReaderIfExists(asdfConfigPth)
	if err != nil {
		return nil, "", err
	}

	if f == nil {
		return nil, "", nil
	}

	versionStr, channel, err := parseASDFFlutterVersion(f)
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

func parseASDFFlutterVersion(asdfConfigReader io.Reader) (string, string, error) {
	versionStr := ""
	channel := ""

	scanner := bufio.NewScanner(asdfConfigReader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "flutter ") {
			versionStr = strings.TrimPrefix(line, "flutter ")
			for _, c := range []string{stableChannel, betaChannel, masterChannel} {
				channelSuffix := "-" + c
				if strings.HasSuffix(versionStr, channelSuffix) {
					versionStr = strings.TrimSuffix(versionStr, channelSuffix)
					channel = c
				}
			}
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}

	return versionStr, channel, nil
}
