package models

import (
	bitriseModels "github.com/bitrise-io/bitrise/models"
)

// ScanResultModel ...
type ScanResultModel struct {
	OptionsMap map[string][]OptionModel                             `json:"options,omitempty" yaml:"options,omitempty"`
	ConfigsMap map[string]map[string]bitriseModels.BitriseDataModel `json:"configs,omitempty" yaml:"configs,omitempty"`
}

/*
- key: project_path
  title: Project (or Workspace) path
  envkey: BITRISE_PROJECT_PATH
  valuemap:
    ~/Develop/bitrise/sample-apps/sample-apps-ios-cocoapods/SampleAppWithCocoapods/SampleAppWithCocoapods.xcodeproj:
    - key: scheme
      title: Scheme name
      envkey: BITRISE_SCHEME
      valuemap:
        SampleAppWithCocoapods: []
*/

// OptionValueMap ...
type OptionValueMap map[string][]OptionModel

// OptionModel ...
type OptionModel struct {
	Key    string `json:"key,omitempty" yaml:"key,omitempty"`
	Title  string `json:"title,omitempty"  yaml:"title,omitempty"`
	EnvKey string `json:"env_key,omitempty"  yaml:"env_key,omitempty"`

	ValueMap OptionValueMap `json:"value_map,omitempty"  yaml:"value_map,omitempty"`
	Config   string         `json:"config,omitempty"  yaml:"config,omitempty"`
}

// NewOptionModel ...
func NewOptionModel(key, title, envKey string) OptionModel {
	return OptionModel{
		Key:    key,
		Title:  title,
		EnvKey: envKey,

		ValueMap: OptionValueMap{},
	}
}

// NewEmptyOptionModel ...
func NewEmptyOptionModel() OptionModel {
	return OptionModel{
		ValueMap: OptionValueMap{},
	}
}

// AddValueMapItems ...
func (option *OptionModel) AddValueMapItems(value string, options ...OptionModel) {
	nestedOptions := option.ValueMap[value]
	nestedOptions = append(nestedOptions, options...)
	option.ValueMap[value] = nestedOptions
}

// GetValues ...
func (option OptionModel) GetValues() []string {
	if option.Config != "" {
		return []string{option.Config}
	}

	values := []string{}
	for value := range option.ValueMap {
		values = append(values, value)
	}
	return values
}
