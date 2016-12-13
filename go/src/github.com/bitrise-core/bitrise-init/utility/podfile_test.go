package utility

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestGetWorkspaceProjectMap(t *testing.T) {
	// ---------------------
	// No workspace defined
	t.Log("no workspace defined, no project in the pod folder -- ERROR")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__utility_test__")
		require.NoError(t, err)

		podfile := `platform :ios, '9.0'
pod 'Alamofire', '~> 3.4'
`
		podfilePth := filepath.Join(tmpDir, "Podfile")
		require.NoError(t, fileutil.WriteStringToFile(podfilePth, podfile))

		workspaceProjectMap, err := GetWorkspaceProjectMap(podfilePth)
		require.Error(t, err)
		require.Equal(t, 0, len(workspaceProjectMap))

		require.NoError(t, os.RemoveAll(tmpDir))
	}

	t.Log("no workspace defined, one project in the pod folder")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__utility_test__")
		require.NoError(t, err)

		podfile := `platform :ios, '9.0'
pod 'Alamofire', '~> 3.4'
`
		podfilePth := filepath.Join(tmpDir, "Podfile")
		require.NoError(t, fileutil.WriteStringToFile(podfilePth, podfile))

		project := ""
		projectPth := filepath.Join(tmpDir, "project.xcodeproj")
		require.NoError(t, fileutil.WriteStringToFile(projectPth, project))

		workspaceProjectMap, err := GetWorkspaceProjectMap(podfilePth)
		require.NoError(t, err)
		require.Equal(t, 1, len(workspaceProjectMap))

		for workspace, project := range workspaceProjectMap {
			workspaceBasename := filepath.Base(workspace)
			workspaceName := strings.TrimSuffix(workspaceBasename, ".xcworkspace")

			projectBasename := filepath.Base(project)
			projectName := strings.TrimSuffix(projectBasename, ".xcodeproj")

			require.Equal(t, "project", workspaceName, fmt.Sprintf("%v", workspaceProjectMap))
			require.Equal(t, "project", projectName, fmt.Sprintf("%v", workspaceProjectMap))
		}

		require.NoError(t, os.RemoveAll(tmpDir))
	}

	t.Log("no workspace defined, two project in the pod folder -- ERROR")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__utility_test__")
		require.NoError(t, err)

		podfile := `platform :ios, '9.0'
pod 'Alamofire', '~> 3.4'
`
		podfilePth := filepath.Join(tmpDir, "Podfile")
		require.NoError(t, fileutil.WriteStringToFile(podfilePth, podfile))

		project1 := ""
		project1Pth := filepath.Join(tmpDir, "project1.xcodeproj")
		require.NoError(t, fileutil.WriteStringToFile(project1Pth, project1))

		project2 := ""
		project2Pth := filepath.Join(tmpDir, "project2.xcodeproj")
		require.NoError(t, fileutil.WriteStringToFile(project2Pth, project2))

		workspaceProjectMap, err := GetWorkspaceProjectMap(podfilePth)
		require.Error(t, err)
		require.Equal(t, 0, len(workspaceProjectMap))

		require.NoError(t, os.RemoveAll(tmpDir))
	}

	// ---------------------
	// No workspace defined
	t.Log("workspace defined, no project in the pod folder -- ERROR")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__utility_test__")
		require.NoError(t, err)

		podfile := `platform :ios, '9.0'
pod 'Alamofire', '~> 3.4'
workspace 'MyWorkspace'
`
		podfilePth := filepath.Join(tmpDir, "Podfile")
		require.NoError(t, fileutil.WriteStringToFile(podfilePth, podfile))

		workspaceProjectMap, err := GetWorkspaceProjectMap(podfilePth)
		require.Error(t, err)
		require.Equal(t, 0, len(workspaceProjectMap))

		require.NoError(t, os.RemoveAll(tmpDir))
	}

	t.Log("workspace defined, one project in the pod folder")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__utility_test__")
		require.NoError(t, err)

		podfile := `platform :ios, '9.0'
pod 'Alamofire', '~> 3.4'
workspace 'MyWorkspace'
`
		podfilePth := filepath.Join(tmpDir, "Podfile")
		require.NoError(t, fileutil.WriteStringToFile(podfilePth, podfile))

		project := ""
		projectPth := filepath.Join(tmpDir, "project.xcodeproj")
		require.NoError(t, fileutil.WriteStringToFile(projectPth, project))

		workspaceProjectMap, err := GetWorkspaceProjectMap(podfilePth)
		require.NoError(t, err)
		require.Equal(t, 1, len(workspaceProjectMap))

		for workspace, project := range workspaceProjectMap {
			workspaceBasename := filepath.Base(workspace)
			workspaceName := strings.TrimSuffix(workspaceBasename, ".xcworkspace")

			projectBasename := filepath.Base(project)
			projectName := strings.TrimSuffix(projectBasename, ".xcodeproj")

			require.Equal(t, "MyWorkspace", workspaceName, fmt.Sprintf("%v", workspaceProjectMap))
			require.Equal(t, "project", projectName, fmt.Sprintf("%v", workspaceProjectMap))
		}

		require.NoError(t, os.RemoveAll(tmpDir))
	}

	t.Log("workspace defined, two project in the pod folder -- ERROR")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__utility_test__")
		require.NoError(t, err)

		podfile := `platform :ios, '9.0'
pod 'Alamofire', '~> 3.4'
workspace 'MyWorkspace'
`
		podfilePth := filepath.Join(tmpDir, "Podfile")
		require.NoError(t, fileutil.WriteStringToFile(podfilePth, podfile))

		project1 := ""
		project1Pth := filepath.Join(tmpDir, "project1.xcodeproj")
		require.NoError(t, fileutil.WriteStringToFile(project1Pth, project1))

		project2 := ""
		project2Pth := filepath.Join(tmpDir, "project2.xcodeproj")
		require.NoError(t, fileutil.WriteStringToFile(project2Pth, project2))

		workspaceProjectMap, err := GetWorkspaceProjectMap(podfilePth)
		require.Error(t, err)
		require.Equal(t, 0, len(workspaceProjectMap))

		require.NoError(t, os.RemoveAll(tmpDir))
	}

	t.Log("workspace defined with smart quotes, one project in the pod folder")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__utility_test__")
		require.NoError(t, err)

		podfile := `platform :ios, '9.0'
pod 'Alamofire', '~> 3.4'
workspace ‘MyWorkspace’
`
		podfilePth := filepath.Join(tmpDir, "Podfile")
		require.NoError(t, fileutil.WriteStringToFile(podfilePth, podfile))

		project := ""
		projectPth := filepath.Join(tmpDir, "project.xcodeproj")
		require.NoError(t, fileutil.WriteStringToFile(projectPth, project))

		workspaceProjectMap, err := GetWorkspaceProjectMap(podfilePth)
		require.NoError(t, err)
		require.Equal(t, 1, len(workspaceProjectMap))

		for workspace, project := range workspaceProjectMap {
			workspaceBasename := filepath.Base(workspace)
			workspaceName := strings.TrimSuffix(workspaceBasename, ".xcworkspace")

			projectBasename := filepath.Base(project)
			projectName := strings.TrimSuffix(projectBasename, ".xcodeproj")

			require.Equal(t, "MyWorkspace", workspaceName, fmt.Sprintf("%v", workspaceProjectMap))
			require.Equal(t, "project", projectName, fmt.Sprintf("%v", workspaceProjectMap))
		}

		require.NoError(t, os.RemoveAll(tmpDir))
	}
}
