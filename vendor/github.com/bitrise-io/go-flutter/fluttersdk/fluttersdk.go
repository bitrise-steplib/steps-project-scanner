package fluttersdk

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func FindLatestRelease(platform Platform, architecture Architecture, channel Channel, query SDKQuery) (*Release, error) {
	releases, err := listReleasesOnChannel(platform, architecture, channel)
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	for _, release := range releases {
		releaseFlutterVersion, err := semver.NewVersion(release.Version)
		if err != nil {
			return nil, err
		}

		releaseDartVersion, err := semver.NewVersion(release.DartSdkVersion)
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

func listReleasesOnChannel(platform Platform, architecture Architecture, channel Channel) ([]Release, error) {
	allReleasesResp, err := listAllReleases(platform)
	if err != nil {
		return nil, err
	}

	var releases []Release

	for _, release := range allReleasesResp.Releases {
		if platform == MacOS && release.DartSdkArch != string(architecture) {
			continue
		}

		if release.Channel != string(channel) {
			continue
		}

		releases = append(releases, release)
	}

	return releases, nil
}

func listAllReleases(platform Platform) (*ReleasesResp, error) {
	flutterReleaseURL := fmt.Sprintf("https://storage.googleapis.com/flutter_infra_release/releases/releases_%s.json", platform)
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
