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

type checkScriptResult struct {
	scripts            []string
	hasLint, hasTest bool
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
		scripts: make([]string, 0),
		hasLint: false,
		hasTest: false,
	}

	packages, err := utility.ParsePackagesJSON(packageJsonPath)
	if err != nil {
		return result, err
	}

	for name := range packages.Scripts {
		log.TDebugf("- %s", name)
		result.scripts = append(result.scripts, name)
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

// detectFramework returns the JS framework detected from package.json dependencies.
// Returns "nextjs", "nestjs", or "" if none is detected.
func detectFramework(packageJsonPath string) string {
	log.TPrintf("Checking framework")

	packages, err := utility.ParsePackagesJSON(packageJsonPath)
	if err != nil {
		log.TPrintf("- framework - failed to parse package.json: %s", err)
		return ""
	}

	allDeps := make(map[string]string)
	for k, v := range packages.Dependencies {
		allDeps[k] = v
	}
	for k, v := range packages.DevDependencies {
		allDeps[k] = v
	}

	if _, ok := allDeps["next"]; ok {
		log.TPrintf("- framework: nextjs")
		return "nextjs"
	}
	if _, ok := allDeps["@nestjs/core"]; ok {
		log.TPrintf("- framework: nestjs")
		return "nestjs"
	}

	log.TPrintf("- framework - not detected")
	return ""
}

// detectNodeVersion returns the Node.js version declared in version files or package.json engines.
// Sources checked in order: .nvmrc, .node-version, .tool-versions, engines.node in package.json.
// Returns an empty string if no version is found.
func detectNodeVersion(projectDir, packageJsonPath string) string {
	log.TPrintf("Checking Node.js version")

	// .nvmrc — single line containing the version (e.g. "22" or "22.14.0")
	if content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, ".nvmrc")); err == nil {
		version := strings.TrimSpace(content)
		if version != "" {
			log.TPrintf("- .nvmrc - found (%s)", version)
			return version
		}
	}

	// .node-version — same format as .nvmrc
	if content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, ".node-version")); err == nil {
		version := strings.TrimSpace(content)
		if version != "" {
			log.TPrintf("- .node-version - found (%s)", version)
			return version
		}
	}

	// .tool-versions — asdf/mise format: "nodejs <version>"
	if content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, ".tool-versions")); err == nil {
		for _, line := range strings.Split(content, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "nodejs" {
				log.TPrintf("- .tool-versions - found nodejs %s", fields[1])
				return fields[1]
			}
		}
	}

	// engines.node in package.json — semver range, e.g. ">=22.0.0"
	packages, err := utility.ParsePackagesJSON(packageJsonPath)
	if err == nil {
		if constraint, ok := packages.Engines["node"]; ok && constraint != "" {
			version := parseEnginesNodeVersion(constraint)
			if version != "" {
				log.TPrintf("- engines.node - found (%s → %s)", constraint, version)
				return version
			}
		}
	}

	log.TPrintf("- Node.js version - not found")
	return ""
}

// parseEnginesNodeVersion strips leading semver operators and returns the first version string.
// e.g. ">=22.0.0" → "22.0.0", "^18.17.0" → "18.17.0"
func parseEnginesNodeVersion(constraint string) string {
	constraint = strings.TrimSpace(constraint)
	i := 0
	for i < len(constraint) {
		c := constraint[i]
		if c == '>' || c == '<' || c == '=' || c == '^' || c == '~' || c == 'v' || c == ' ' {
			i++
		} else {
			break
		}
	}
	version := strings.TrimSpace(constraint[i:])
	// Take only the first token in case of compound ranges like ">=18.0.0 <20"
	if parts := strings.Fields(version); len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Options & Configs
type configDescriptor struct {
	workdir     string
	pkgManager  string
	hasLint     bool
	hasTest     bool
	isDefault   bool
	nodeVersion string
}

func createConfigDescriptor(project project, isDefault bool) configDescriptor {
	descriptor := configDescriptor{
		workdir:     "$" + projectDirInputEnvKey,
		pkgManager:  project.packageManager,
		hasLint:     project.hasLint,
		hasTest:     project.hasTest,
		isDefault:   isDefault,
		nodeVersion: project.nodeVersion,
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
	if descriptor.nodeVersion != "" {
		configBuilder.AddTool("node", descriptor.nodeVersion)
	}

	prepareSteps := steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKey})
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, prepareSteps...)

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
