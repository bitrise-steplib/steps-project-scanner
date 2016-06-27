package ios

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// embeddedWorkspaceExp = regexp.MustCompile(`.+.xcodeproj/.+.xcworkspace`)

func TestIsEmbededWorkspace(t *testing.T) {
	t.Log("not embedded workspace")
	{
		actual := isEmbededWorkspace("samplexcodeproj/sample.xcworkspace")
		expected := false
		require.Equal(t, expected, actual)
	}

	t.Log("embedded workspace")
	{
		actual := isEmbededWorkspace("sample.xcodeproj/sample.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}

	t.Log("not embedded workspace")
	{
		actual := isEmbededWorkspace("/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcworkspace")
		expected := false
		require.Equal(t, expected, actual)
	}

	t.Log("not embedded workspace - relative path")
	{
		actual := isEmbededWorkspace("SampleAppWithCocoapods.xcworkspace")
		expected := false
		require.Equal(t, expected, actual)
	}

	t.Log("embedded workspace")
	{
		actual := isEmbededWorkspace("/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj/project.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}

	t.Log("embedded workspace - relative path")
	{
		actual := isEmbededWorkspace("SampleAppWithCocoapods.xcodeproj/project.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}
}

func TestIsPodProject(t *testing.T) {
	t.Log("not pod workspace")
	{
		actual := isPodProject("/Users/bitrise/sample-apps-ios-cocoapods/PodsSampleAppWithCocoapods.xcworkspace")
		expected := false
		require.Equal(t, expected, actual)
	}

	t.Log("not pod project - relative path")
	{
		actual := isPodProject("PodsSampleAppWithCocoapods.xcodeproj")
		expected := false
		require.Equal(t, expected, actual)
	}

	t.Log("pod project")
	{
		actual := isPodProject("/Users/bitrise/sample-apps-ios-cocoapods/Pods/Pods.xcodeproj")
		expected := true
		require.Equal(t, expected, actual)
	}

	t.Log("pod workspace - relative path")
	{
		actual := isPodProject("Pods/Pods.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}

	t.Log("pod workspace - relative path")
	{
		actual := isPodProject("./sub/dir/Pods/Pods.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}

	t.Log("pod workspace - relative path")
	{
		actual := isPodProject("sub/dir/Pods/Pods.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}
}

func TestIsCarthageProject(t *testing.T) {
	t.Log("not Carthage workspace")
	{
		actual := isCarthageProject("/Users/bitrise/sample-apps-ios-cocoapods/CarthageSampleAppWithCocoapods.xcworkspace")
		expected := false
		require.Equal(t, expected, actual)
	}

	t.Log("not Carthage project - relative path")
	{
		actual := isCarthageProject("CarthageSampleAppWithCocoapods.xcodeproj")
		expected := false
		require.Equal(t, expected, actual)
	}

	t.Log("Carthage project")
	{
		actual := isCarthageProject("/Users/bitrise/ios-no-shared-schemes/Carthage/Checkouts/Result/Result.xcodeproj")
		expected := true
		require.Equal(t, expected, actual)
	}

	t.Log("Carthage workspace - relative path")
	{
		actual := isCarthageProject("Carthage/Checkouts/Result/Result.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}

	t.Log("Carthage workspace - relative path")
	{
		actual := isCarthageProject("./sub/dir/Carthage/Checkouts/Result/Result.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}

	t.Log("Carthage workspace - relative path")
	{
		actual := isCarthageProject("sub/dir/Carthage/Checkouts/Result/Result.xcworkspace")
		expected := true
		require.Equal(t, expected, actual)
	}
}

func TestFilterXcodeprojectFiles(t *testing.T) {
	t.Log(`embedded, pod, carthage, relevant project`)
	{
		fileList := []string{
			"/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj/project.xcworkspace",
			"/Users/bitrise/sample-apps-ios-cocoapods/Pods/Pods.xcodeproj",
			"/Users/bitrise/ios-no-shared-schemes/Carthage/Checkouts/Result/Result.xcodeproj",
			"/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj",
		}

		files := filterXcodeprojectFiles(fileList)
		require.Equal(t, 1, len(files))
		require.Equal(t, "/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj", files[0])
	}

	t.Log(`embedded, pod, carthage, relevant project - relative path`)
	{
		fileList := []string{
			"SampleAppWithCocoapods.xcodeproj/project.xcworkspace",
			"Pods/Pods.xcodeproj",
			"Carthage/Checkouts/Result/Result.xcodeproj",
			"SampleAppWithCocoapods.xcodeproj",
		}

		files := filterXcodeprojectFiles(fileList)
		require.Equal(t, 1, len(files))
		require.Equal(t, "SampleAppWithCocoapods.xcodeproj", files[0])
	}

	t.Log(`Contains ".xcodeproj" & ".xcworkspace" files - also sort paths`)
	{
		fileList := []string{
			"/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods/SampleAppWithCocoapods.xcodeproj",
			"/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcworkspace",
		}

		files := filterXcodeprojectFiles(fileList)
		require.Equal(t, 2, len(files))

		require.Equal(t, "/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcworkspace", files[0])
		require.Equal(t, "/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods/SampleAppWithCocoapods.xcodeproj", files[1])
	}

	t.Log(`Not Contains ".xcodeproj" & ".xcworkspace" files - also sort paths`)
	{
		fileList := []string{
			"xcodeproj",
			"xcworkspace",
			"build.gradle",
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
