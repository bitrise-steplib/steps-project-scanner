package reactnative

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/bitrise-init/steps"
	"github.com/bitrise-io/bitrise-init/utility"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/xcode-project/serialized"
	"gopkg.in/yaml.v2"
)

const (
	expoConfigName                   = "react-native-expo-config"
	expoDefaultConfigName            = "default-" + expoConfigName
	expoWithExpoKitDefaultConfigName = "default-react-native-expo-expo-kit-config"
)

func appJSONError(appJSONPth, reason, explanation string) error {
	return fmt.Errorf("app.json file (%s) %s\n%s", appJSONPth, reason, explanation)
}

// expoOptions implements ScannerInterface.Options function for Expo based React Native projects.
func (scanner *Scanner) expoOptions() (models.OptionNode, models.Warnings, error) {
	warnings := models.Warnings{}

	// we need to know if the project uses the Expo Kit,
	// since its usage differentiates the eject process and the config options
	usesExpoKit := false

	fileList, err := utility.ListPathInDirSortedByComponents(scanner.searchDir, true)
	if err != nil {
		return models.OptionNode{}, warnings, err
	}

	filters := []utility.FilterFunc{
		utility.ExtensionFilter(".js", true),
		utility.ComponentFilter("node_modules", false),
	}
	sourceFiles, err := utility.FilterPaths(fileList, filters...)
	if err != nil {
		return models.OptionNode{}, warnings, err
	}

	re := regexp.MustCompile(`import .* from 'expo'`)

SourceFileLoop:
	for _, sourceFile := range sourceFiles {
		f, err := os.Open(sourceFile)
		if err != nil {
			return models.OptionNode{}, warnings, err
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				log.Warnf("Failed to close: %s, error: %s", f.Name(), err)
			}
		}()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if match := re.FindString(scanner.Text()); match != "" {
				usesExpoKit = true
				break SourceFileLoop
			}
		}
		if err := scanner.Err(); err != nil {
			return models.OptionNode{}, warnings, err
		}
	}

	scanner.usesExpoKit = usesExpoKit
	log.TPrintf("Uses ExpoKit: %v", usesExpoKit)

	// ensure app.json contains the required information (for non interactive eject)
	// and predict the ejected project name
	var projectName string

	rootDir := filepath.Dir(scanner.packageJSONPth)
	appJSONPth := filepath.Join(rootDir, "app.json")
	appJSON, err := fileutil.ReadStringFromFile(appJSONPth)
	if err != nil {
		return models.OptionNode{}, warnings, err
	}
	var app serialized.Object
	if err := json.Unmarshal([]byte(appJSON), &app); err != nil {
		return models.OptionNode{}, warnings, err
	}

	if usesExpoKit {
		// if the project uses Expo Kit app.json needs to contain expo/ios/bundleIdentifier and expo/android/package entries
		// to be able to eject in non interactive mode
		errorMessage := `If the project uses Expo Kit the app.json file needs to contain:
- expo/name
- expo/ios/bundleIdentifier
- expo/android/package
entries.`

		expoObj, err := app.Object("expo")
		if err != nil {
			return models.OptionNode{}, warnings, appJSONError(appJSONPth, "missing expo entry", errorMessage)
		}
		projectName, err = expoObj.String("name")
		if err != nil || projectName == "" {
			return models.OptionNode{}, warnings, appJSONError(appJSONPth, "missing or empty expo/name entry", errorMessage)
		}

		iosObj, err := expoObj.Object("ios")
		if err != nil {
			return models.OptionNode{}, warnings, appJSONError(appJSONPth, "missing expo/ios entry", errorMessage)
		}
		bundleID, err := iosObj.String("bundleIdentifier")
		if err != nil || bundleID == "" {
			return models.OptionNode{}, warnings, appJSONError(appJSONPth, "missing or empty expo/ios/bundleIdentifier entry", errorMessage)
		}

		androidObj, err := expoObj.Object("android")
		if err != nil {
			return models.OptionNode{}, warnings, appJSONError(appJSONPth, "missing expo/android entry", errorMessage)
		}
		packageName, err := androidObj.String("package")
		if err != nil || packageName == "" {
			return models.OptionNode{}, warnings, appJSONError(appJSONPth, "missing or empty expo/android/package entry", errorMessage)
		}
	} else {
		// if the project does not use Expo Kit app.json needs to contain name and displayName entries
		// to be able to eject in non interactive mode
		errorMessage := `The app.json file needs to contain:
- name
- displayName
entries.`

		projectName, err = app.String("name")
		if err != nil || projectName == "" {
			return models.OptionNode{}, warnings, appJSONError(appJSONPth, "missing or empty name entry", errorMessage)
		}
		displayName, err := app.String("displayName")
		if err != nil || displayName == "" {
			return models.OptionNode{}, warnings, appJSONError(appJSONPth, "missing or empty displayName entry", errorMessage)
		}
	}

	log.TPrintf("Project name: %v", projectName)

	// ios options
	projectPathOption := models.NewOption(ios.ProjectPathInputTitle, ios.ProjectPathInputEnvKey)
	schemeOption := models.NewOption(ios.SchemeInputTitle, ios.SchemeInputEnvKey)

	if usesExpoKit {
		projectName = strings.ToLower(regexp.MustCompile(`(?i:[^a-z0-9_\-])`).ReplaceAllString(projectName, "-"))
		projectPathOption.AddOption(filepath.Join("./", "ios", projectName+".xcworkspace"), schemeOption)
	} else {
		projectPathOption.AddOption(filepath.Join("./", "ios", projectName+".xcodeproj"), schemeOption)
	}

	developmentTeamOption := models.NewOption("iOS Development team", "BITRISE_IOS_DEVELOPMENT_TEAM")
	schemeOption.AddOption(projectName, developmentTeamOption)

	exportMethodOption := models.NewOption(ios.IosExportMethodInputTitle, ios.ExportMethodInputEnvKey)
	developmentTeamOption.AddOption("_", exportMethodOption)

	// android options
	packageJSONDir := filepath.Dir(scanner.packageJSONPth)
	relPackageJSONDir, err := utility.RelPath(scanner.searchDir, packageJSONDir)
	if err != nil {
		return models.OptionNode{}, warnings, fmt.Errorf("Failed to get relative package.json dir path, error: %s", err)
	}
	if relPackageJSONDir == "." {
		// package.json placed in the search dir, no need to change-dir in the workflows
		relPackageJSONDir = ""
	}

	var moduleOption *models.OptionNode
	if relPackageJSONDir == "" {
		projectLocationOption := models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputEnvKey)
		for _, exportMethod := range ios.IosExportMethods {
			exportMethodOption.AddOption(exportMethod, projectLocationOption)
		}

		moduleOption = models.NewOption(android.ModuleInputTitle, android.ModuleInputEnvKey)
		projectLocationOption.AddOption("./android", moduleOption)
	} else {
		workDirOption := models.NewOption("Project root directory (the directory of the project app.json/package.json file)", "WORKDIR")
		for _, exportMethod := range ios.IosExportMethods {
			exportMethodOption.AddOption(exportMethod, workDirOption)
		}

		projectLocationOption := models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputEnvKey)
		workDirOption.AddOption(relPackageJSONDir, projectLocationOption)

		moduleOption = models.NewOption(android.ModuleInputTitle, android.ModuleInputEnvKey)
		projectLocationOption.AddOption(filepath.Join(relPackageJSONDir, "android"), moduleOption)
	}

	buildVariantOption := models.NewOption(android.VariantInputTitle, android.VariantInputEnvKey)
	moduleOption.AddOption("app", buildVariantOption)

	// expo options
	if scanner.usesExpoKit {
		userNameOption := models.NewOption("Expo username", "EXPO_USERNAME")
		buildVariantOption.AddOption("Release", userNameOption)

		passwordOption := models.NewOption("Expo password", "EXPO_PASSWORD")
		userNameOption.AddOption("_", passwordOption)

		configOption := models.NewConfigOption(expoConfigName, nil)
		passwordOption.AddConfig("_", configOption)
	} else {
		configOption := models.NewConfigOption(expoConfigName, nil)
		buildVariantOption.AddConfig("Release", configOption)
	}

	return *projectPathOption, warnings, nil
}

