package nodejs

import (
	"gopkg.in/yaml.v2"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
)

func configName(pkgMgr string, isDefault bool) string {
	name := "node-js-" + pkgMgr + "-config"
	if isDefault {
		name = "default-" + name
	}
	return name
}

func checkPackageJSON(searchDir string) string {
	log.TPrintf("Checking package.json")
	packageJsonPath := filepath.Join(searchDir, packageJson)
	if exists := utility.FileExists(packageJsonPath); exists {
		log.TPrintf("- %s - found", packageJsonPath)
		return packageJsonPath
	}

	log.TPrintf("- %s - not found", packageJsonPath)
	return ""
}

var nodeVersionFiles = []string{".nvmrc", ".node-version", ".tool-versions"}

func checkNodeVersion(searchDir string) string {
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

		// The NVM step will pick up the actual version from the .nvmrc file when running the step
		// Better not hardcode it to a detected version
		if fileName == ".nvmrc" {
			nodeVersion = ""
		}

		return nodeVersion
	}

	return ""
}

func getNodeVersionFromToolVersions(fileContent string) string {
	lines := strings.Split(fileContent, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 1 && fields[0] == "nodejs" {
			return fields[1]
		}
	}

	return ""
}

func checkPackageManager(searchDir string) string {
	log.TPrintf("Checking package manager")
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

func checkPackageScripts(packageJsonPath string) []string {
	log.TPrintf("Checking package scripts")

	var scripts []string

	packages, err := utility.ParsePackagesJSON(packageJsonPath)
	if err != nil {
		return scripts
	}

	for name, _ := range packages.Scripts {
		log.TDebugf("- %s", name)
		scripts = append(scripts, name)
	}

	return scripts
}

func generateConfig(sshKeyActivation models.SSHKeyActivation, manager string, nodeVersion string) (string, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	// test workflow
	workflowName := models.WorkflowID("test")
	prepareSteps := steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKeyActivation})
	configBuilder.AppendStepListItemsTo(workflowName, prepareSteps...)
	configBuilder.AppendStepListItemsTo(workflowName, steps.NvmStepListItem(nodeVersion))
	configBuilder.AppendStepListItemsTo(workflowName, steps.RestoreNPMCache())

	switch manager {
	case "yarn":
		configBuilder.AppendStepListItemsTo(workflowName, steps.YarnStepListItem("install", ""))
		configBuilder.AppendStepListItemsTo(workflowName, steps.YarnStepListItem("run lint", ""))
		configBuilder.AppendStepListItemsTo(workflowName, steps.YarnStepListItem("run test", ""))
	case "npm":
		fallthrough
	default:
		configBuilder.AppendStepListItemsTo(workflowName, steps.NpmStepListItem("install", ""))
		configBuilder.AppendStepListItemsTo(workflowName, steps.NpmStepListItem("run lint", ""))
		configBuilder.AppendStepListItemsTo(workflowName, steps.NpmStepListItem("run test", ""))
	}

	configBuilder.AppendStepListItemsTo(workflowName, steps.SaveNPMCache())

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
