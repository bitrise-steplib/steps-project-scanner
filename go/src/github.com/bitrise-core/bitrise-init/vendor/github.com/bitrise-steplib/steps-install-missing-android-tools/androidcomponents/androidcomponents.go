package androidcomponents

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/sliceutil"

	"github.com/bitrise-tools/go-android/sdk"
	"github.com/bitrise-tools/go-android/sdkcomponent"
	"github.com/bitrise-tools/go-android/sdkmanager"
	"github.com/bitrise-tools/go-steputils/tools"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	_log "github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

type logger interface {
	Warnf(string, ...interface{})
	Printf(string, ...interface{})
	Errorf(string, ...interface{})
}

var log logger = _log.NewDummyLogger()

type installer struct {
	androidSDK  *sdk.Model
	sdkManager  *sdkmanager.Model
	gradlewPath string
}

// SetLogger ...
func SetLogger(l logger) {
	log = l
}

// InstallLicences ...
func InstallLicences(androidSdk *sdk.Model) error {
	sdkManager, err := sdkmanager.New(androidSdk)
	if err != nil {
		return err
	}

	if !sdkManager.IsLegacySDK() {
		licensesCmd := command.New(filepath.Join(androidSdk.GetAndroidHome(), "tools/bin/sdkmanager"), "--licenses")
		licensesCmd.SetStdin(bytes.NewReader([]byte(strings.Repeat("y\n", 1000))))
		if err := licensesCmd.Run(); err != nil {
			log.Warnf("Failed to install licenses using $(sdkmanager --licenses) command")
			log.Printf("Continue using legacy license installation...")
			log.Printf("")
		} else {
			return nil
		}
	}

	licenceMap := map[string]string{
		"android-sdk-license":           "8933bad161af4178b1185d1a37fbf41ea5269c55\n\nd56f5187479451eabf01fb78af6dfcb131a6481e",
		"android-googletv-license":      "\n601085b94cd77f0b54ff86406957099ebe79c4d6",
		"android-sdk-preview-license":   "\n84831b9409646a918e30573bab4c9c91346d8abd",
		"intel-android-extra-license":   "\nd975f751698a77b662f1254ddbeed3901e976f5a",
		"google-gdk-license":            "\n33b6a2b64607f11b759f320ef9dff4ae5c47d97a",
		"mips-android-sysimage-license": "\ne9acab5b5fbb560a72cfaecce8946896ff6aab9d",
	}

	licencesDir := filepath.Join(androidSdk.GetAndroidHome(), "licenses")
	if exist, err := pathutil.IsDirExists(licencesDir); err != nil {
		return err
	} else if !exist {
		if err := os.MkdirAll(licencesDir, os.ModePerm); err != nil {
			return err
		}
	}

	for name, content := range licenceMap {
		pth := filepath.Join(licencesDir, name)

		if err := fileutil.WriteStringToFile(pth, content); err != nil {
			return err
		}
	}

	return nil
}

// Ensure ...
func Ensure(androidSdk *sdk.Model, gradlewPath string) error {
	sdkManager, err := sdkmanager.New(androidSdk)
	if err != nil {
		return err
	}
	i := installer{
		androidSdk,
		sdkManager,
		gradlewPath,
	}

	return retry.Times(1).Wait(time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("Retrying...")
		}
		return i.scanDependencies()
	})
}

func (i installer) getDependencyCases() map[string]func(match string) error {
	return map[string]func(match string) error{
		`(Observed package id 'ndk-bundle' in inconsistent location)`:         i.ndkInconsistentLocation,
		`(NDK not configured)`:                                                i.ndkNotConfigured,
		`failed to find target with hash string 'android-(.*)'\s*`:            i.target,
		`failed to find Build Tools revision ([0-9.]*)\s*`:                    i.buildTool,
		`Could not find (com\.android\.support\..*)\.`:                        i.extrasLib,
		`Could not find any version that matches (com\.android\.support.*)\.`: i.extrasLib,
	}
}

func getDependenciesOutput(projectLocation string) (string, error) {
	gradleCmd := command.New("./gradlew", "dependencies")
	gradleCmd.SetStdin(strings.NewReader("y"))
	gradleCmd.SetDir(projectLocation)
	return gradleCmd.RunAndReturnTrimmedCombinedOutput()
}

