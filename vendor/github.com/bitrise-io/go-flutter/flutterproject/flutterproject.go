package flutterproject

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/bitrise-io/go-flutter/flutterproject/internal/sdk"
	"github.com/bitrise-io/go-flutter/fluttersdk"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"gopkg.in/yaml.v3"
)

type FlutterAndDartSDKVersions struct {
	FVMFlutterVersion         *semver.Version
	ASDFFlutterVersion        *semver.Version
	PubspecFlutterVersion     *sdk.VersionConstraint
	PubspecDartVersion        *sdk.VersionConstraint
	PubspecLockFlutterVersion *sdk.VersionConstraint
	PubspecLockDartVersion    *sdk.VersionConstraint
}

type Pubspec struct {
	Name string `yaml:"name"`
}

type Project struct {
	rootDir    string
	pubspecPth string
	pubspec    Pubspec

	fileManager fileutil.FileManager
	pathChecker pathutil.PathChecker
}

func New(rootDir string, fileManager fileutil.FileManager, pathChecker pathutil.PathChecker) (*Project, error) {
	pubspecPth := filepath.Join(rootDir, sdk.PubspecRelPath)
	pubspecFile, err := fileManager.Open(pubspecPth)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %s", pubspecPth, err)
	}

	var pubspec Pubspec
	if err := yaml.NewDecoder(pubspecFile).Decode(&pubspec); err != nil {
		return nil, fmt.Errorf("failed to parse pubspec.yaml at %s: %s", pubspecPth, err)
	}

	return &Project{
		rootDir:     rootDir,
		pubspecPth:  pubspecPth,
		fileManager: fileManager,
		pathChecker: pathChecker,
		pubspec:     pubspec,
	}, nil
}

func (p *Project) RootDir() string {
	return p.rootDir
}

func (p *Project) Pubspec() Pubspec {
	return p.pubspec
}

func (p *Project) TestDirPth() string {
	const testDirRelPth = "test"

	hasTests := false
	testsDirPath := filepath.Join(p.rootDir, testDirRelPth)

	if exists, err := p.pathChecker.IsDirExists(testsDirPath); err == nil && exists {
		if entries, err := p.fileManager.ReadDirEntryNames(testsDirPath); err == nil && len(entries) > 0 {
			for _, entry := range entries {
				if strings.HasSuffix(entry, "_test.dart") {
					hasTests = true
					break
				}
			}
		}
	}

	if !hasTests {
		testsDirPath = ""
	}

	return testsDirPath
}

func (p *Project) IOSProjectPth() string {
	const iosProjectRelPth = "ios/Runner.xcworkspace"

	hasIOSProject := false
	iosProjectPth := filepath.Join(p.rootDir, iosProjectRelPth)
	if exists, err := p.pathChecker.IsPathExists(iosProjectPth); err == nil && exists {
		hasIOSProject = true
	}

	if !hasIOSProject {
		iosProjectPth = ""
	}

	return iosProjectPth

}

func (p *Project) AndroidProjectPth() string {
	const androidProjectRelPth = "android/build.gradle"
	const androidProjectKtsRelPth = "android/build.gradle.kts"

	hasAndroidProject := false
	androidProjectPth := filepath.Join(p.rootDir, androidProjectRelPth)
	if exists, err := p.pathChecker.IsPathExists(androidProjectPth); err == nil && exists {
		hasAndroidProject = true
	}

	if !hasAndroidProject {
		androidProjectPth = filepath.Join(p.rootDir, androidProjectKtsRelPth)
		if exists, err := p.pathChecker.IsPathExists(androidProjectPth); err == nil && exists {
			hasAndroidProject = true
		}
	}

	if !hasAndroidProject {
		androidProjectPth = ""
	}

	return androidProjectPth
}

