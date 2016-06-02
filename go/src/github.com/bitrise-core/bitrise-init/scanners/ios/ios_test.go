package ios

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterXcodeprojectFiles(t *testing.T) {
	t.Log(`Contains ".xcodeproj" & ".xcworkspace" files`)
	{
		fileList := []string{
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-ios-cocoapods/SampleAppWithCocoapods/SampleAppWithCocoapods.xcodeproj",
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcworkspace",
			"/Users/SampleAppWithCocoapods.xcodeproj/SampleAppWithCocoapods.xcworkspace",
			"/Users/Pods/SampleAppWithCocoapods.xcodeproj",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files := filterXcodeprojectFiles(fileList)
		require.Equal(t, 2, len(files))

		// Also sorts ".xcodeproj" & ".xcworkspace" files by path components length
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcworkspace", files[0])
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-ios-cocoapods/SampleAppWithCocoapods/SampleAppWithCocoapods.xcodeproj", files[1])
	}

	t.Log(`Do not contains ".xcodeproj" | ".xcworkspace" file`)
	{
		fileList := []string{
			"path/to/my/gradlew/build.",
			"path/to/my/gradle",
		}

		files := filterXcodeprojectFiles(fileList)
		require.Equal(t, 0, len(files))
	}
}

func TestFilterPodFiles(t *testing.T) {
	t.Log(`Contains "Podfile" files`)
	{
		fileList := []string{
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/fastlane/Podfile",
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/Podfile",
			"path/to/my/gradlew/file",
			"path/to/my/Podfile.lock",
		}

		files := filterPodFiles(fileList)
		require.Equal(t, 2, len(files))

		// Also sorts "Podfile" files by path components length
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/Podfile", files[0])
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/fastlane/Podfile", files[1])
	}

	t.Log(`Do not contains "Podfile" file`)
	{
		fileList := []string{
			"path/to/my/gradlew/build.",
			"path/to/my/gradle",
		}

		files := filterPodFiles(fileList)
		require.Equal(t, 0, len(files))
	}
}

func TestIOSConfigName(t *testing.T) {
	require.Equal(t, "ios-config", configName(false, false))
	require.Equal(t, "ios-pod-config", configName(true, false))
	require.Equal(t, "ios-test-config", configName(false, true))
	require.Equal(t, "ios-pod-test-config", configName(true, true))
}
