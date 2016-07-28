package ios

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsPathMatchRegexp(t *testing.T) {
	// Should filter embedded workspace
	t.Log("not embedded workspace")
	{
		actual := isPathMatchRegexp("samplexcodeproj/sample.xcworkspace", embeddedWorkspacePathRegexp)
		require.Equal(t, false, actual)
	}

	t.Log("embedded workspace")
	{
		actual := isPathMatchRegexp("sample.xcodeproj/sample.xcworkspace", embeddedWorkspacePathRegexp)
		require.Equal(t, true, actual)
	}

	t.Log("not embedded workspace")
	{
		actual := isPathMatchRegexp("/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcworkspace", embeddedWorkspacePathRegexp)
		require.Equal(t, false, actual)
	}

	t.Log("not embedded workspace - relative path")
	{
		actual := isPathMatchRegexp("SampleAppWithCocoapods.xcworkspace", embeddedWorkspacePathRegexp)
		require.Equal(t, false, actual)
	}

	t.Log("embedded workspace")
	{
		actual := isPathMatchRegexp("/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj/project.xcworkspace", embeddedWorkspacePathRegexp)
		require.Equal(t, true, actual)
	}

	t.Log("embedded workspace - relative path")
	{
		actual := isPathMatchRegexp("SampleAppWithCocoapods.xcodeproj/project.xcworkspace", embeddedWorkspacePathRegexp)
		require.Equal(t, true, actual)
	}
}

func TestIsPathContainsComponent(t *testing.T) {
	// Should filter .git folder
	t.Log("not inside .git workspace")
	{
		actual := isPathContainsComponent("/Users/bitrise/sample-apps-ios-cocoapods/CarthageSampleAppWithCocoapods.xcworkspace", gitFolderName)
		require.Equal(t, false, actual)
	}

	t.Log("not .git project - relative path")
	{
		actual := isPathContainsComponent("CarthageSampleAppWithCocoapods.xcodeproj", gitFolderName)
		require.Equal(t, false, actual)
	}

	t.Log(".git project")
	{
		actual := isPathContainsComponent("/Users/bitrise/ios-no-shared-schemes/.git/Checkouts/Result/Result.xcodeproj", gitFolderName)
		require.Equal(t, true, actual)
	}

	t.Log(".git workspace - relative path")
	{
		actual := isPathContainsComponent(".git/Checkouts/Result/Result.xcworkspace", gitFolderName)
		require.Equal(t, true, actual)
	}

	t.Log(".git workspace - relative path")
	{
		actual := isPathContainsComponent("./sub/dir/.git/Checkouts/Result/Result.xcworkspace", gitFolderName)
		require.Equal(t, true, actual)
	}

	t.Log(".git workspace - relative path")
	{
		actual := isPathContainsComponent("sub/dir/.git/Checkouts/Result/Result.xcworkspace", gitFolderName)
		require.Equal(t, true, actual)
	}

	// Should filter Pods folder
	t.Log("not pod workspace")
	{
		actual := isPathContainsComponent("/Users/bitrise/sample-apps-ios-cocoapods/PodsSampleAppWithCocoapods.xcworkspace", podsFolderName)
		require.Equal(t, false, actual)
	}

	t.Log("not pod project - relative path")
	{
		actual := isPathContainsComponent("PodsSampleAppWithCocoapods.xcodeproj", podsFolderName)
		require.Equal(t, false, actual)
	}

	t.Log("pod project")
	{
		actual := isPathContainsComponent("/Users/bitrise/sample-apps-ios-cocoapods/Pods/Pods.xcodeproj", podsFolderName)
		require.Equal(t, true, actual)
	}

	t.Log("pod workspace - relative path")
	{
		actual := isPathContainsComponent("Pods/Pods.xcworkspace", podsFolderName)
		require.Equal(t, true, actual)
	}

	t.Log("pod workspace - relative path")
	{
		actual := isPathContainsComponent("./sub/dir/Pods/Pods.xcworkspace", podsFolderName)
		require.Equal(t, true, actual)
	}

	t.Log("pod workspace - relative path")
	{
		actual := isPathContainsComponent("sub/dir/Pods/Pods.xcworkspace", podsFolderName)
		require.Equal(t, true, actual)
	}

	// Should filter Carthage folder
	t.Log("not Carthage workspace")
	{
		actual := isPathContainsComponent("/Users/bitrise/sample-apps-ios-cocoapods/CarthageSampleAppWithCocoapods.xcworkspace", carthageFolderName)
		require.Equal(t, false, actual)
	}

	t.Log("not Carthage project - relative path")
	{
		actual := isPathContainsComponent("CarthageSampleAppWithCocoapods.xcodeproj", carthageFolderName)
		require.Equal(t, false, actual)
	}

	t.Log("Carthage project")
	{
		actual := isPathContainsComponent("/Users/bitrise/ios-no-shared-schemes/Carthage/Checkouts/Result/Result.xcodeproj", carthageFolderName)
		require.Equal(t, true, actual)
	}

	t.Log("Carthage workspace - relative path")
	{
		actual := isPathContainsComponent("Carthage/Checkouts/Result/Result.xcworkspace", carthageFolderName)
		require.Equal(t, true, actual)
	}

	t.Log("Carthage workspace - relative path")
	{
		actual := isPathContainsComponent("./sub/dir/Carthage/Checkouts/Result/Result.xcworkspace", carthageFolderName)
		require.Equal(t, true, actual)
	}

	t.Log("Carthage workspace - relative path")
	{
		actual := isPathContainsComponent("sub/dir/Carthage/Checkouts/Result/Result.xcworkspace", carthageFolderName)
		require.Equal(t, true, actual)
	}
}