func (p *Project) FlutterAndDartSDKVersions() (FlutterAndDartSDKVersions, error) {
	sdkVersions := FlutterAndDartSDKVersions{}

	fvmFlutterVersion, err := sdk.NewFVMVersionReader(p.fileManager).ReadSDKVersion(p.rootDir)
	if err != nil {
		return FlutterAndDartSDKVersions{}, err
	} else {
		sdkVersions.FVMFlutterVersion = fvmFlutterVersion
	}

	asdfFlutterVersion, err := sdk.NewASDFVersionReader(p.fileManager).ReadSDKVersions(p.rootDir)
	if err != nil {
		return FlutterAndDartSDKVersions{}, err
	} else {
		sdkVersions.ASDFFlutterVersion = asdfFlutterVersion
	}

	pubspecLockFlutterVersion, pubspecLockDartVersion, err := sdk.NewPubspecLockVersionReader(p.fileManager).ReadSDKVersions(p.rootDir)
	if err != nil {
		return FlutterAndDartSDKVersions{}, err
	} else {
		sdkVersions.PubspecLockFlutterVersion = pubspecLockFlutterVersion
		sdkVersions.PubspecLockDartVersion = pubspecLockDartVersion
	}

	pubspecFlutterVersion, pubspecDartVersion, err := sdk.NewPubspecVersionReader(p.fileManager).ReadSDKVersions(p.rootDir)
	if err != nil {
		return FlutterAndDartSDKVersions{}, err
	} else {
		sdkVersions.PubspecFlutterVersion = pubspecFlutterVersion
		sdkVersions.PubspecDartVersion = pubspecDartVersion
	}

	return sdkVersions, nil
}

func (p *Project) FlutterSDKVersionToUse() (string, error) {
	sdkVersions, err := p.FlutterAndDartSDKVersions()
	if err != nil {
		return "", err
	}

	sdkQuery := createSDKQuery(sdkVersions)
	release, err := fluttersdk.FindLatestRelease(fluttersdk.MacOS, fluttersdk.ARM64, fluttersdk.Stable, sdkQuery)
	if err != nil {
		return "", err
	}

	return release.Version, nil
}

func createSDKQuery(sdkVersions FlutterAndDartSDKVersions) fluttersdk.SDKQuery {
	var flutterVersion *semver.Version
	var flutterVersionConstraint *semver.Constraints
	switch {
	case sdkVersions.FVMFlutterVersion != nil:
		flutterVersion = sdkVersions.FVMFlutterVersion
	case sdkVersions.ASDFFlutterVersion != nil:
		flutterVersion = sdkVersions.ASDFFlutterVersion
	case sdkVersions.PubspecLockFlutterVersion != nil && sdkVersions.PubspecLockFlutterVersion.Version != nil:
		flutterVersion = sdkVersions.PubspecLockFlutterVersion.Version
	case sdkVersions.PubspecLockFlutterVersion != nil && sdkVersions.PubspecLockFlutterVersion.Constraint != nil:
		flutterVersionConstraint = sdkVersions.PubspecLockFlutterVersion.Constraint
	case sdkVersions.PubspecFlutterVersion != nil && sdkVersions.PubspecFlutterVersion.Version != nil:
		flutterVersion = sdkVersions.PubspecFlutterVersion.Version
	case sdkVersions.PubspecFlutterVersion != nil && sdkVersions.PubspecFlutterVersion.Constraint != nil:
		flutterVersionConstraint = sdkVersions.PubspecFlutterVersion.Constraint
	}

	var dartVersion *semver.Version
	var dartVersionConstraint *semver.Constraints
	switch {
	case sdkVersions.PubspecLockDartVersion != nil && sdkVersions.PubspecLockDartVersion.Version != nil:
		dartVersion = sdkVersions.PubspecLockDartVersion.Version
	case sdkVersions.PubspecLockDartVersion != nil && sdkVersions.PubspecLockDartVersion.Constraint != nil:
		dartVersionConstraint = sdkVersions.PubspecLockDartVersion.Constraint
	case sdkVersions.PubspecDartVersion != nil && sdkVersions.PubspecDartVersion.Version != nil:
		dartVersion = sdkVersions.PubspecDartVersion.Version
	case sdkVersions.PubspecDartVersion != nil && sdkVersions.PubspecDartVersion.Constraint != nil:
		dartVersionConstraint = sdkVersions.PubspecDartVersion.Constraint
	}

	return fluttersdk.SDKQuery{
		FlutterVersion:           flutterVersion,
		FlutterVersionConstraint: flutterVersionConstraint,
		DartVersion:              dartVersion,
		DartVersionConstraint:    dartVersionConstraint,
	}
}
