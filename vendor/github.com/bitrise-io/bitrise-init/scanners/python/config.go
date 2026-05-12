package python

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	envmanModels "github.com/bitrise-io/envman/v2/models"
)

const (
	runTestsWorkflowID = models.WorkflowID("run_tests")

	pipCachePaths    = "~/.cache/pip"
	poetryCachePaths = "~/.cache/pypoetry"
	uvCachePaths     = "~/.cache/uv"

	pythonVersionInstallScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

bitrise tools install python $PYTHON_VERSION
`

	pipInstallScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

pip install --upgrade pip
pip install -r requirements.txt
`
	pipInstallWithDevScriptTemplate = `#!/usr/bin/env bash
set -euxo pipefail

pip install --upgrade pip
pip install -r requirements.txt
pip install -r %s
`

	pytestRunScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

pytest
`

	poetryInstallScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

pip install poetry
poetry install
`

	poetryInstallNoRootScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

pip install poetry
poetry install --no-root
`

	poetryPytestRunScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

poetry run pytest
`

	uvSyncScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

pip install uv
uv sync
`

	uvPytestRunScriptContent = `#!/usr/bin/env bash
set -euxo pipefail

uv run pytest
`
)

type configDescriptor struct {
	workdir             string
	packageManager      string
	hasPytest           bool
	pythonVersion       string
	devRequirementsFile string
	poetryNeedsNoRoot   bool
	isDefault           bool
}

// packageManagerSetup holds the per-package-manager bits used by the run_tests
// workflow: cache config (key prefix + lockfile + paths) and install/test
// scripts. generateConfigBasedOn resolves one of these and then runs a uniform
// restore-install-test-save sequence.
type packageManagerSetup struct {
	cacheKeyPrefix string
	cacheLockFile  string
	cachePaths     string
	installScript  string
	testScript     string
}

func createConfigDescriptor(proj project, isDefault bool) configDescriptor {
	d := configDescriptor{
		workdir:             "$" + projectDirInputEnvKey,
		packageManager:      proj.packageManager,
		hasPytest:           proj.hasPytest,
		pythonVersion:       proj.pythonVersion,
		devRequirementsFile: proj.devRequirementsFile,
		poetryNeedsNoRoot:   proj.poetryNeedsNoRoot,
		isDefault:           isDefault,
	}
	if proj.projectRelDir == "." {
		d.workdir = ""
	}
	return d
}

func createDefaultConfigDescriptor(packageManager string) configDescriptor {
	return createConfigDescriptor(project{
		projectRelDir:     "$" + projectDirInputEnvKey,
		packageManager:    packageManager,
		hasPytest:         true,
		poetryNeedsNoRoot: true,
	}, true)
}

func generateOptions(projects []project) (models.OptionNode, models.Warnings, models.Icons, error) {
	if len(projects) == 0 {
		return models.OptionNode{}, nil, nil, fmt.Errorf("no Python project files found")
	}

	projectRootOption := models.NewOption(projectDirInputTitle, projectDirInputSummary, projectDirInputEnvKey, models.TypeSelector)
	for _, proj := range projects {
		if proj.packageManager != "" {
			descriptor := createConfigDescriptor(proj, false)
			configOption := models.NewConfigOption(configName(descriptor), nil)
			projectRootOption.AddConfig(proj.projectRelDir, configOption)
		} else {
			pkgMgrOption := models.NewOption(packageManagerInputTitle, packageManagerInputSummary, "", models.TypeSelector)
			for _, pm := range packageManagers {
				descriptor := createConfigDescriptor(proj, false)
				descriptor.packageManager = pm
				configOption := models.NewConfigOption(configName(descriptor), nil)
				pkgMgrOption.AddConfig(pm, configOption)
			}
			projectRootOption.AddOption(proj.projectRelDir, pkgMgrOption)
		}
	}

	return *projectRootOption, nil, nil, nil
}

func generateConfigs(projects []project, sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	if len(projects) == 0 {
		return models.BitriseConfigMap{}, fmt.Errorf("no Python project files found")
	}

	configs := models.BitriseConfigMap{}
	for _, proj := range projects {
		if proj.packageManager != "" {
			descriptor := createConfigDescriptor(proj, false)
			config, err := generateConfigBasedOn(descriptor, sshKeyActivation)
			if err != nil {
				return nil, err
			}
			configs[configName(descriptor)] = config
		} else {
			for _, pm := range packageManagers {
				descriptor := createConfigDescriptor(proj, false)
				descriptor.packageManager = pm
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

func configName(d configDescriptor) string {
	if d.isDefault {
		return "default-python-" + d.packageManager + "-config"
	}

	name := "python"
	if d.workdir == "" {
		name += "-root"
	}
	name += "-" + d.packageManager
	if d.hasPytest {
		name += "-pytest"
	}
	return name + "-config"
}

func generateConfigBasedOn(d configDescriptor, sshKey models.SSHKeyActivation) (string, error) {
	configBuilder := models.NewDefaultConfigBuilder()

	if d.pythonVersion != "" {
		configBuilder.AddTool("python", d.pythonVersion)
	}

	prepareSteps := steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKey})
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, prepareSteps...)

	if d.isDefault {
		configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem("Install Python", pythonVersionInstallScriptContent))
	}

	setup := packageManagerSetupFor(d)
	key := cacheKey(setup.cacheKeyPrefix, setup.cacheLockFile)
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.RestoreCache(key))
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem("Install dependencies", setup.installScript, workdirInputs(d.workdir)...))
	if d.hasPytest {
		configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem("Run tests", setup.testScript, workdirInputs(d.workdir)...))
	}
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.SaveCache(key, setup.cachePaths))

	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.DefaultDeployStepList()...)

	bitriseConfig, err := configBuilder.Generate(scannerName)
	if err != nil {
		return "", err
	}

	data, err := yaml.Marshal(bitriseConfig)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// packageManagerSetupFor returns the cache config and scripts for the package
// manager named in d. Pip and Poetry inject extra context (dev requirements
// file, --no-root) from the descriptor; uv has no per-project variants.
func packageManagerSetupFor(d configDescriptor) packageManagerSetup {
	switch d.packageManager {
	case "uv":
		return packageManagerSetup{
			cacheKeyPrefix: "uv",
			cacheLockFile:  "uv.lock",
			cachePaths:     uvCachePaths,
			installScript:  uvSyncScriptContent,
			testScript:     uvPytestRunScriptContent,
		}
	case "poetry":
		install := poetryInstallScriptContent
		if d.poetryNeedsNoRoot {
			install = poetryInstallNoRootScriptContent
		}
		return packageManagerSetup{
			cacheKeyPrefix: "poetry",
			cacheLockFile:  "poetry.lock",
			cachePaths:     poetryCachePaths,
			installScript:  install,
			testScript:     poetryPytestRunScriptContent,
		}
	default: // pip
		install := pipInstallScriptContent
		if d.devRequirementsFile != "" {
			install = fmt.Sprintf(pipInstallWithDevScriptTemplate, d.devRequirementsFile)
		}
		return packageManagerSetup{
			cacheKeyPrefix: "pip",
			cacheLockFile:  "requirements.txt",
			cachePaths:     pipCachePaths,
			installScript:  install,
			testScript:     pytestRunScriptContent,
		}
	}
}

func cacheKey(prefix, lockFile string) string {
	return fmt.Sprintf(`%s-{{ checksum "%s" }}`, prefix, lockFile)
}

func workdirInputs(workdir string) []envmanModels.EnvironmentItemModel {
	if workdir == "" {
		return nil
	}
	return []envmanModels.EnvironmentItemModel{{"working_dir": workdir}}
}
