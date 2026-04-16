package ruby

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	bitriseModels "github.com/bitrise-io/bitrise/v2/models"
	envmanModels "github.com/bitrise-io/envman/v2/models"
	"github.com/bitrise-io/go-utils/pointers"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

const (
	runTestsWorkflowID = models.WorkflowID("run_tests")

	gemCachePaths = "vendor/bundle"
	gemCacheKey   = `gem-{{ checksum "Gemfile.lock" }}`
)

const (
	systemDepsInstallScriptStepTitle = "Install system dependencies"

	bundlerInstallScriptStepTitle   = "Install dependencies"
	bundlerInstallScriptStepContent = `#!/usr/bin/env bash
set -euxo pipefail

bundle config set --local path vendor/bundle
bundle install
`
)

const (
	rubyVersionInputTitle           = "Ruby version"
	rubyVersionInputSummary         = "The Ruby version to be used for the project. Use exact (3.2.0) or partial (3:latest, 3:installed) versions."
	rubyVersionEnvKey               = "RUBY_VERSION"
	rubyVersionInstallScriptContent = "bitrise tools install ruby $RUBY_VERSION"
)

type configDescriptor struct {
	workdir        string
	hasBundler     bool
	hasRakefile    bool
	testFramework  string
	rubyVersion    string
	hasRails       bool
	isDefault      bool
	databases      []databaseGem
	dbYMLInfo      databaseYMLInfo
	mongoidYMLInfo mongoidYMLInfo
}

func generateOptions(projects []project) (models.OptionNode, models.Warnings, models.Icons, error) {
	if len(projects) == 0 {
		return models.OptionNode{}, nil, nil, fmt.Errorf("no Gemfile files found")
	}

	projectRootOption := models.NewOption(projectDirInputTitle, projectDirInputSummary, projectDirInputEnvKey, models.TypeSelector)
	for _, project := range projects {
		descriptor := createConfigDescriptor(project, false)
		configOption := models.NewConfigOption(configName(descriptor), nil)
		projectRootOption.AddConfig(project.projectRelDir, configOption)
	}

	return *projectRootOption, nil, nil, nil
}

func generateConfigs(projects []project, sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	configs := models.BitriseConfigMap{}

	if len(projects) == 0 {
		return models.BitriseConfigMap{}, fmt.Errorf("no Gemfile files found")
	}

	for _, project := range projects {
		descriptor := createConfigDescriptor(project, false)
		config, err := generateConfigBasedOn(descriptor, sshKeyActivation)
		if err != nil {
			return nil, err
		}
		configs[configName(descriptor)] = config
	}

	return configs, nil
}