// expoConfigs implements ScannerInterface.Configs function for Expo based React Native projects.
func (scanner *Scanner) expoConfigs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	// determine workdir
	packageJSONDir := filepath.Dir(scanner.packageJSONPth)
	relPackageJSONDir, err := utility.RelPath(scanner.searchDir, packageJSONDir)
	if err != nil {
		return models.BitriseConfigMap{}, fmt.Errorf("Failed to get relative package.json dir path, error: %s", err)
	}
	if relPackageJSONDir == "." {
		// package.json placed in the search dir, no need to change-dir in the workflows
		relPackageJSONDir = ""
	}
	log.TPrintf("Working directory: %v", relPackageJSONDir)

	workdirEnvList := []envmanModels.EnvironmentItemModel{}
	if relPackageJSONDir != "" {
		workdirEnvList = append(workdirEnvList, envmanModels.EnvironmentItemModel{workDirInputKey: relPackageJSONDir})
	}

	// determine dependency manager step
	hasYarnLockFile := false
	if exist, err := pathutil.IsPathExists(filepath.Join(relPackageJSONDir, "yarn.lock")); err != nil {
		log.Warnf("Failed to check if yarn.lock file exists in the workdir: %s", err)
		log.TPrintf("Dependency manager: npm")
	} else if exist {
		log.TPrintf("Dependency manager: yarn")
		hasYarnLockFile = true
	} else {
		log.TPrintf("Dependency manager: npm")
	}

	// find test script in package.json file
	b, err := fileutil.ReadBytesFromFile(scanner.packageJSONPth)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}
	var packageJSON serialized.Object
	if err := json.Unmarshal([]byte(b), &packageJSON); err != nil {
		return models.BitriseConfigMap{}, err
	}

	hasTest := false
	if scripts, err := packageJSON.Object("scripts"); err == nil {
		if _, err := scripts.String("test"); err == nil {
			hasTest = true
		}
	}
	log.TPrintf("Test script found in package.json: %v", hasTest)

	if !hasTest {
		// if the project has no test script defined,
		// we can only provide deploy like workflow,
		// so that is going to be the primary workflow

		configBuilder := models.NewDefaultConfigBuilder()
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)

		if hasYarnLockFile {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		} else {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		}

		projectDir := relPackageJSONDir
		if relPackageJSONDir == "" {
			projectDir = "./"
		}
		if scanner.usesExpoKit {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.ExpoDetachStepListItem(
				envmanModels.EnvironmentItemModel{"project_path": projectDir},
				envmanModels.EnvironmentItemModel{"user_name": "$EXPO_USERNAME"},
				envmanModels.EnvironmentItemModel{"password": "$EXPO_PASSWORD"},
				envmanModels.EnvironmentItemModel{"run_publish": "yes"},
			))
		} else {
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.ExpoDetachStepListItem(
				envmanModels.EnvironmentItemModel{"project_path": projectDir},
			))
		}

		// android build
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
			envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
		))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + android.ProjectLocationInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))

		// ios build
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

		if scanner.usesExpoKit {
			// in case of expo kit rn project expo eject generates an ios project with Podfile
			configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.CocoapodsInstallStepListItem())
		}

		xcodeArchiveInputs := []envmanModels.EnvironmentItemModel{
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
			envmanModels.EnvironmentItemModel{ios.ExportMethodInputKey: "$" + ios.ExportMethodInputEnvKey},
			envmanModels.EnvironmentItemModel{"force_team_id": "$BITRISE_IOS_DEVELOPMENT_TEAM"},
		}
		if !scanner.usesExpoKit {
			// in case of plain rn project new xcode build system needs to be turned off
			xcodeArchiveInputs = append(xcodeArchiveInputs, envmanModels.EnvironmentItemModel{"xcodebuild_options": "-UseModernBuildSystem=NO"})
		}
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.XcodeArchiveStepListItem(xcodeArchiveInputs...))

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)
		configBuilder.SetWorkflowDescriptionTo(models.PrimaryWorkflowID, deployWorkflowDescription)

		bitriseDataModel, err := configBuilder.Generate(scannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(bitriseDataModel)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configMap[expoConfigName] = string(data)

		return configMap, nil
	}

	// primary workflow
	configBuilder := models.NewDefaultConfigBuilder()
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
	if hasYarnLockFile {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
	} else {
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "test"})...))
	}
	configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

	// deploy workflow
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
	if hasYarnLockFile {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.YarnStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
	} else {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(append(workdirEnvList, envmanModels.EnvironmentItemModel{"command": "install"})...))
	}

	projectDir := relPackageJSONDir
	if relPackageJSONDir == "" {
		projectDir = "./"
	}
	if scanner.usesExpoKit {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ExpoDetachStepListItem(
			envmanModels.EnvironmentItemModel{"project_path": projectDir},
			envmanModels.EnvironmentItemModel{"user_name": "$EXPO_USERNAME"},
			envmanModels.EnvironmentItemModel{"password": "$EXPO_PASSWORD"},
			envmanModels.EnvironmentItemModel{"run_publish": "yes"},
		))
	} else {
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ExpoDetachStepListItem(
			envmanModels.EnvironmentItemModel{"project_path": projectDir},
		))
	}

	// android build
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
		envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
	))
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
		envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + android.ProjectLocationInputEnvKey},
		envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
		envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
	))

	// ios build
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

	if scanner.usesExpoKit {
		// in case of expo kit rn project expo eject generates an ios project with Podfile
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CocoapodsInstallStepListItem())
	}

	xcodeArchiveInputs := []envmanModels.EnvironmentItemModel{
		envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
		envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
		envmanModels.EnvironmentItemModel{ios.ExportMethodInputKey: "$" + ios.ExportMethodInputEnvKey},
		envmanModels.EnvironmentItemModel{"force_team_id": "$BITRISE_IOS_DEVELOPMENT_TEAM"},
	}
	if !scanner.usesExpoKit {
		xcodeArchiveInputs = append(xcodeArchiveInputs, envmanModels.EnvironmentItemModel{"xcodebuild_options": "-UseModernBuildSystem=NO"})
	}
	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(xcodeArchiveInputs...))

	configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)
	configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)

	bitriseDataModel, err := configBuilder.Generate(scannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(bitriseDataModel)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	configMap[expoConfigName] = string(data)

	return configMap, nil
}

