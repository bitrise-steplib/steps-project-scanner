package macos

import (
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
)

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	detectResult ios.DetectResult

	configDescriptors []ios.ConfigDescriptor
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (Scanner) Name() string {
	return string(ios.XcodeProjectTypeMacOS)
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	result, err := ios.ParseProjects(ios.XcodeProjectTypeMacOS, searchDir, true, false)
	if err != nil {
		return false, err
	}

	if len(result.Projects) == 0 {
		result, err = ios.ParseSPMProject(ios.XcodeProjectTypeMacOS, searchDir)
		if err != nil {
			return false, err
		}
	}

	scanner.detectResult = result
	detected := len(result.Projects) > 0
	return detected, err
}

// ExcludedScannerNames ...
func (Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionNode, models.Warnings, models.Icons, error) {
	options, configDescriptors, _, warnings, err := ios.GenerateOptions(ios.XcodeProjectTypeMacOS, scanner.detectResult)
	if err != nil {
		return models.OptionNode{}, warnings, nil, err
	}

	scanner.configDescriptors = configDescriptors

	return options, warnings, nil, nil
}

func (Scanner) DefaultOptions() models.OptionNode {
	return ios.GenerateDefaultOptions(ios.XcodeProjectTypeMacOS)
}

func (scanner *Scanner) Configs(repoAccess models.RepoAccess) (models.BitriseConfigMap, error) {
	return ios.GenerateConfig(ios.XcodeProjectTypeMacOS, scanner.configDescriptors, repoAccess)
}

func (Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	return ios.GenerateDefaultConfig(ios.XcodeProjectTypeMacOS)
}
