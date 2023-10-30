package fluttersdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
)

type Platform string

const (
	Linux   Platform = "linux"
	MacOS   Platform = "macos"
	Windows Platform = "windows"
)

type Architecture string

const (
	X64   Architecture = "x64"
	ARM64 Architecture = "arm64"
)

type Release struct {
	Hash           string    `json:"hash"`
	Channel        string    `json:"channel"`
	Version        string    `json:"version"`
	DartSdkVersion string    `json:"dart_sdk_version"`
	DartSdkArch    string    `json:"dart_sdk_arch"`
	ReleaseDate    time.Time `json:"release_date"`
	Archive        string    `json:"archive"`
	Sha256         string    `json:"sha256"`
}

type ReleasesResp struct {
	BaseURL        string `json:"base_url"`
	CurrentRelease struct {
		Beta   string `json:"beta"`
		Dev    string `json:"dev"`
		Stable string `json:"stable"`
	} `json:"current_release"`
	Releases []Release `json:"releases"`
}

type Channel string

const (
	Stable Channel = "stable"
	Beta   Channel = "beta"
	Dev    Channel = "dev"
)

type SDKQuery struct {
	FlutterVersion           *semver.Version
	FlutterVersionConstraint *semver.Constraints
	DartVersion              *semver.Version
	DartVersionConstraint    *semver.Constraints
}

type SDKVersionFinder struct {
	SDKVersionLister SDKVersionLister
}

func NewSDKVersionFinder() SDKVersionFinder {
	return SDKVersionFinder{SDKVersionLister: NewSDKVersionLister()}
}

func (f SDKVersionFinder) FindLatestReleaseFor(platform Platform, architecture Architecture, channel Channel, query SDKQuery) (*Release, error) {
	releasesByChannel, err := f.SDKVersionLister.ListReleasesByChannel(platform, architecture)
	if err != nil {
		return nil, err
	}

	if channel != "" {
		releases := releasesByChannel[string(channel)]
		return findLatestReleaseFor(releases, query)
	}

	for _, c := range []Channel{Stable, Beta, Dev} {
		releases := releasesByChannel[string(c)]
		release, err := findLatestReleaseFor(releases, query)
		if err != nil {
			return nil, err
		}
		if release != nil {
			return release, nil
		}
	}

	return nil, nil
}

func findLatestReleaseFor(releases []Release, query SDKQuery) (*Release, error) {
	var sortErr error
	// sorted in descending version order
	sort.Slice(releases, func(i, j int) bool {
		releaseI := releases[i]
		releaseJ := releases[j]

		releaseVersionI, err := semver.NewVersion(releaseI.Version)
		if err != nil {
			sortErr = err
			return false
		}

		releaseVersionJ, err := semver.NewVersion(releaseJ.Version)
		if err != nil {
			sortErr = err
			return false
		}

		return releaseVersionJ.LessThan(releaseVersionI)
	})
	if sortErr != nil {
		return nil, sortErr
	}

	// Used for parsing the version number from Dart SDK versions like: "2.17.0 (build 2.17.0-266.1.beta)"
	dartSDKWithBuildVersionExp := regexp.MustCompile(`(.+) \(build (.+)\)`)

	for _, release := range releases {
		releaseFlutterVersion, err := semver.NewVersion(release.Version)
		if err != nil {
			return nil, err
		}

		dartSDKVersion := release.DartSdkVersion
		matches := dartSDKWithBuildVersionExp.FindStringSubmatch(dartSDKVersion)
		if len(matches) == 3 {
			dartSDKVersion = matches[1]
		}

		releaseDartVersion, err := semver.NewVersion(dartSDKVersion)
		if err != nil {
			return nil, err
		}

		flutterVersionMatch := false
		dartVersionMatch := false

		if query.FlutterVersion != nil {
			flutterVersionMatch = query.FlutterVersion.Equal(releaseFlutterVersion)
		} else if query.FlutterVersionConstraint != nil {
			flutterVersionMatch = query.FlutterVersionConstraint.Check(releaseFlutterVersion)
		} else {
			flutterVersionMatch = true
		}

		if query.DartVersion != nil {
			dartVersionMatch = query.DartVersion.Equal(releaseDartVersion)
		} else if query.DartVersionConstraint != nil {
			dartVersionMatch = query.DartVersionConstraint.Check(releaseDartVersion)
		} else {
			dartVersionMatch = true
		}

		if flutterVersionMatch && dartVersionMatch {
			return &release, nil
		}
	}

	return nil, nil
}

type SDKVersionLister interface {
	ListReleasesByChannel(platform Platform, architecture Architecture) (map[string][]Release, error)
}

type defaultSDKVersionLister struct {
	baseURLFormat string
}

func NewSDKVersionLister() SDKVersionLister {
	return defaultSDKVersionLister{baseURLFormat: flutterInfraReleasesURLFormat}
}

const flutterInfraReleasesURLFormat = "https://storage.googleapis.com/flutter_infra_release/releases/releases_%s.json"

func (l defaultSDKVersionLister) ListReleasesByChannel(platform Platform, architecture Architecture) (map[string][]Release, error) {
	allReleasesResp, err := l.listAllReleases(platform)
	if err != nil {
		return nil, err
	}

	releasesByChannel := map[string][]Release{}

	for _, release := range allReleasesResp.Releases {
		if platform == MacOS && release.DartSdkArch != string(architecture) {
			continue
		}

		releases := releasesByChannel[release.Channel]
		releases = append(releases, release)
		releasesByChannel[release.Channel] = releases
	}

	return releasesByChannel, nil
}

func (l defaultSDKVersionLister) listAllReleases(platform Platform) (*ReleasesResp, error) {
	flutterReleaseURL := fmt.Sprintf(l.baseURLFormat, platform)
	client := http.DefaultClient
	resp, err := client.Get(flutterReleaseURL)
	if err != nil {
		return nil, err
	}
	var releases ReleasesResp
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&releases); err != nil {
		return nil, err
	}

	return &releases, nil
}