// expoDefaultOptions implements ScannerInterface.DefaultOptions function for Expo based React Native projects.
func (Scanner) expoDefaultOptions() models.OptionNode {
	expoKitOption := models.NewOption("Project uses Expo Kit (any js file imports expo dependency)?", "USES_EXPO_KIT")

	// with Expo Kit
	{
		// ios options
		workspacePathOption := models.NewOption("The iOS workspace path generated ny the 'expo eject' process", ios.ProjectPathInputEnvKey)
		expoKitOption.AddOption("yes", workspacePathOption)

		schemeOption := models.NewOption("The iOS scheme name generated by the 'expo eject' process", ios.SchemeInputEnvKey)
		workspacePathOption.AddOption("_", schemeOption)

		exportMethodOption := models.NewOption(ios.IosExportMethodInputTitle, ios.ExportMethodInputEnvKey)
		schemeOption.AddOption("_", exportMethodOption)

		// android options
		workDirOption := models.NewOption("Project root directory (the directory of the project app.json/package.json file)", "WORKDIR")
		for _, exportMethod := range ios.IosExportMethods {
			exportMethodOption.AddOption(exportMethod, workDirOption)
		}

		projectLocationOption := models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputEnvKey)
		workDirOption.AddOption("_", projectLocationOption)

		moduleOption := models.NewOption(android.ModuleInputTitle, android.ModuleInputEnvKey)
		projectLocationOption.AddOption("./android", moduleOption)

		buildVariantOption := models.NewOption(android.VariantInputTitle, android.VariantInputEnvKey)
		moduleOption.AddOption("app", buildVariantOption)

		// Expo CLI options
		userNameOption := models.NewOption("Expo username", "EXPO_USERNAME")
		buildVariantOption.AddOption("Release", userNameOption)

		passwordOption := models.NewOption("Expo password", "EXPO_PASSWORD")
		userNameOption.AddOption("_", passwordOption)

		configOption := models.NewConfigOption(expoWithExpoKitDefaultConfigName, nil)
		passwordOption.AddConfig("_", configOption)
	}

	// without Expo Kit
	{
		// ios options
		projectPathOption := models.NewOption("The iOS project path generated ny the 'expo eject' process", ios.ProjectPathInputEnvKey)
		expoKitOption.AddOption("no", projectPathOption)

		schemeOption := models.NewOption("The iOS scheme name generated by the 'expo eject' process", ios.SchemeInputEnvKey)
		projectPathOption.AddOption("_", schemeOption)

		exportMethodOption := models.NewOption(ios.IosExportMethodInputTitle, ios.ExportMethodInputEnvKey)
		schemeOption.AddOption("_", exportMethodOption)

		// android options
		workDirOption := models.NewOption("Project root directory (the directory of the project app.json/package.json file)", "WORKDIR")
		for _, exportMethod := range ios.IosExportMethods {
			exportMethodOption.AddOption(exportMethod, workDirOption)
		}

		projectLocationOption := models.NewOption(android.ProjectLocationInputTitle, android.ProjectLocationInputEnvKey)
		workDirOption.AddOption("_", projectLocationOption)

		moduleOption := models.NewOption(android.ModuleInputTitle, android.ModuleInputEnvKey)
		projectLocationOption.AddOption("./android", moduleOption)

		buildVariantOption := models.NewOption(android.VariantInputTitle, android.VariantInputEnvKey)
		moduleOption.AddOption("app", buildVariantOption)

		configOption := models.NewConfigOption(expoDefaultConfigName, nil)
		buildVariantOption.AddConfig("Release", configOption)
	}

	return *expoKitOption
}

