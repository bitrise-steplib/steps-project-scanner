package ios

import (
	"github.com/bitrise-io/bitrise-init/models"
)

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	DetectResult DetectResult

	ConfigDescriptors []ConfigDescriptor

	ExcludeAppIcon            bool
	SuppressPodFileParseError bool
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (scanner *Scanner) Name() string {
	return string(XcodeProjectTypeIOS)
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	result, err := ParseProjects(XcodeProjectTypeIOS, searchDir, scanner.ExcludeAppIcon, scanner.SuppressPodFileParseError)
	if err != nil {
		return false, err
	}

	if len(result.Projects) == 0 {
		result, err = ParseSPMProject(XcodeProjectTypeIOS, searchDir)
		if err != nil {
			return false, err
		}
	}

	scanner.DetectResult = result
	detected := len(result.Projects) > 0
	return detected, nil
}

// ExcludedScannerNames ...
func (scanner *Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	options, configDescriptors, icons, warnings, err := GenerateOptions(XcodeProjectTypeIOS, scanner.DetectResult)
	if err != nil {
		return models.OptionNode{}, warnings, nil, err
	}

	scanner.ConfigDescriptors = configDescriptors

	return options, warnings, icons, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionNode {
	return GenerateDefaultOptions(XcodeProjectTypeIOS)
}

func (scanner *Scanner) Configs(sshKeyActivation models.SSHKeyActivation) (models.BitriseConfigMap, error) {
	return GenerateConfig(XcodeProjectTypeIOS, scanner.ConfigDescriptors, sshKeyActivation)
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	return GenerateDefaultConfig(XcodeProjectTypeIOS)
}

// GetProjectType returns the project_type property used in a bitrise config
func (scanner *Scanner) GetProjectType() string {
	return string(XcodeProjectTypeIOS)
}
