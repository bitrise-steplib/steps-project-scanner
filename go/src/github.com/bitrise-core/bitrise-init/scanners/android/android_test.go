package android

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixedGradlewPath(t *testing.T) {
	require.Equal(t, "./gradlew", FixedGradlewPath("gradlew"))
	require.Equal(t, "./gradlew", FixedGradlewPath("./gradlew"))
	require.Equal(t, "test/gradlew", FixedGradlewPath("test/gradlew"))
}

func TestFilterRootBuildGradleFiles(t *testing.T) {
	t.Log(`Contains "build.gradle" files`)
	{
		fileList := []string{
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/app/build.gradle",
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/build.gradle",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files, err := FilterRootBuildGradleFiles(fileList)
		require.NoError(t, err)
		require.Equal(t, 1, len(files))

		// Also sorts "build.gradle" files by path components length
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/build.gradle", files[0])
	}

	t.Log(`Do not contains "build.gradle" file`)
	{
		fileList := []string{
			"path/to/my/gradlew/build.",
			"path/to/my/gradle",
		}

		files, err := FilterRootBuildGradleFiles(fileList)
		require.NoError(t, err)
		require.Equal(t, 0, len(files))
	}

	t.Log(`Contains 2 top-level "build.gradle" files`)
	{
		fileList := []string{
			"path/to/my/app1/build.gradle",
			"path/to/my/app2/build.gradle",
			"path/to/my/file",
		}

		files, err := FilterRootBuildGradleFiles(fileList)
		require.NoError(t, err)
		require.Equal(t, 2, len(files))
	}
}
