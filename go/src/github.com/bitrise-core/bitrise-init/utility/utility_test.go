package utility

import (
	"os"
	"strings"
	"testing"

	"path/filepath"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestCaseInsensitiveContains(t *testing.T) {
	require.Equal(t, true, CaseInsensitiveContains(`    <Reference Include="monotouch" />`, `Include="monotouch"`))
	require.Equal(t, true, CaseInsensitiveContains(`    <Reference Include="Xamarin.iOS" />`, `Include="Xamarin.iOS"`))
	require.Equal(t, true, CaseInsensitiveContains(`    <Reference Include="Mono.Android" />`, `Include="Mono.Android`))

	require.Equal(t, false, CaseInsensitiveContains(`    <Reference Include="monotouch" />`, `Include="Xamarin.iOS"`))
	require.Equal(t, false, CaseInsensitiveContains(`    <Reference Include="monotouch" />`, `Include="Mono.Android`))

	require.Equal(t, true, CaseInsensitiveContains(`TEST`, `es`))
	require.Equal(t, true, CaseInsensitiveContains(`TEST`, `eS`))
	require.Equal(t, false, CaseInsensitiveContains(`TEST`, `a`))

	require.Equal(t, true, CaseInsensitiveContains(`test`, `e`))
	require.Equal(t, false, CaseInsensitiveContains(`test`, `a`))

	require.Equal(t, true, CaseInsensitiveContains(` `, ``))
	require.Equal(t, false, CaseInsensitiveContains(` `, `a`))

	require.Equal(t, true, CaseInsensitiveContains(``, ``))
	require.Equal(t, false, CaseInsensitiveContains(``, `a`))
}

func TestListPathInDirSortedByComponents(t *testing.T) {
	files, err := ListPathInDirSortedByComponents("./")
	require.NoError(t, err)
	require.NotEqual(t, 0, len(files))
}

func TestFilterPaths(t *testing.T) {
	t.Log("without any filter")
	{
		paths := []string{
			"/Users/bitrise/test",
			"/Users/vagrant/test",
		}
		filtered, err := FilterPaths(paths)
		require.NoError(t, err)
		require.Equal(t, paths, filtered)
	}

	t.Log("with filter")
	{
		paths := []string{
			"/Users/bitrise/test",
			"/Users/vagrant/test",
		}
		filter := func(pth string) (bool, error) {
			return strings.Contains(pth, "vagrant"), nil
		}
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"/Users/vagrant/test"}, filtered)
	}
}

func TestBaseFilter(t *testing.T) {
	t.Log("allow")
	{
		paths := []string{
			"path/to/my/gradlew",
			"path/to/my/gradlew/file",
		}
		filter := BaseFilter("gradlew", true)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"path/to/my/gradlew"}, filtered)
	}

	t.Log("forbid")
	{
		paths := []string{
			"path/to/my/gradlew",
			"path/to/my/gradlew/file",
		}
		filter := BaseFilter("gradlew", false)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"path/to/my/gradlew/file"}, filtered)
	}
}

func TestExtensionFilter(t *testing.T) {
	t.Log("allow")
	{
		paths := []string{
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
		}
		filter := ExtensionFilter(".xcodeproj", true)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"path/to/my/project.xcodeproj"}, filtered)
	}

	t.Log("forbid")
	{
		paths := []string{
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
		}
		filter := ExtensionFilter(".xcodeproj", false)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"path/to/my/project.xcworkspace"}, filtered)
	}
}

func TestRegexpFilter(t *testing.T) {
	t.Log("allow")
	{
		paths := []string{
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
		}
		filter := RegexpFilter(".*.xcodeproj", true)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"path/to/my/project.xcodeproj"}, filtered)
	}

	t.Log("forbid")
	{
		paths := []string{
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
		}
		filter := RegexpFilter(".*.xcodeproj", false)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"path/to/my/project.xcworkspace"}, filtered)
	}
}

func TestComponentFilter(t *testing.T) {
	t.Log("allow")
	{
		paths := []string{
			"/Users/bitrise/test",
			"/Users/vagrant/test",
		}
		filter := ComponentFilter("bitrise", true)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"/Users/bitrise/test"}, filtered)
	}

	t.Log("forbid")
	{
		paths := []string{
			"/Users/bitrise/test",
			"/Users/vagrant/test",
		}
		filter := ComponentFilter("bitrise", false)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"/Users/vagrant/test"}, filtered)
	}
}

func TestComponentWithExtensionFilter(t *testing.T) {
	t.Log("allow")
	{
		paths := []string{
			"/Users/bitrise.framework/test",
			"/Users/vagrant/test",
		}
		filter := ComponentWithExtensionFilter(".framework", true)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"/Users/bitrise.framework/test"}, filtered)
	}

	t.Log("forbid")
	{
		paths := []string{
			"/Users/bitrise.framework/test",
			"/Users/vagrant/test",
		}
		filter := ComponentWithExtensionFilter(".framework", false)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"/Users/vagrant/test"}, filtered)
	}
}

func TestIsDirectoryFilter(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__bitrise-init__")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	}()

	tmpFile := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, fileutil.WriteStringToFile(tmpFile, ""))

	t.Log("allow")
	{
		paths := []string{
			tmpDir,
			tmpFile,
		}
		filter := IsDirectoryFilter(true)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{tmpDir}, filtered)
	}

	t.Log("forbid")
	{
		paths := []string{
			tmpDir,
			tmpFile,
		}
		filter := IsDirectoryFilter(false)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{tmpFile}, filtered)
	}
}

func TestInDirectoryFilter(t *testing.T) {
	t.Log("allow")
	{
		paths := []string{
			"/Users/bitrise/test",
			"/Users/vagrant/test",
		}
		filter := InDirectoryFilter("/Users/bitrise", true)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"/Users/bitrise/test"}, filtered)
	}

	t.Log("forbid")
	{
		paths := []string{
			"/Users/bitrise/test",
			"/Users/vagrant/test",
		}
		filter := InDirectoryFilter("/Users/bitrise", false)
		filtered, err := FilterPaths(paths, filter)
		require.NoError(t, err)
		require.Equal(t, []string{"/Users/vagrant/test"}, filtered)
	}
}
