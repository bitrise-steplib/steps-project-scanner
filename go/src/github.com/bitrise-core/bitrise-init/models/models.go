package models

// OptionModel ...
type OptionModel struct {
	Title  string `json:"title,omitempty"  yaml:"title,omitempty"`
	EnvKey string `json:"env_key,omitempty"  yaml:"env_key,omitempty"`

	ValueMap OptionValueMap `json:"value_map,omitempty"  yaml:"value_map,omitempty"`
	Config   string         `json:"config,omitempty"  yaml:"config,omitempty"`
}

// OptionValueMap ...
type OptionValueMap map[string]OptionModel

// BitriseConfigMap ...
type BitriseConfigMap map[string]string

// Warnings ...
type Warnings []string

// Errors ...
type Errors []string

// ScanResultModel ...
type ScanResultModel struct {
	OptionsMap  map[string]OptionModel      `json:"options,omitempty" yaml:"options,omitempty"`
	ConfigsMap  map[string]BitriseConfigMap `json:"configs,omitempty" yaml:"configs,omitempty"`
	WarningsMap map[string]Warnings         `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	ErrorsMap   map[string]Errors           `json:"errors,omitempty" yaml:"errors,omitempty"`
}
