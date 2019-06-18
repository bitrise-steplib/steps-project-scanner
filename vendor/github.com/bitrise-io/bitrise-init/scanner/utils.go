package scanner

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/bitrise-io/bitrise-init/models"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/goinp/goinp"
)

func getDefaultValue(opt models.OptionNode) string {
	if opt.Type == models.TypeOptionalSelector {
		return ""
	}

	for key := range opt.ChildOptionMap {
		return key
	}
	return ""
}

func getOptions(opts map[string]*models.OptionNode) (options []string) {
	for key := range opts {
		options = append(options, key)
	}
	return
}

func selectOption(options []string) (string, error) {
	for i, option := range options {
		fmt.Printf("[%d] : %s\n", i+1, option)
	}
	fmt.Printf("Type in the option's number, then hit Enter: ")

	answer, err := goinp.AskForOptionalInput("", false)
	if err != nil {
		return "", err
	}

	optionNo, err := strconv.Atoi(strings.TrimSpace(answer))
	if err != nil {
		return "", fmt.Errorf("failed to parse option number, pick a number from 1-%d", len(options))
	}

	if optionNo-1 < 0 || optionNo-1 >= len(options) {
		return "", fmt.Errorf("invalid option number, pick a number from 1-%d", len(options))
	}

	return options[optionNo-1], nil
}

func askForOptionValue(option models.OptionNode) (string, string, error) {
	const customValueOptionText = "<custom value>"

	// this options is a last element in a tree, contains only config name
	if option.Config != "" {
		return "", option.Config, nil
	}

	optional := option.Type == models.TypeOptionalUserInput || option.Type == models.TypeOptionalSelector

	switch option.Type {
	case models.TypeSelector, models.TypeOptionalSelector:
		fmt.Println("Select \"" + option.Title + "\" from the list:")

		options := getOptions(option.ChildOptionMap)
		if optional {
			options = append(options, customValueOptionText)
		}

		if len(options) == 1 {
			return option.EnvKey, options[0], nil
		}

		selected, err := selectOption(options)
		if err != nil {
			return "", "", err
		}

		if option.Type == models.TypeSelector || selected != customValueOptionText {
			return option.EnvKey, selected, nil
		}

		fallthrough
	case models.TypeUserInput, models.TypeOptionalUserInput:
		suffix := ": "
		if optional {
			suffix = " (optional): "
		}
		fmt.Print("Enter value for \"" + option.Title + "\"" + suffix)

		answer, err := goinp.AskForOptionalInput(getDefaultValue(option), optional)
		return option.EnvKey, strings.TrimSpace(answer), err
	}

	return "", "", fmt.Errorf("invalid input type")
}

// AskForOptions ...
func AskForOptions(options models.OptionNode) (string, []envmanModels.EnvironmentItemModel, error) {
	configPth := ""
	appEnvs := []envmanModels.EnvironmentItemModel{}

	var walkDepth func(models.OptionNode) error
	walkDepth = func(opt models.OptionNode) error {
		optionEnvKey, selectedValue, err := askForOptionValue(opt)
		if err != nil {
			return fmt.Errorf("Failed to ask for value, error: %s", err)
		}

		if opt.Title == "" {
			// last option selected, config got
			configPth = selectedValue
			return nil
		} else if optionEnvKey != "" {
			// env's value selected
			appEnvs = append(appEnvs, envmanModels.EnvironmentItemModel{
				optionEnvKey: selectedValue,
			})
		}

		var nestedOptions *models.OptionNode
		if len(opt.ChildOptionMap) == 1 {
			// auto select the next option
			for _, childOption := range opt.ChildOptionMap {
				nestedOptions = childOption
				break
			}
		} else {
			// go to the next option, based on the selected value
			childOption, found := opt.ChildOptionMap[selectedValue]
			if !found {
				if opt.Type != models.TypeOptionalSelector {
					return nil
				}
				// if user select custom value from the optional list then we need to select any next option
				for _, option := range opt.ChildOptionMap {
					childOption = option
					break
				}
			}
			nestedOptions = childOption
		}

		return walkDepth(*nestedOptions)
	}

	if err := walkDepth(options); err != nil {
		return "", []envmanModels.EnvironmentItemModel{}, err
	}

	if configPth == "" {
		return "", nil, errors.New("no config selected")
	}

	return configPth, appEnvs, nil
}

// AskForConfig ...
func AskForConfig(scanResult models.ScanResultModel) (bitriseModels.BitriseDataModel, error) {

	//
	// Select platform
	platforms := []string{}
	for platform := range scanResult.ScannerToOptionRoot {
		platforms = append(platforms, platform)
	}

	platform := ""
	if len(platforms) == 0 {
		return bitriseModels.BitriseDataModel{}, errors.New("no platform detected")
	} else if len(platforms) == 1 {
		platform = platforms[0]
	} else {
		fmt.Println("Select platform:")
		var err error
		platform, err = selectOption(platforms)
		if err != nil {
			return bitriseModels.BitriseDataModel{}, err
		}
	}
	// ---

	//
	// Select config
	options, ok := scanResult.ScannerToOptionRoot[platform]
	if !ok {
		return bitriseModels.BitriseDataModel{}, fmt.Errorf("invalid platform selected: %s", platform)
	}

	configPth, appEnvs, err := AskForOptions(options)
	if err != nil {
		return bitriseModels.BitriseDataModel{}, err
	}
	// --

	//
	// Build config
	configMap := scanResult.ScannerToBitriseConfigMap[platform]
	configStr := configMap[configPth]

	var config bitriseModels.BitriseDataModel
	if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
		return bitriseModels.BitriseDataModel{}, fmt.Errorf("failed to unmarshal config, error: %s", err)
	}

	config.App.Environments = append(config.App.Environments, appEnvs...)
	// ---

	return config, nil
}
