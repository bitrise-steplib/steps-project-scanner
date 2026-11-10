package ios

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-xcode/pathfilters"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcworkspace"
)

// XcodeProjectType ...
type XcodeProjectType string

const (
	// XcodeProjectTypeIOS ...
	XcodeProjectTypeIOS XcodeProjectType = "ios"
	// XcodeProjectTypeMacOS ...
	XcodeProjectTypeMacOS XcodeProjectType = "macos"
)

func findInList(path string, containers []container) (container, bool) {
	for _, container := range containers {
		if container.path() == path {
			return container, true
		}
	}
	return nil, false
}

func removeProjectFromList(projectPth string, projects []container) []container {
	newProjects := []container{}
	for _, project := range projects {
		if project.path() != projectPth {
			newProjects = append(newProjects, project)
		}
	}
	return newProjects
}

func createStandaloneProjectsAndWorkspaces(projectFiles, workspaceFiles []string, projectFactory xcodeproj.Factory, workspaceFactory xcworkspace.Factory) (containers, error) {
	var (
		workspaces         []container
		standaloneProjects []container
	)

	for _, workspaceFile := range workspaceFiles {
		workspace, err := newWorkspace(workspaceFile, workspaceFactory, projectFactory)
		if err != nil {
			return containers{}, err
		}

		workspaces = append(workspaces, workspace)
	}

	for _, projectFile := range projectFiles {
		workspaceContains := false
		for _, workspace := range workspaces {
			workspaceProjectFiles, err := workspace.projectPaths()
			if err != nil {
				return containers{}, err
			}

			if found := sliceutil.IsStringInSlice(projectFile, workspaceProjectFiles); found {
				workspaceContains = true
				break
			}
		}

		if !workspaceContains {
			project, err := newProject(projectFile, projectFactory)
			if err != nil {
				return containers{}, err
			}

			standaloneProjects = append(standaloneProjects, project)
		}
	}

	return containers{
		standaloneProjects: standaloneProjects,
		workspaces:         workspaces,
	}, nil
}

func filterRelevantProjectFiles(fileList []string, sdkFilter pathutil.FilterFunc) ([]string, error) {
	filters := []pathutil.FilterFunc{
		pathfilters.AllowXcodeProjExtFilter,
		pathfilters.AllowIsDirectoryFilter,
		pathfilters.ForbidEmbeddedWorkspaceRegexpFilter,
		pathfilters.ForbidGitDirComponentFilter,
		pathfilters.ForbidPodsDirComponentFilter,
		pathfilters.ForbidCarthageDirComponentFilter,
		pathfilters.ForbidFramworkComponentWithExtensionFilter,
		pathfilters.ForbidCordovaLibDirComponentFilter,
		pathfilters.ForbidNodeModulesComponentFilter,
		sdkFilter,
	}

	return pathutil.FilterPaths(fileList, filters...)
}

func filterRelevantWorkspaceFiles(fileList []string, sdkFilter pathutil.FilterFunc) ([]string, error) {
	filters := []pathutil.FilterFunc{
		pathfilters.AllowXCWorkspaceExtFilter,
		pathfilters.AllowIsDirectoryFilter,
		pathfilters.AllowWorkspaceWithContentsFile,
		pathfilters.ForbidEmbeddedWorkspaceRegexpFilter,
		pathfilters.ForbidGitDirComponentFilter,
		pathfilters.ForbidPodsDirComponentFilter,
		pathfilters.ForbidCarthageDirComponentFilter,
		pathfilters.ForbidFramworkComponentWithExtensionFilter,
		pathfilters.ForbidCordovaLibDirComponentFilter,
		pathfilters.ForbidNodeModulesComponentFilter,
		sdkFilter,
	}

	return pathutil.FilterPaths(fileList, filters...)
}

func newSDKFilter(projectType XcodeProjectType, projectFactory xcodeproj.Factory, workspaceFactory xcworkspace.Factory) pathutil.FilterFunc {
	sdk := "iphoneos"
	if projectType == XcodeProjectTypeMacOS {
		sdk = "macosx"
	}

	return func(pth string) (bool, error) {
		var (
			c   container
			err error
		)
		switch {
		case xcodeproj.IsXcodeProj(pth):
			c, err = newProject(pth, projectFactory)
		case xcworkspace.IsWorkspace(pth):
			c, err = newWorkspace(pth, workspaceFactory, projectFactory)
		default:
			return false, fmt.Errorf("not Xcode project nor workspace file: %s", pth)
		}
		if err != nil {
			return false, err
		}

		projects, err := c.projects()
		if err != nil {
			return false, err
		}

		return slices.ContainsFunc(projects, func(project xcodeproj.XcodeProj) bool {
			return projectSupportsSDK(project, sdk)
		}), nil
	}
}

func projectSupportsSDK(project xcodeproj.XcodeProj, sdk string) bool {
	configurations := slices.Clone(project.BuildConfigurations())
	for _, target := range project.Targets() {
		configurations = append(configurations, target.BuildConfigurations...)
	}

	return slices.ContainsFunc(configurations, func(configuration xcodeproj.BuildConfiguration) bool {
		sdkRoot, _ := configuration.BuildSetting("SDKROOT")
		if sdkRoot == sdk {
			return true
		}
		// An iOS target that lists macosx in SUPPORTED_PLATFORMS is still an iOS target.
		if sdkRoot != "auto" {
			return false
		}

		supportedPlatforms, _ := configuration.BuildSetting("SUPPORTED_PLATFORMS")
		return slices.Contains(strings.Split(supportedPlatforms, " "), sdk)
	})
}

// FilterRelevantPodfiles ...
func FilterRelevantPodfiles(fileList []string) ([]string, error) {
	return pathutil.FilterPaths(fileList,
		AllowPodfileBaseFilter,
		pathfilters.ForbidGitDirComponentFilter,
		pathfilters.ForbidPodsDirComponentFilter,
		pathfilters.ForbidCarthageDirComponentFilter,
		pathfilters.ForbidFramworkComponentWithExtensionFilter,
		pathfilters.ForbidCordovaLibDirComponentFilter,
		pathfilters.ForbidNodeModulesComponentFilter)
}

// FilterRelevantCartFile ...
func FilterRelevantCartFile(fileList []string) ([]string, error) {
	return pathutil.FilterPaths(fileList,
		AllowCartfileBaseFilter,
		pathfilters.ForbidGitDirComponentFilter,
		pathfilters.ForbidPodsDirComponentFilter,
		pathfilters.ForbidCarthageDirComponentFilter,
		pathfilters.ForbidFramworkComponentWithExtensionFilter,
		pathfilters.ForbidCordovaLibDirComponentFilter,
		pathfilters.ForbidNodeModulesComponentFilter)
}
