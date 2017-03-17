package macos

import "github.com/bitrise-core/bitrise-init/models"
import "github.com/bitrise-core/bitrise-init/scanners/xcode"

// ScannerName ...
const ScannerName = "macos"

// Scanner ...
type Scanner struct {
	xcode.Scanner
}

// NewScanner ...
func NewScanner() *Scanner {
	return &Scanner{xcode.Scanner{ProjectType: xcode.ProjectTypeMacOS}}
}

// Name ...
func (scanner *Scanner) Name() string { return ScannerName }

// DetectPlatform ...
func (scanner *Scanner) DetectPlatform(searchDir string) (bool, error) {
	return scanner.CommonDetectPlatform(searchDir)
}

// Options ...
func (scanner *Scanner) Options() (models.OptionModel, models.Warnings, error) {
	return scanner.CommonOptions()
}

// DefaultOptions ...
func (scanner *Scanner) DefaultOptions() models.OptionModel {
	return scanner.CommonDefaultOptions()
}

// Configs ...
func (scanner *Scanner) Configs() (models.BitriseConfigMap, error) {
	return scanner.CommonConfigs()
}

// DefaultConfigs ...
func (scanner *Scanner) DefaultConfigs() (models.BitriseConfigMap, error) {
	return scanner.CommonDefaultConfigs()
}
