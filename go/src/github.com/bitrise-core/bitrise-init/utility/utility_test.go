package utility

import (
	"sort"
	"testing"

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

func TestFileList(t *testing.T) {
	files, err := FileList("./")
	require.NoError(t, err)
	require.NotEqual(t, 0, len(files))
}

func TestFilterFilesWithBasPaths(t *testing.T) {
	t.Log(`Contains "gradlew" basePath`)
	{
		fileList := []string{
			"gradlew",
			"path/to/my/gradlew",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files := FilterFilesWithBasPaths(fileList, "gradlew")
		require.Equal(t, 2, len(files))
		require.Equal(t, "gradlew", files[0])
		require.Equal(t, "path/to/my/gradlew", files[1])
	}

	t.Log(`Contains "gradlew" & "my" basePath`)
	{
		fileList := []string{
			"gradlew",
			"path/to/my/gradlew",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files := FilterFilesWithBasPaths(fileList, "gradlew", "my")
		require.Equal(t, 3, len(files))
		require.Contains(t, files, "gradlew", "path/to/my/gradlew", "path/to/my")
		require.Equal(t, "gradlew", files[0])
		require.Equal(t, "path/to/my/gradlew", files[1])
		require.Equal(t, "path/to/my", files[2])
	}

	t.Log(`Does not contains "test" basePath`)
	{
		fileList := []string{
			"gradlew",
			"path/to/my/gradlew",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files := FilterFilesWithBasPaths(fileList, "test")
		require.Equal(t, 0, len(files))
	}

	t.Log(`Empty fileList`)
	{
		files := FilterFilesWithBasPaths([]string{}, "gradlew")
		require.Equal(t, 0, len(files))
	}

	t.Log(`Empty basePath`)
	{
		fileList := []string{
			"gradlew",
			"path/to/my/gradlew",
			"path/to/my/gradlew/file",
			"path/to/my/",
		}

		files := FilterFilesWithBasPaths(fileList, "")
		require.Equal(t, 0, len(files))
	}
}

func TestFilterFilesWithExtensions(t *testing.T) {
	t.Log(`Contains ".xcodeproj" extension`)
	{
		fileList := []string{
			"project.xcodeproj",
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
			"path/to/my",
		}

		files := FilterFilesWithExtensions(fileList, ".xcodeproj")
		require.Equal(t, 2, len(files))
		require.Equal(t, "project.xcodeproj", files[0])
		require.Equal(t, "path/to/my/project.xcodeproj", files[1])
	}

	t.Log(`Contains ".xcodeproj" & ".xcworkspace" extension`)
	{
		fileList := []string{
			"project.xcodeproj",
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
			"path/to/my",
		}

		files := FilterFilesWithExtensions(fileList, ".xcodeproj", ".xcworkspace")
		require.Equal(t, 3, len(files))
		require.Equal(t, "project.xcodeproj", files[0])
		require.Equal(t, "path/to/my/project.xcodeproj", files[1])
		require.Equal(t, "path/to/my/project.xcworkspace", files[2])
	}

	t.Log(`Missing "." in extension`)
	{
		fileList := []string{
			"project.xcodeproj",
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
			"path/to/my",
		}

		files := FilterFilesWithBasPaths(fileList, "xcodeproj")
		require.Equal(t, 0, len(files))
	}

	t.Log(`Does not contains ".test" extension`)
	{
		fileList := []string{
			"project.xcodeproj",
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
			"path/to/my",
		}

		files := FilterFilesWithBasPaths(fileList, ".test")
		require.Equal(t, 0, len(files))
	}

	t.Log(`Empty fileList`)
	{
		files := FilterFilesWithBasPaths([]string{}, ".test")
		require.Equal(t, 0, len(files))
	}

	t.Log(`Empty extension`)
	{
		fileList := []string{
			"project.xcodeproj",
			"path/to/my/project.xcodeproj",
			"path/to/my/project.xcworkspace",
			"path/to/my",
		}

		files := FilterFilesWithBasPaths(fileList, "")
		require.Equal(t, 0, len(files))
	}
}

func TestPathDept(t *testing.T) {
	t.Log("Simple path")
	{
		depth, err := PathDept("/a")
		require.NoError(t, err)
		require.Equal(t, 1, depth)

		depth, err = PathDept("/a/b")
		require.NoError(t, err)
		require.Equal(t, 2, depth)

		depth, err = PathDept("/a/b/c")
		require.NoError(t, err)
		require.Equal(t, 3, depth)
	}

	t.Log("Root path")
	{
		depth, err := PathDept("/")
		require.NoError(t, err)
		require.Equal(t, 0, depth)
	}

	t.Log("Rel path")
	{
		currentDepth, err := PathDept("./")
		require.NoError(t, err)

		depth, err := PathDept("./a")
		require.NoError(t, err)
		require.Equal(t, currentDepth+1, depth)

		depth, err = PathDept("a")
		require.NoError(t, err)
		require.Equal(t, currentDepth+1, depth)

		depth, err = PathDept("a/b")
		require.NoError(t, err)
		require.Equal(t, currentDepth+2, depth)
	}
}

func TestByComponents(t *testing.T) {
	t.Log("Simple sort")
	{
		fileList := []string{
			"path/to",
			"path/to/my",
			"path",
		}

		sort.Sort(ByComponents(fileList))
		require.Equal(t, []string{"path", "path/to", "path/to/my"}, fileList)
	}

	t.Log("Path with equal components length")
	{
		fileList := []string{
			"path1",
			"path/to",
			"path/to/my",
			"path",
		}

		sort.Sort(ByComponents(fileList))
		require.Equal(t, 4, len(fileList))
		require.Equal(t, "path/to/my", fileList[3])
		require.Equal(t, "path/to", fileList[2])
	}

	t.Log("Epxand path test")
	{
		fileList := []string{
			"path/to",
			"./path",
		}

		sort.Sort(ByComponents(fileList))
		require.Equal(t, 2, len(fileList))
		require.Equal(t, "./path", fileList[0])
		require.Equal(t, "path/to", fileList[1])
	}

	t.Log("Same components length, alpahabetic sort test")
	{
		fileList := []string{
			"./c",
			"./a",
			"b",
		}

		sort.Sort(ByComponents(fileList))
		require.Equal(t, 3, len(fileList))
		require.Equal(t, "./a", fileList[0])
		require.Equal(t, "b", fileList[1])
		require.Equal(t, "./c", fileList[2])
	}
}

func TestMapStringStringHasValue(t *testing.T) {
	mapStringString := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	t.Log("Found")
	{
		found := MapStringStringHasValue(mapStringString, "value1")
		require.Equal(t, true, found)
	}

	t.Log("NOT Found")
	{
		found := MapStringStringHasValue(mapStringString, "value")
		require.Equal(t, false, found)
	}
}
