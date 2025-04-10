package nodejs

import (
	"fmt"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
)

type checkScriptResult struct {
	scripts                    []string
	hasBuild, hasLint, hasTest bool
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

	for name := range packages.Scripts {
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
	workdir    string
	pkgManager string
	hasBuild   bool
	hasLint    bool
	hasTest    bool
	isDefault  bool
}

func createConfigDescriptor(project project, isDefault bool) configDescriptor {
	descriptor := configDescriptor{
		workdir:    "$" + projectDirInputEnvKey,
		pkgManager: project.packageManager,
		hasBuild:   project.hasBuild,
		hasLint:    project.hasLint,
		hasTest:    project.hasTest,
		isDefault:  isDefault,
	}

	// package.json placed in the search dir, no need to change-dir
	if project.projectRelDir == "." {
		descriptor.workdir = ""
	}

	return descriptor
}

func createDefaultConfigDescriptor(packageManager string) configDescriptor {
	return createConfigDescriptor(project{
		projectRelDir:  "$" + projectDirInputEnvKey,
		packageManager: packageManager,
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

	return *packageManagerOption
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

	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem())
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
