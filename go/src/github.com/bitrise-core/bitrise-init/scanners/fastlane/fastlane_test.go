package fastlane

import (
	"fmt"
	"strings"
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

		files, err := filterFastfiles(fileList)
		require.NoError(t, err)
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

		files, err := filterFastfiles(fileList)
		require.NoError(t, err)
		require.Equal(t, 0, len(files))
	}
}

func TestFastlaneConfigName(t *testing.T) {
	require.Equal(t, "fastlane-config", configName())
}

func TestInspectFastFileContent(t *testing.T) {
	lines := []string{
		" test ",
		" lane ",
		":xcode",

		"  lane :xcode do",
		"lane :deploy do",
		"  lane :unit_tests do |params|",

		"  private_lane :post_to_slack do |options|",
		"  private_lane :verify_xcode_version do",
	}
	content := strings.Join(lines, "\n")

	expectedMap := map[string]bool{
		"xcode":      false,
		"deploy":     false,
		"unit_tests": false,
	}

	lanes, err := inspectFastfileContent(content)
	require.NoError(t, err)
	require.Equal(t, 3, len(lanes), strings.Join(lanes, "; "))

	for _, lane := range lanes {
		expectedMap[lane] = true
	}

	for lane, found := range expectedMap {
		require.Equal(t, true, found, fmt.Sprintf("lane: %s not found", lane))
	}
}

func TestFastlaneWorkDir(t *testing.T) {
	t.Log("Fastfile's dir, if Fastfile is NOT in fastlane dir")
	{
		expected := "."
		actual := fastlaneWorkDir("Fastfile")
		require.Equal(t, expected, actual)
	}

	t.Log("fastlane dir's parent, if Fastfile is in fastlane dir")
	{
		expected := "."
		actual := fastlaneWorkDir("fastlane/Fastfile")
		require.Equal(t, expected, actual)
	}

	t.Log("Fastfile's dir, if Fastfile is NOT in fastlane dir")
	{
		expected := "test"
		actual := fastlaneWorkDir("test/Fastfile")
		require.Equal(t, expected, actual)
	}

	t.Log("fastlane dir's parent, if Fastfile is in fastlane dir")
	{
		expected := "test"
		actual := fastlaneWorkDir("test/fastlane/Fastfile")
		require.Equal(t, expected, actual)
	}

	t.Log("Fastfile's dir, if Fastfile is NOT in fastlane dir")
	{
		expected := "my/app/test"
		actual := fastlaneWorkDir("my/app/test/Fastfile")
		require.Equal(t, expected, actual)
	}

	t.Log("fastlane dir's parent, if Fastfile is in fastlane dir")
	{
		expected := "my/app/test"
		actual := fastlaneWorkDir("my/app/test/fastlane/Fastfile")
		require.Equal(t, expected, actual)
	}
}
