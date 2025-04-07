package nodejs

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
)

type nodeVersionResult struct {
	version string
	file    string
}

type checkScriptResult struct {
	scripts                    []string
	hasBuild, hasLint, hasTest bool
}

// Detection
var nodeVersionFiles = []string{".nvmrc", ".node-version", ".tool-versions"}

func checkNodeVersion(searchDir string) nodeVersionResult {
	log.TPrintf("Checking Node version")

	for _, fileName := range nodeVersionFiles {
		versionFilePath := filepath.Join(searchDir, fileName)
		hasVersionFile := utility.FileExists(versionFilePath)

		if !hasVersionFile {
			log.TPrintf("- %s - not found", fileName)
			continue
		}

		log.TPrintf("- %s - found", fileName)
		fileContent, err := fileutil.ReadStringFromFile(versionFilePath)
		if err != nil {
			log.TWarnf("Failed to read node version from %s", fileName)
			continue
		}

		nodeVersion := strings.TrimSpace(fileContent)
		if fileName == ".tool-versions" {
			nodeVersion = getNodeVersionFromToolVersions(fileContent)
		}

		log.TPrintf("Node version: %s", nodeVersion)

		return nodeVersionResult{
			version: nodeVersion,
			file:    fileName,
		}
	}

	return nodeVersionResult{}
}

func getNodeVersionFromToolVersions(fileContent string) string {
	lines := strings.Split(fileContent, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "nodejs" {
			return fields[1]
		}
	}

	return ""
}

func checkPackageManager(searchDir string) string {
	log.TPrintf("Checking package manager lock files")
	for _, pkgMgr := range pkgManagers {
		hasLockFile := utility.FileExists(filepath.Join(searchDir, pkgMgr.lockFile))

		if !hasLockFile {
			log.TPrintf("- %s - not found", pkgMgr.lockFile)
			continue
		}

		log.TPrintf("- %s - found", pkgMgr.lockFile)
		log.TPrintf("Package manager: %s", pkgMgr.name)
		return pkgMgr.name
	}

	return ""
}

func checkPackageScripts(packageJsonPath string) (checkScriptResult, error) {
	log.TPrintf("Checking package scripts")

	result := checkScriptResult{
		scripts:  make([]string, 0),
		hasBuild: false,
		hasLint:  false,
		hasTest:  false,
	}

	packages, err := utility.ParsePackagesJSON(packageJsonPath)
	if err != nil {
		return result, err
	}

	for name, _ := range packages.Scripts {
		log.TDebugf("- %s", name)
		result.scripts = append(result.scripts, name)
	}

	if slices.Contains(result.scripts, "build") {
		log.TPrintf("- build - found")
		result.hasBuild = true
	} else {
		log.TPrintf("- build - not found")
	}

	if slices.Contains(result.scripts, "lint") {
		log.TPrintf("- lint - found")
		result.hasLint = true
	} else {
		log.TPrintf("- lint - not found")
	}

	if slices.Contains(result.scripts, "test") {
		log.TPrintf("- test - found")
		result.hasTest = true
	} else {
		log.TPrintf("- test - not found")
	}

	return result, nil
}

// Options & Configs
type configDescriptor struct {
	workdir     string
	pkgManager  string
	nodeVersion string
	hasBuild    bool
	hasLint     bool
	hasTest     bool
	isDefault   bool
}

func createConfigDescriptor(project project, isDefault bool) configDescriptor {
	descriptor := configDescriptor{
		workdir:     "$" + projectDirInputEnvKey,
		nodeVersion: "$" + nodeVersionInputEnvKey,
		pkgManager:  project.packageManager,
		hasBuild:    project.hasBuild,
		hasLint:     project.hasLint,
		hasTest:     project.hasTest,
		isDefault:   isDefault,
	}

	// package.json placed in the search dir, no need to change-dir
	if project.projectRelDir == "." {
		descriptor.workdir = ""
	}

	// Don't pin the Node version if the project uses .nvmrc
	if project.usesNvmrc {
		descriptor.nodeVersion = ""
	}

	return descriptor
}

func createDefaultConfigDescriptor(packageManager string) configDescriptor {
	return createConfigDescriptor(project{
		projectRelDir:  "$" + projectDirInputEnvKey,
		nodeVersion:    "$" + nodeVersionInputEnvKey,
		packageManager: packageManager,
		usesNvmrc:      false,
		hasBuild:       true,
		hasLint:        true,
		hasTest:        true,
	}, true)
}

