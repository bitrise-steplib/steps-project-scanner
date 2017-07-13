package android

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-core/bitrise-init/models"
)

// Scanner ...
type Scanner struct {
	SearchDir string
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (Scanner) Name() string {
	return ScannerName
}

// ExcludedScannerNames ...
func (*Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	scanner.SearchDir = searchDir

	buildGradleFiles, err := CollectRootBuildGradleFiles(searchDir)
	if err != nil {
		return false, fmt.Errorf("failed to search for build.gradle files, error: %s", err)
	}

	return (len(buildGradleFiles) > 0), nil
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	return GenerateOptions(scanner.SearchDir)
}

// DefaultOptions ...
func (*Scanner) DefaultOptions() models.OptionModel {
	gradleFileOption := models.NewOption(GradleFileInputTitle, GradleFileInputEnvKey)

	gradlewPthOption := models.NewOption(GradlewPathInputTitle, GradlewPathInputEnvKey)
	gradleFileOption.AddOption("_", gradlewPthOption)

	configOption := models.NewConfigOption(DefaultConfigName)
	gradlewPthOption.AddConfig("_", configOption)

	return *gradleFileOption
}

// Configs ...
func (*Scanner) Configs() (models.BitriseConfigMap, error) {
	configBuilder := GenerateConfigBuilder(true)

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		ConfigName: string(data),
	}, nil
}

// DefaultConfigs ...
func (*Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	configBuilder := GenerateConfigBuilder(true)

	config, err := configBuilder.Generate(ScannerName)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return models.BitriseConfigMap{}, err
	}

	return models.BitriseConfigMap{
		DefaultConfigName: string(data),
	}, nil
}
