package scanners

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterFastFiles(t *testing.T) {

	t.Log(`Contains "Fastfile" files`)
	{
		fileList := []string{
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/fastlane/Fastfile",
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/Fastfile",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files := filterFastFiles(fileList)
		require.Equal(t, 2, len(files))

		// Also sorts "Fastfile" files by path components length
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/Fastfile", files[0])
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/fastlane/Fastfile", files[1])
	}

	t.Log(`Do not contains "Fastfile" file`)
	{
		fileList := []string{
			"path/to/my/gradlew/build.",
			"path/to/my/gradle",
		}

		files := filterFastFiles(fileList)
		require.Equal(t, 0, len(files))
	}
}

func TestFastlaneConfigName(t *testing.T) {
	require.Equal(t, "fastlane-config.json", fastlaneConfigName(false))
	require.Equal(t, "fastlane-workdir-config.json", fastlaneConfigName(true))
}
