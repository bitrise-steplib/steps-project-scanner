package utility

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
)

const getWorkspacePathGemfileContent = `source 'https://rubygems.org'
gem 'cocoapods-core'
`

const getWorkspacePathRubyScriptContent = `require 'cocoapods-core'
podfile_path = ENV['PODFILE_PATH']
podfile = Pod::Podfile.from_file(podfile_path)
puts podfile.workspace_path
`

// GetWorkspaceProjectMap ...
func GetWorkspaceProjectMap(podfilePth string) (map[string]string, error) {
	// fix podfile quotation
	podfileContent, err := fileutil.ReadStringFromFile(podfilePth)
	if err != nil {
		return map[string]string{}, err
	}

	podfileContent = strings.Replace(podfileContent, `‘`, `'`, -1)
	podfileContent = strings.Replace(podfileContent, `’`, `'`, -1)
	podfileContent = strings.Replace(podfileContent, `“`, `"`, -1)
	podfileContent = strings.Replace(podfileContent, `”`, `"`, -1)

	if err := fileutil.WriteStringToFile(podfilePth, podfileContent); err != nil {
		return map[string]string{}, err
	}
	// ----

	envs := []string{fmt.Sprintf("PODFILE_PATH=%s", podfilePth)}
	podfileDir := filepath.Dir(podfilePth)

	workspaceBase, err := runRubyScriptForOutput(getWorkspacePathRubyScriptContent, getWorkspacePathGemfileContent, podfileDir, envs)
	if err != nil {
		return map[string]string{}, err
	}

	pattern := filepath.Join(podfileDir, "*.xcodeproj")
	projects, err := filepath.Glob(pattern)
	if err != nil {
		return map[string]string{}, err
	}

	if len(projects) > 1 {
		return map[string]string{}, fmt.Errorf("more then 1 xcodeproj exist in Podfile's dir")
	} else if len(projects) == 0 {
		return map[string]string{}, fmt.Errorf("no xcodeproj exist in Podfile's dir")
	}

	project := projects[0]
	workspace := ""

	if workspaceBase != "" {
		workspace = filepath.Join(podfileDir, workspaceBase)
	} else {
		projectBasename := filepath.Base(project)
		projectName := strings.TrimSuffix(projectBasename, ".xcodeproj")
		workspace = filepath.Join(podfileDir, projectName+".xcworkspace")
	}

	return map[string]string{
		workspace: project,
	}, nil
}