func (i installer) scanDependencies(foundMatches ...string) error {
	out, err := getDependenciesOutput(filepath.Dir(i.gradlewPath))
	if err == nil {
		return nil
	}
	err = fmt.Errorf("output: %s\nerror: %s", out, err)
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		for pattern, callback := range i.getDependencyCases() {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(line); len(matches) == 2 {
				if sliceutil.IsStringInSlice(matches[1], foundMatches) {
					return fmt.Errorf("unable to solve a dependency installation for the output:\n%s", out)
				}
				if callbackErr := callback(matches[1]); callbackErr != nil {
					log.Printf(out)
					return callbackErr
				}
				err = nil
				return i.scanDependencies(append(foundMatches, matches[1])...)
			}
		}
	}
	if scanner.Err() != nil {
		log.Printf(out)
		return scanner.Err()
	}
	return err
}

func (i installer) ndkNotConfigured(_ string) error {
	log.Warnf("NDK not configured")

	ndkComponent := sdkcomponent.SDKTool{SDKStylePath: "ndk-bundle", LegacySDKStylePath: "ndk-bundle"}
	cmd := i.sdkManager.InstallCommand(ndkComponent)
	cmd.SetStdin(strings.NewReader("y"))

	log.Printf("Install and configure NDK bundle using:")
	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("output: %s, error: %s", out, err)
	}

	bundlePath := filepath.Join(i.androidSDK.GetAndroidHome(), "ndk-bundle")

	log.Printf("Setting environment variable (ANDROID_NDK_HOME) to:  %s", bundlePath)
	if err := os.Setenv("ANDROID_NDK_HOME", bundlePath); err != nil {
		return err
	}
	return tools.ExportEnvironmentWithEnvman("ANDROID_NDK_HOME", bundlePath)
}

func (i installer) ndkInconsistentLocation(_ string) error {
	log.Warnf("NDK found on inconsistent path")

	ndkUninstallComponent := sdkcomponent.SDKTool{SDKStylePath: "ndk-bundle", LegacySDKStylePath: "ndk-bundle"}
	cmd := i.sdkManager.InstallCommand(ndkUninstallComponent)
	cmd.SetStdin(strings.NewReader("y"))
	cmd.GetCmd().Args = append([]string{cmd.GetCmd().Args[0], "--uninstall"}, cmd.GetCmd().Args[1:]...)

	log.Printf("Removing NDK bundle using:")
	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("output: %s, error: %s", out, err)
	}
	return i.ndkNotConfigured("")
}

func (i installer) target(version string) error {
	log.Warnf("Missing platform version found: %s", version)

	version = "android-" + version
	platformComponent := sdkcomponent.Platform{
		Version: version,
	}
	cmd := i.sdkManager.InstallCommand(platformComponent)
	cmd.SetStdin(strings.NewReader("y"))

	log.Printf("Installing platform version using:")
	log.Printf("$ %s", cmd.PrintableCommandArgs())

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("output: %s, error: %s", out, err)
	}
	return nil
}

func (i installer) buildTool(buildToolsVersion string) error {
	log.Warnf("Missing build tools version found: %s", buildToolsVersion)

	buildToolsComponent := sdkcomponent.BuildTool{
		Version: buildToolsVersion,
	}

	cmd := i.sdkManager.InstallCommand(buildToolsComponent)
	cmd.SetStdin(strings.NewReader("y"))

	log.Printf("Installing build tools version using:")
	log.Printf("$ %s", cmd.PrintableCommandArgs())

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("output: %s, error: %s", out, err)
	}
	return nil
}

func (i installer) extrasLib(lib string) error {
	log.Warnf("Missing extras library found: %s", lib)

	firstColon := strings.Index(lib, ":")
	lib = strings.Replace(lib[:firstColon], ".", ";", -1) + strings.Replace(lib[firstColon:], ":", ";", -1)

	extrasComponents := sdkcomponent.SupportLibraryInstallComponents()
	extrasComponents = append(extrasComponents, sdkcomponent.Extras{
		Provider:    "m2repository",
		PackageName: lib,
	})
	for _, e := range extrasComponents {
		cmd := i.sdkManager.InstallCommand(e)
		cmd.SetStdin(strings.NewReader("y"))

		log.Printf("Installing extras using:")
		log.Printf("$ %s", cmd.PrintableCommandArgs())

		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			return fmt.Errorf("output: %s, error: %s", out, err)
		}
		return nil
	}
	return nil
}
