package macos

import (
	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/scanners/xcode"
	"github.com/bitrise-core/bitrise-init/utility"
)

//------------------
// ScannerInterface
//------------------

// Scanner ...
type Scanner struct {
	searchDir         string
	configDescriptors []xcode.ConfigDescriptor
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{}
}

// Name ...
func (scanner *Scanner) Name() string {
	return string(utility.XcodeProjectTypeMacOS)
}

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	scanner.searchDir = searchDir

	detected, err := xcode.Detect(utility.XcodeProjectTypeMacOS, searchDir)
	if err != nil {
		return false, err
	}

	return detected, nil
}

// ExcludedScannerNames ...
func (scanner *Scanner) ExcludedScannerNames() []string {
	return []string{}
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	options, configDescriptors, warnings, err := xcode.GenerateOptions(utility.XcodeProjectTypeMacOS, scanner.searchDir)
	if err != nil {
		return models.OptionModel{}, warnings, err
	}

	scanner.configDescriptors = configDescriptors

	return options, warnings, nil
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	return xcode.GenerateDefaultOptions(utility.XcodeProjectTypeMacOS)
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	return xcode.GenerateConfig(utility.XcodeProjectTypeMacOS, scanner.configDescriptors)
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	return xcode.GenerateDefaultConfig(utility.XcodeProjectTypeMacOS)
}