// expoDefaultConfigs implements ScannerInterface.DefaultConfigs function for Expo based React Native projects.
func (Scanner) expoDefaultConfigs() (models.BitriseConfigMap, error) {
	configMap := models.BitriseConfigMap{}

	// with Expo Kit
	{
		// primary workflow
		configBuilder := models.NewDefaultConfigBuilder()

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "install"}))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "test"}))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		// deploy workflow
		configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "install"}))

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ExpoDetachStepListItem(
			envmanModels.EnvironmentItemModel{"project_path": "$WORKDIR"},
			envmanModels.EnvironmentItemModel{"user_name": "$EXPO_USERNAME"},
			envmanModels.EnvironmentItemModel{"password": "$EXPO_PASSWORD"},
			envmanModels.EnvironmentItemModel{"run_publish": "yes"},
		))

		// android build
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
			envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
		))
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + android.ProjectLocationInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))

		// ios build
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CocoapodsInstallStepListItem())

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ExportMethodInputKey: "$" + ios.ExportMethodInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
		))

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

		bitriseDataModel, err := configBuilder.Generate(scannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(bitriseDataModel)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configMap[expoWithExpoKitDefaultConfigName] = string(data)
	}

	// without Expo Kit
	{
		// primary workflow
		configBuilder := models.NewDefaultConfigBuilder()

		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "install"}))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "test"}))
		configBuilder.AppendStepListItemsTo(models.PrimaryWorkflowID, steps.DefaultDeployStepList(false)...)

		// deploy workflow
		configBuilder.SetWorkflowDescriptionTo(models.DeployWorkflowID, deployWorkflowDescription)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultPrepareStepList(false)...)
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.NpmStepListItem(envmanModels.EnvironmentItemModel{workDirInputKey: "$WORKDIR"}, envmanModels.EnvironmentItemModel{"command": "install"}))

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.ExpoDetachStepListItem(envmanModels.EnvironmentItemModel{"project_path": "$WORKDIR"}))

		// android build
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.InstallMissingAndroidToolsStepListItem(
			envmanModels.EnvironmentItemModel{android.GradlewPathInputKey: "$" + android.ProjectLocationInputEnvKey + "/gradlew"},
		))
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.AndroidBuildStepListItem(
			envmanModels.EnvironmentItemModel{android.ProjectLocationInputKey: "$" + android.ProjectLocationInputEnvKey},
			envmanModels.EnvironmentItemModel{android.ModuleInputKey: "$" + android.ModuleInputEnvKey},
			envmanModels.EnvironmentItemModel{android.VariantInputKey: "$" + android.VariantInputEnvKey},
		))

		// ios build
		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.CertificateAndProfileInstallerStepListItem())

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.XcodeArchiveStepListItem(
			envmanModels.EnvironmentItemModel{ios.ProjectPathInputKey: "$" + ios.ProjectPathInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.SchemeInputKey: "$" + ios.SchemeInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ExportMethodInputKey: "$" + ios.ExportMethodInputEnvKey},
			envmanModels.EnvironmentItemModel{ios.ConfigurationInputKey: "Release"},
		))

		configBuilder.AppendStepListItemsTo(models.DeployWorkflowID, steps.DefaultDeployStepList(false)...)

		bitriseDataModel, err := configBuilder.Generate(scannerName)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		data, err := yaml.Marshal(bitriseDataModel)
		if err != nil {
			return models.BitriseConfigMap{}, err
		}

		configMap[expoDefaultConfigName] = string(data)
	}

	return configMap, nil
}