func configName(params configDescriptor) string {
	name := "node-js"

	if params.pkgManager != "" {
		name = name + "-" + params.pkgManager
	}

	if params.isDefault {
		return "default-" + name + "-config"
	}

	if params.workdir == "" {
		name = name + "-root"
	}

	if params.nodeVersion == "" {
		name = name + "-nvm"
	}

	if params.hasBuild {
		name = name + "-build"
	}

	if params.hasLint {
		name = name + "-lint"
	}

	if params.hasTest {
		name = name + "-test"
	}

	return name + "-config"
}

func generateOptions(projects []project) (models.OptionNode, models.Warnings, models.Icons, error) {
	if len(projects) == 0 {
		return models.OptionNode{}, nil, nil, fmt.Errorf("no package.json files found")
	}

	projectRootOption := models.NewOption(projectDirInputTitle, projectDirInputSummary, projectDirInputEnvKey, models.TypeSelector)
	for _, project := range projects {
		options := generateProjectOption(project)
		projectRootOption.AddOption(project.projectRelDir, &options)
	}

	return *projectRootOption, nil, nil, nil
}

func generateProjectOption(project project) models.OptionNode {
	descriptor := createConfigDescriptor(project, false)

	var nodeVersionOption *models.OptionNode
	if project.nodeVersion != "" {
		nodeVersionOption = models.NewOption(nodeVersionInputTitle, nodeVersionInputSummary, nodeVersionInputEnvKey, models.TypeSelector)
	} else {
		nodeVersionOption = models.NewOption(nodeVersionInputTitle, nodeVersionInputSummary, nodeVersionInputEnvKey, models.TypeOptionalUserInput)
	}

	packageManagerOption := models.NewOption(packageManagerInputTitle, packageManagerInputSummary, "", models.TypeSelector)
	if project.packageManager != "" {
		configOption := models.NewConfigOption(configName(descriptor), nil)
		packageManagerOption.AddConfig(project.packageManager, configOption)
	} else {
		for _, pkgMgr := range pkgManagers {
			descriptor.pkgManager = pkgMgr.name
			configOption := models.NewConfigOption(configName(descriptor), nil)
			packageManagerOption.AddConfig(pkgMgr.name, configOption)
		}
	}

	nodeVersionOption.AddOption(project.nodeVersion, packageManagerOption)

	return *nodeVersionOption
}

func generateConfigs(projects []project, sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	if len(projects) == 0 {
		return models.BitriseConfigMap{}, fmt.Errorf("no package.json files found")
	}

	for _, project := range projects {
		descriptor := createConfigDescriptor(project, false)

		if project.packageManager != "" {
			config, err := generateConfigBasedOn(descriptor, sshKeyActivation)
			if err != nil {
				return nil, err
			}
			configs[configName(descriptor)] = config
		} else {
			for _, pkgMgr := range pkgManagers {
				descriptor.pkgManager = pkgMgr.name
				config, err := generateConfigBasedOn(descriptor, sshKeyActivation)
				if err != nil {
					return nil, err
				}
				configs[configName(descriptor)] = config
			}
		}
	}

	return configs, nil
}

func generateConfigBasedOn(descriptor configDescriptor, sshKey models.SSHKeyActivation) (string, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	prepareSteps := steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKey})
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, prepareSteps...)

	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.NvmStepListItem(descriptor.nodeVersion, descriptor.workdir))
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.RestoreNPMCache())

	switch descriptor.pkgManager {
	case "yarn":
		configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.YarnStepListItem("install", descriptor.workdir))
		if descriptor.hasLint {
			configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.YarnStepListItem("run lint", descriptor.workdir))
		}
		if descriptor.hasTest {
			configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.YarnStepListItem("run test", descriptor.workdir))
		}
	case "npm":
		fallthrough
	default:
		configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.NpmStepListItem("install", descriptor.workdir))
		if descriptor.hasLint {
			configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.NpmStepListItem("run lint", descriptor.workdir))

		}
		if descriptor.hasTest {
			configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.NpmStepListItem("run test", descriptor.workdir))
		}
	}

	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.SaveNPMCache())

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return "", err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