func generateConfigBasedOn(descriptor configDescriptor, sshKey models.SSHKeyActivation) (string, error) {
	configBuilder := models.NewDefaultConfigBuilder()
	// Declarative Ruby version — runs before any step, no explicit install step needed
	if descriptor.rubyVersion != "" {
		configBuilder.AddTool("ruby", descriptor.rubyVersion)
	}

	prepareSteps := steps.DefaultPrepareStepList(steps.PrepareListParams{SSHKeyActivation: sshKey})
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, prepareSteps...)

	if descriptor.isDefault {
		configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem("Install Ruby", rubyVersionInstallScriptContent))
	}

	// Restore gem cache
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.RestoreCache(gemCacheKey))

	// Install system dependencies (e.g. native library headers required by some gems)
	if aptPackages := collectAptPackages(descriptor.databases); len(aptPackages) > 0 {
		configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem(systemDepsInstallScriptStepTitle, generateSystemDepsScript(aptPackages)))
	}

	// Install dependencies
	if descriptor.hasBundler {
		configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem(bundlerInstallScriptStepTitle, bundlerInstallScriptStepContent, workdirInputs(descriptor.workdir)...))
	}

	serviceContainerNames := serviceContainerReferences(descriptor.databases)
	relationalServiceContainerNames := relationalServiceContainerReferences(descriptor.databases)

	// Database setup (only for relational DBs)
	if hasRelationalDB(descriptor.databases) {
		dbSetupScript := generateDBSetupScript(descriptor)
		if len(relationalServiceContainerNames) > 0 {
			configBuilder.AppendStepListItemsTo(runTestsWorkflowID, scriptStepWithServiceContainers("Database setup", dbSetupScript, relationalServiceContainerNames, descriptor.workdir))
		} else {
			configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem("Database setup", dbSetupScript, workdirInputs(descriptor.workdir)...))
		}
	}

	// Run tests based on detected framework
	testScript := generateTestScript(descriptor)
	if testScript != "" {
		if len(serviceContainerNames) > 0 {
			configBuilder.AppendStepListItemsTo(runTestsWorkflowID, scriptStepWithServiceContainers("Run tests", testScript, serviceContainerNames, descriptor.workdir))
		} else {
			configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.ScriptStepListItem("Run tests", testScript, workdirInputs(descriptor.workdir)...))
		}
	}

	// Save gem cache
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.SaveCache(gemCacheKey, gemCachePaths))

	// Deploy steps
	configBuilder.AppendStepListItemsTo(runTestsWorkflowID, steps.DefaultDeployStepList()...)

	// Build app-level env vars for database connections
	appEnvs := buildAppEnvs(descriptor.databases, descriptor.dbYMLInfo, descriptor.mongoidYMLInfo)

	if len(descriptor.databases) > 0 {
		containers := buildContainerDefinitions(descriptor.databases, descriptor.dbYMLInfo)
		if len(containers) > 0 {
			configBuilder.SetContainerDefinitions(containers)
		}
	}

	config, err := configBuilder.Generate(scannerName, appEnvs...)
	if err != nil {
		return "", err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func createConfigDescriptor(project project, isDefault bool) configDescriptor {
	descriptor := configDescriptor{
		workdir:        "$" + projectDirInputEnvKey,
		hasBundler:     project.hasBundler,
		hasRakefile:    project.hasRakefile,
		testFramework:  project.testFramework,
		rubyVersion:    project.rubyVersion,
		hasRails:       project.hasRails,
		isDefault:      isDefault,
		databases:      project.databases,
		dbYMLInfo:      project.dbYMLInfo,
		mongoidYMLInfo: project.mongoidYMLInfo,
	}

	// Gemfile placed in the search dir, no need to change-dir
	if project.projectRelDir == "." {
		descriptor.workdir = ""
	}

	return descriptor
}

func createDefaultConfigDescriptor() configDescriptor {
	return createConfigDescriptor(project{
		projectRelDir: "$" + projectDirInputEnvKey,
		hasBundler:    true,
		hasRakefile:   true,
		testFramework: "rspec",
	}, true)
}

func configName(params configDescriptor) string {
	name := "ruby"

	if params.isDefault {
		return "default-" + name + "-config"
	}

	if params.workdir == "" {
		name = name + "-root"
	}

	if params.hasBundler {
		name = name + "-bundler"
	}

	if params.testFramework != "" {
		name = name + "-" + params.testFramework
	}

	for _, db := range params.databases {
		if db.containerName != "" {
			name = name + "-" + db.containerName
		}
	}

	return name + "-config"
}

func collectAptPackages(databases []databaseGem) []string {
	seen := map[string]bool{}
	var packages []string
	for _, db := range databases {
		for _, pkg := range db.aptPackages {
			if !seen[pkg] {
				seen[pkg] = true
				packages = append(packages, pkg)
			}
		}
	}
	return packages
}

func generateSystemDepsScript(packages []string) string {
	return "#!/usr/bin/env bash\nset -euxo pipefail\n\napt-get update\napt-get install -y " + strings.Join(packages, " ") + "\n"
}

func serviceContainerReferences(databases []databaseGem) []stepmanModels.ContainerReference {
	var refs []stepmanModels.ContainerReference
	for _, db := range databases {
		if db.containerName != "" {
			refs = append(refs, db.containerName)
		}
	}
	return refs
}

func relationalServiceContainerReferences(databases []databaseGem) []stepmanModels.ContainerReference {
	var refs []stepmanModels.ContainerReference
	for _, db := range databases {
		if db.isRelationalDB && db.containerName != "" {
			refs = append(refs, db.containerName)
		}
	}
	return refs
}

func scriptStepWithServiceContainers(title, content string, serviceContainerRefs []stepmanModels.ContainerReference, workdir string) bitriseModels.StepListItemModel {
	stepID := steps.ScriptID + "@" + steps.ScriptVersion
	inputs := []envmanModels.EnvironmentItemModel{{"content": content}}
	inputs = append(inputs, workdirInputs(workdir)...)
	step := stepmanModels.StepModel{
		Title:             pointers.NewStringPtr(title),
		Inputs:            inputs,
		ServiceContainers: serviceContainerRefs,
	}
	return bitriseModels.StepListItemModel{stepID: step}
}

func workdirInputs(workdir string) []envmanModels.EnvironmentItemModel {
	if workdir == "" {
		return nil
	}
	return []envmanModels.EnvironmentItemModel{{"working_dir": workdir}}
}

func generateDBSetupScript(descriptor configDescriptor) string {
	runner := "rake"
	if descriptor.hasRails {
		runner = "rails"
	}
	dbCommand := runner + " db:create db:schema:load"
	if descriptor.hasBundler {
		dbCommand = "bundle exec " + runner + " db:create db:schema:load"
	}

	return fmt.Sprintf(`#!/usr/bin/env bash
set -euxo pipefail

%s`, dbCommand)
}

func generateTestScript(descriptor configDescriptor) string {
	testCommand := ""
	switch descriptor.testFramework {
	case "rspec":
		if descriptor.hasBundler {
			testCommand = "bundle exec rspec"
		} else {
			testCommand = "rspec"
		}
	case "minitest":
		if descriptor.hasRails {
			if descriptor.hasBundler {
				testCommand = "bundle exec rails test"
			} else {
				testCommand = "rails test"
			}
		} else if descriptor.hasRakefile {
			if descriptor.hasBundler {
				testCommand = "bundle exec rake test"
			} else {
				testCommand = "rake test"
			}
		} else {
			if descriptor.hasBundler {
				testCommand = "bundle exec ruby -Itest test/**/*_test.rb"
			} else {
				testCommand = "ruby -Itest test/**/*_test.rb"
			}
		}
	default:
		// Default to rake if Rakefile exists
		if descriptor.hasRakefile {
			if descriptor.hasBundler {
				testCommand = "bundle exec rake test"
			} else {
				testCommand = "rake test"
			}
		} else {
			return ""
		}
	}

	return fmt.Sprintf(`#!/usr/bin/env bash
set -euxo pipefail

%s`, testCommand)
}

func buildAppEnvs(databases []databaseGem, ymlInfo databaseYMLInfo, mongoidInfo mongoidYMLInfo) []envmanModels.EnvironmentItemModel {
	hasRelational := hasRelationalDB(databases)
	hasMongoidURL := mongoidInfo.connectionURLEnvKey != ""
	if !hasRelational && !hasMongoidURL {
		return nil
	}

	var envs []envmanModels.EnvironmentItemModel

	if hasRelational {
		// Host env var: use name from database.yml or default to DB_HOST
		hostEnvName := "DB_HOST"
		if ymlInfo.hostEnvVar.name != "" {
			hostEnvName = ymlInfo.hostEnvVar.name
		}
		// Script steps run on the host machine, not inside Docker, so they connect to service
		// containers via mapped ports. Most databases work with "localhost", but MySQL treats
		// "localhost" as a Unix socket path — "127.0.0.1" forces TCP/IP.
		for _, db := range databases {
			if db.isRelationalDB && db.containerName != "" {
				hostValue := "localhost"
				if db.hostValue != "" {
					hostValue = db.hostValue
				}
				envs = append(envs, envmanModels.EnvironmentItemModel{hostEnvName: hostValue})
				break
			}
		}

		// Username env var
		if ymlInfo.usernameEnvVar.name != "" {
			envs = append(envs, envmanModels.EnvironmentItemModel{ymlInfo.usernameEnvVar.name: ymlInfo.usernameEnvVar.defaultValue})
		}

		// Password env var
		if ymlInfo.passwordEnvVar.name != "" {
			envs = append(envs, envmanModels.EnvironmentItemModel{ymlInfo.passwordEnvVar.name: ymlInfo.passwordEnvVar.defaultValue})
		}

		// Connection URL env vars for databases with a standard URL convention (e.g. REDIS_URL)
		for _, db := range databases {
			if db.connectionURLEnvKey != "" {
				envs = append(envs, envmanModels.EnvironmentItemModel{db.connectionURLEnvKey: db.connectionURL})
			}
		}
	}

	// MongoDB connection URL parsed from config/mongoid.yml
	if hasMongoidURL {
		envs = append(envs, envmanModels.EnvironmentItemModel{mongoidInfo.connectionURLEnvKey: mongoidInfo.connectionURL})
	}

	return envs
}

func buildContainerDefinitions(databases []databaseGem, ymlInfo databaseYMLInfo) map[string]bitriseModels.Container {
	containers := map[string]bitriseModels.Container{}
	for _, db := range databases {
		if db.containerName == "" {
			continue
		}
		def := bitriseModels.Container{
			Type:    "service",
			Image:   db.image,
			Ports:   db.ports,
			Options: db.healthCheck,
		}

		// Set container env var referencing the app-level env var
		if db.containerEnvKey != "" && ymlInfo.passwordEnvVar.name != "" {
			def.Envs = []envmanModels.EnvironmentItemModel{
				{db.containerEnvKey: "$" + ymlInfo.passwordEnvVar.name},
			}
		}

		containers[db.containerName] = def
	}
	return containers
}
