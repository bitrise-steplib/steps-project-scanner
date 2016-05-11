package scanners

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterGradleFiles(t *testing.T) {
	t.Log(`Contains "build.gradle" files`)
	{
		fileList := []string{
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/app/build.gradle",
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/build.gradle",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files := filterGradleFiles(fileList)
		require.Equal(t, 2, len(files))

		// Also sorts "build.gradle" files by path components length
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/build.gradle", files[0])
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/app/build.gradle", files[1])
	}

	t.Log(`Do not contains "build.gradle" file`)
	{
		fileList := []string{
			"path/to/my/gradlew/build.",
			"path/to/my/gradle",
		}

		files := filterGradleFiles(fileList)
		require.Equal(t, 0, len(files))
	}
}

func TestFilterGradlewFiles(t *testing.T) {
	t.Log(`Contains "gradlew" files`)
	{
		fileList := []string{
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/gradlew",
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/app/gradlew",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files := filterGradlewFiles(fileList)
		require.Equal(t, 2, len(files))

		// Also sorts "gradlew" files by path components length
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/gradlew", files[0])
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/app/gradlew", files[1])
	}

	t.Log(`Do not contains "gradlew" file`)
	{
		fileList := []string{
			"path/to/my/gradlew/build.",
			"path/to/my/gradle",
		}

		files := filterGradlewFiles(fileList)
		require.Equal(t, 0, len(files))
	}
}

func TestAndroidConfigName(t *testing.T) {
	require.Equal(t, "android-config", androidConfigName(false))
	require.Equal(t, "android-gradlew-config", androidConfigName(true))
}
