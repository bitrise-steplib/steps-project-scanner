package scanner

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-init/errormapper"
	"github.com/bitrise-io/bitrise-init/scanners"
	"github.com/bitrise-io/bitrise-init/step"
)

func newPatternErrorMatcher(defaultBuilder errormapper.DefaultDetailedErrorBuilder, patternToBuilder map[string]errormapper.DetailedErrorBuilder) *errormapper.PatternErrorMatcher {
	m := errormapper.PatternErrorMatcher{
		PatternToBuilder: patternToBuilder,
		DefaultBuilder:   defaultBuilder,
	}

	return &m
}

func mapRecommendation(tag, err string) step.Recommendation {
	var matcher *errormapper.PatternErrorMatcher
	switch tag {
	case detectPlatformFailedTag:
		matcher = newDetectPlatformFailedMatcher()
	case optionsFailedTag:
		matcher = newOptionsFailedMatcher()
	}

	if matcher == nil {
		matcher = newGenericMatcher()
	}

	return matcher.Run(err)
}

func newGenericMatcher() *errormapper.PatternErrorMatcher {
	return newPatternErrorMatcher(
		newGenericDetail,
		nil,
	)
}

func newGenericDetail(errorMsg string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       errorMsg,
		Description: "For more information, please see the log.",
	}
}

func newNoPlatformDetectedGenericDetail() errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t recognize your platform.",
		Description: fmt.Sprintf("Our auto-configurator supports %s projects. If you’re adding something else, skip this step and configure your Workflow manually.", strings.Join(availableScanners(), ", ")),
	}
}

func availableScanners() (scannerNames []string) {
	for _, scanner := range scanners.ProjectScanners {
		scannerNames = append(scannerNames, scanner.Name())
	}
	for _, scanner := range scanners.AutomationToolScanners {
		scannerNames = append(scannerNames, scanner.Name())
	}
	return
}

// detectPlatformFailedTag
func newDetectPlatformFailedMatcher() *errormapper.PatternErrorMatcher {
	return newPatternErrorMatcher(
		newDetectPlatformFailedGenericDetail,
		nil,
	)
}

func newDetectPlatformFailedGenericDetail(errorMsg string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t parse your project files.",
		Description: fmt.Sprintf("You can fix the problem and try again, or skip auto-configuration and set up your project manually. Our auto-configurator returned the following error:\n%s", errorMsg),
	}
}

// optionsFailedTag
func newOptionsFailedMatcher() *errormapper.PatternErrorMatcher {
	return newPatternErrorMatcher(
		newOptionsFailedGenericDetail,
		map[string]errormapper.DetailedErrorBuilder{
			`No Gradle Wrapper \(gradlew\) found\.`:                                                                                 newGradlewNotFoundDetail,
			`app\.json file \((.+)\) missing or empty (.+) entry\nThe app\.json file needs to contain:`:                             newAppJSONIssueDetail,
			`app\.json file \((.+)\) missing or empty (.+) entry\nIf the project uses Expo Kit the app.json file needs to contain:`: newExpoAppJSONIssueDetail,
			`Cordova config.xml not found.`:                                                                                         newIonicCapacitorNotSupportedIssueDetail,
		},
	)
}

var newOptionsFailedGenericDetail = newDetectPlatformFailedGenericDetail

func newGradlewNotFoundDetail(errorMsg string, params ...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t find your Gradle Wrapper. Please make sure there is a gradlew file in your project’s root directory.",
		Description: `The Gradle Wrapper ensures that the right Gradle version is installed and used for the build. You can find out more about <a target="_blank" href="https://docs.gradle.org/current/userguide/gradle_wrapper.html">the Gradle Wrapper in the Gradle docs</a>.`,
	}
}

func newAppJSONIssueDetail(errorMsg string, params ...string) errormapper.DetailedError {
	appJSONPath := params[0]
	entryName := params[1]
	return errormapper.DetailedError{
		Title: fmt.Sprintf("Your app.json file (%s) doesn’t have a %s field.", appJSONPath, entryName),
		Description: `The app.json file needs to contain the following entries:
- name
- displayName`,
	}
}

func newExpoAppJSONIssueDetail(errorMsg string, params ...string) errormapper.DetailedError {
	appJSONPath := params[0]
	entryName := params[1]
	return errormapper.DetailedError{
		Title: fmt.Sprintf("Your app.json file (%s) doesn’t have a %s field.", appJSONPath, entryName),
		Description: `If your project uses Expo Kit, the app.json file needs to contain the following entries:
- expo/name
- expo/ios/bundleIdentifier
- expo/android/package`,
	}
}

func newIonicCapacitorNotSupportedIssueDetail(errorMsg string, params ...string) errormapper.DetailedError {
	return errormapper.DetailedError{
		Title:       "We couldn’t find your cordova.xml file.",
		Description: `Our auto-configurator only supports Ionic projects with Cordova at the moment. If you’re trying to add a project with Ionic Capacitor, or something else, some Steps in your automatically generated Workflow might fail. To fix this, replace the failing Steps with script Steps in the Workflow editor later.`,
	}
}
