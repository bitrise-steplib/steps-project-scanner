package ios

import (
	"encoding/json"
	"github.com/bitrise-io/go-utils/command"
	"os"
	"path/filepath"
)

const (
	spmProjectFile    = "Package.swift"
	testTargetType    = "test"
	schemeSuffix      = "-Package"
	platformNameiOS   = "ios"
	platformNameMacOS = "macos"
)

type spmPlatform struct {
	Name string `json:"platformName"`
}

type spmProduct struct {
	Name string `json:"name"`
}

type spmTarget struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// We do not need any of the properties because we only check for the existence of the dependencies.
type spmDependency struct {
}

type spmProject struct {
	Name         string          `json:"name"`
	Platforms    []spmPlatform   `json:"platforms"`
	Products     []spmProduct    `json:"products"`
	Targets      []spmTarget     `json:"targets"`
	Dependencies []spmDependency `json:"dependencies"`
}

func ParseSPMProject(projectType XcodeProjectType, searchDir string) (DetectResult, error) {
	packagePath := filepath.Join(searchDir, spmProjectFile)
	if !fileExists(packagePath) {
		return DetectResult{}, nil
	}

	cmd := command.New("swift", "package", "dump-package")
	cmd.SetDir(searchDir)
	output, err := cmd.RunAndReturnTrimmedOutput()
	if err != nil {
		return DetectResult{}, err
	}

	var proj spmProject
	err = json.Unmarshal([]byte(output), &proj)
	if err != nil {
		return DetectResult{}, err
	}

	if !supportsProjectType(projectType, proj.Platforms) {
		return DetectResult{}, nil
	}

	scheme := Scheme{
		Name:       schemeName(proj),
		Missing:    false,
		HasXCTests: hasTests(proj.Targets),
		HasAppClip: false,
		Icons:      nil,
	}
	project := Project{
		RelPath:         spmProjectFile,
		IsWorkspace:     false,
		IsPodWorkspace:  false,
		IsSPMProject:    true,
		CarthageCommand: "",
		Warnings:        nil,
		Schemes:         []Scheme{scheme},
	}
	hasDependencies := 0 < len(proj.Dependencies)
	result := DetectResult{
		Projects:           []Project{project},
		HasSPMDependencies: hasDependencies,
		Warnings:           nil,
	}

	return result, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func supportsProjectType(projectType XcodeProjectType, platforms []spmPlatform) bool {
	// Developers can either explicitly specify which platforms are supported or leave it empty to indicate that all
	// the platforms are supported.
	if len(platforms) == 0 {
		return true
	}

	var platformName string
	if projectType == XcodeProjectTypeIOS {
		platformName = platformNameiOS
	} else if projectType == XcodeProjectTypeMacOS {
		platformName = platformNameMacOS
	}

	for _, platform := range platforms {
		if platform.Name == platformName {
			return true
		}
	}
	return false
}

func schemeName(project spmProject) string {
	// SPM has the following behavior. If there is only a single product defined, then the package name will be used as
	// the scheme name and there will be only one scheme. This scheme will contain all the test targets too.
	//
	// But if there are multiple products, then every product will have a dedicated scheme with the same name as the
	// product is called. It will also create an additional scheme which will have the same name as the package plus an
	// additional `-Package` suffix. Only this package level scheme will contain all the tests and the rest will not.
	// Based on my testing, this is the only valuable scheme which we should be looking for.
	if len(project.Products) == 1 {
		return project.Name
	}
	return project.Name + schemeSuffix
}

func hasTests(targets []spmTarget) bool {
	for _, target := range targets {
		if target.Type == testTargetType {
			return true
		}
	}
	return false
}