func TestIsPathContainsComponentWithExtension(t *testing.T) {
	// Should filter .framework folder
	t.Log("not .framework workspace")
	{
		actual := isPathContainsComponentWithExtension("/Users/bitrise/sample-apps-ios-cocoapods/CarthageSampleAppWithCocoapods.xcworkspace", frameworkExt)
		require.Equal(t, false, actual)
	}

	t.Log("not .framework project - relative path")
	{
		actual := isPathContainsComponentWithExtension("CarthageSampleAppWithCocoapods.xcodeproj", frameworkExt)
		require.Equal(t, false, actual)
	}

	t.Log(".framework project")
	{
		actual := isPathContainsComponentWithExtension("/Users/bitrise/ios-no-shared-schemes/test.framework/Checkouts/Result/Result.xcodeproj", frameworkExt)
		require.Equal(t, true, actual)
	}

	t.Log(".framework workspace - relative path")
	{
		actual := isPathContainsComponentWithExtension("test.framework/Checkouts/Result/Result.xcworkspace", frameworkExt)
		require.Equal(t, true, actual)
	}

	t.Log(".framework workspace - relative path")
	{
		actual := isPathContainsComponentWithExtension("./sub/dir/test.framework/Checkouts/Result/Result.xcworkspace", frameworkExt)
		require.Equal(t, true, actual)
	}

	t.Log(".framework workspace - relative path")
	{
		actual := isPathContainsComponentWithExtension("sub/dir/test.framework/Checkouts/Result/Result.xcworkspace", frameworkExt)
		require.Equal(t, true, actual)
	}
}

func TestIsRelevantProject(t *testing.T) {
	t.Log(`embedded, .git, pod, carthage, .framework - not relevant`)
	{
		fileList := []string{
			"/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj/project.xcworkspace",
			"/Users/bitrise/.git/SampleAppWithCocoapods.xcodeproj/project.xcworkspace",
			"/Users/bitrise/sample-apps-ios-cocoapods/Pods/Pods.xcodeproj",
			"/Users/bitrise/ios-no-shared-schemes/Carthage/Checkouts/Result/Result.xcodeproj",
			"/Users/bitrise/ios-no-shared-schemes/test.framework/Checkouts/Result/Result.xcodeproj",
		}

		for _, file := range fileList {
			require.Equal(t, false, isRelevantProject(file))
		}
	}

	t.Log(`relevant project`)
	{
		fileList := []string{
			"/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj",
		}

		for _, file := range fileList {
			require.Equal(t, true, isRelevantProject(file))
		}
	}
}

func TestFilterXcodeprojectFiles(t *testing.T) {
	t.Log(`embedded, .git, pod, carthage, .framework, relevant project`)
	{
		fileList := []string{
			"/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj/project.xcworkspace",
			"/Users/bitrise/.git/SampleAppWithCocoapods.xcodeproj/project.xcworkspace",
			"/Users/bitrise/sample-apps-ios-cocoapods/Pods/Pods.xcodeproj",
			"/Users/bitrise/ios-no-shared-schemes/Carthage/Checkouts/Result/Result.xcodeproj",
			"/Users/bitrise/ios-no-shared-schemes/test.framework/Checkouts/Result/Result.xcodeproj",
			"/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj",
		}

		files := filterXcodeprojectFiles(fileList)
		require.Equal(t, 1, len(files))
		require.Equal(t, "/Users/bitrise/sample-apps-ios-cocoapods/SampleAppWithCocoapods.xcodeproj", files[0])
	}

	t.Log(`embedded, .git, pod, carthage, .framework, relevant project - relative path`)
	{
		fileList := []string{
			"SampleAppWithCocoapods.xcodeproj/project.xcworkspace",
			".git/SampleAppWithCocoapods.xcodeproj/project.xcworkspace",
			"Pods/Pods.xcodeproj",
			"Carthage/Checkouts/Result/Result.xcodeproj",
			"test.framework/Checkouts/Result/Result.xcodeproj",
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

func TestIsRelevantPodfile(t *testing.T) {
	t.Log(`.git, pod, carthage, .framework - not relevant`)
	{
		fileList := []string{
			"/Users/bitrise/.git/Podfile",
			"/Users/bitrise/sample-apps-ios-cocoapods/Pods/Podfile",
			"/Users/bitrise/ios-no-shared-schemes/Carthage/Checkouts/Result/Podfile",
			"/Users/bitrise/ios-no-shared-schemes/test.framework/Checkouts/Result/Podfile",
		}

		for _, file := range fileList {
			require.Equal(t, false, isRelevantPodfile(file))
		}
	}

	t.Log(`relevant podfile`)
	{
		fileList := []string{
			"/Users/bitrise/sample-apps-ios-cocoapods/Podfile",
		}

		for _, file := range fileList {
			require.Equal(t, true, isRelevantPodfile(file))
		}
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
