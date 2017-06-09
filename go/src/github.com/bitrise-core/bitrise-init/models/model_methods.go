package models

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bitrise-core/bitrise-init/steps"
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
)

const (
	// FormatVersion ...
	FormatVersion = bitriseModels.Version

	defaultSteplibSource = "https://github.com/bitrise-io/bitrise-steplib.git"
)

// ---
// OptionModel

// NewOption ...
func NewOption(title, envKey string) *OptionModel {
	return &OptionModel{
		Title:          title,
		EnvKey:         envKey,
		ChildOptionMap: map[string]*OptionModel{},
		Components:     []string{},
	}
}

// NewConfigOption ...
func NewConfigOption(name string) *OptionModel {
	return &OptionModel{
		ChildOptionMap: map[string]*OptionModel{},
		Config:         name,
		Components:     []string{},
	}
}

func (option *OptionModel) String() string {
	if option.Config != "" {
		return fmt.Sprintf(`Config Option:
  config: %s
`, option.Config)
	}

	values := option.GetValues()
	return fmt.Sprintf(`Option:
  title: %s
  env_key: %s
  values: %v
`, option.Title, option.EnvKey, values)
}

// AddOption ...
func (option *OptionModel) AddOption(forValue string, newOption *OptionModel) {
	option.ChildOptionMap[forValue] = newOption

	if newOption != nil {
		newOption.Components = append(option.Components, forValue)

		if option.Head == nil {
			// first option's head is nil
			newOption.Head = option
		} else {
			newOption.Head = option.Head
		}
	}
}

// AddConfig ...
func (option *OptionModel) AddConfig(forValue string, newConfigOption *OptionModel) {
	option.ChildOptionMap[forValue] = newConfigOption

	if newConfigOption != nil {
		newConfigOption.Components = append(option.Components, forValue)

		if option.Head == nil {
			// first option's head is nil
			newConfigOption.Head = option
		} else {
			newConfigOption.Head = option.Head
		}
	}
}

// Parent ...
func (option *OptionModel) Parent() (*OptionModel, string, bool) {
	if option.Head == nil {
		return nil, "", false
	}

	parentComponents := option.Components[:len(option.Components)-1]
	parentOption, ok := option.Head.Child(parentComponents...)
	if !ok {
		return nil, "", false
	}
	underKey := option.Components[len(option.Components)-1:][0]
	return parentOption, underKey, true
}

// Child ...
func (option *OptionModel) Child(components ...string) (*OptionModel, bool) {
	currentOption := option
	for _, component := range components {
		childOption := currentOption.ChildOptionMap[component]
		if childOption == nil {
			return nil, false
		}
		currentOption = childOption
	}
	return currentOption, true
}

// LastChilds ...
func (option *OptionModel) LastChilds() []*OptionModel {
	lastOptions := []*OptionModel{}

	var walk func(option *OptionModel)
	walk = func(option *OptionModel) {
		if len(option.ChildOptionMap) == 0 {
			// no more child, this is the last option in this branch
			lastOptions = append(lastOptions, option)
			return
		}

		for _, childOption := range option.ChildOptionMap {
			if childOption == nil {
				// values are set to this option, but has value without child
				lastOptions = append(lastOptions, option)
				return
			}

			walk(childOption)
		}
	}

	walk(option)

	return lastOptions
}

// Copy ...
func (option *OptionModel) Copy() *OptionModel {
	bytes, err := json.Marshal(*option)
	if err != nil {
		return nil
	}

	var optionCopy OptionModel
	if err := json.Unmarshal(bytes, &optionCopy); err != nil {
		return nil
	}

	return &optionCopy
}

// GetValues ...
func (option *OptionModel) GetValues() []string {
	if option.Config != "" {
		return []string{option.Config}
	}

	values := []string{}
	for value := range option.ChildOptionMap {
		values = append(values, value)
	}
	return values
}

// ---

// ---
// Config Builder

func newDefaultWorkflowBuilder(isIncludeCache bool) *workflowBuilderModel {
	return &workflowBuilderModel{
		PrepareSteps:    steps.DefaultPrepareStepList(isIncludeCache),
		DependencySteps: []bitriseModels.StepListItemModel{},
		MainSteps:       []bitriseModels.StepListItemModel{},
		DeploySteps:     steps.DefaultDeployStepList(isIncludeCache),
	}
}

func newWorkflowBuilder(items ...bitriseModels.StepListItemModel) *workflowBuilderModel {
	return &workflowBuilderModel{
		steps: items,
	}
}

func (builder *workflowBuilderModel) appendPreparStepList(items ...bitriseModels.StepListItemModel) {
	builder.PrepareSteps = append(builder.PrepareSteps, items...)
}

func (builder *workflowBuilderModel) appendDependencyStepList(items ...bitriseModels.StepListItemModel) {
	builder.DependencySteps = append(builder.DependencySteps, items...)
}

func (builder *workflowBuilderModel) appendMainStepList(items ...bitriseModels.StepListItemModel) {
	builder.MainSteps = append(builder.MainSteps, items...)
}

func (builder *workflowBuilderModel) appendDeployStepList(items ...bitriseModels.StepListItemModel) {
	builder.DeploySteps = append(builder.DeploySteps, items...)
}

func (builder *workflowBuilderModel) stepList() []bitriseModels.StepListItemModel {
	if len(builder.steps) > 0 {
		return builder.steps
	}

	stepList := []bitriseModels.StepListItemModel{}
	stepList = append(stepList, builder.PrepareSteps...)
	stepList = append(stepList, builder.DependencySteps...)
	stepList = append(stepList, builder.MainSteps...)
	stepList = append(stepList, builder.DeploySteps...)
	return stepList
}

func (builder *workflowBuilderModel) generate() bitriseModels.WorkflowModel {
	return bitriseModels.WorkflowModel{
		Steps: builder.stepList(),
	}
}

// NewDefaultConfigBuilder ...
func NewDefaultConfigBuilder(isIncludeCache bool) *ConfigBuilderModel {
	return &ConfigBuilderModel{
		workflowBuilderMap: map[WorkflowID]*workflowBuilderModel{
			PrimaryWorkflowID: newDefaultWorkflowBuilder(isIncludeCache),
		},
	}
}

// NewConfigBuilder ...
func NewConfigBuilder(primarySteps []bitriseModels.StepListItemModel) *ConfigBuilderModel {
	return &ConfigBuilderModel{
		workflowBuilderMap: map[WorkflowID]*workflowBuilderModel{
			PrimaryWorkflowID: newWorkflowBuilder(primarySteps...),
		},
	}
}

// AddDefaultWorkflowBuilder ...
func (builder *ConfigBuilderModel) AddDefaultWorkflowBuilder(workflow WorkflowID, isIncludeCache bool) {
	builder.workflowBuilderMap[workflow] = newDefaultWorkflowBuilder(isIncludeCache)
}

// AppendPreparStepListTo ...
func (builder *ConfigBuilderModel) AppendPreparStepListTo(workflow WorkflowID, items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = &workflowBuilderModel{}
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.appendPreparStepList(items...)
}

// AppendDependencyStepListTo ...
func (builder *ConfigBuilderModel) AppendDependencyStepListTo(workflow WorkflowID, items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = &workflowBuilderModel{}
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.appendDependencyStepList(items...)
}

// AppendMainStepListTo ...
func (builder *ConfigBuilderModel) AppendMainStepListTo(workflow WorkflowID, items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = &workflowBuilderModel{}
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.appendMainStepList(items...)
}

// AppendDeployStepListTo ...
func (builder *ConfigBuilderModel) AppendDeployStepListTo(workflow WorkflowID, items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[workflow]
	if workflowBuilder == nil {
		workflowBuilder = &workflowBuilderModel{}
		builder.workflowBuilderMap[workflow] = workflowBuilder
	}
	workflowBuilder.appendDeployStepList(items...)
}

// AppendPreparStepList ...
func (builder *ConfigBuilderModel) AppendPreparStepList(items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[PrimaryWorkflowID]
	if workflowBuilder == nil {
		workflowBuilder = &workflowBuilderModel{}
		builder.workflowBuilderMap[PrimaryWorkflowID] = workflowBuilder
	}
	workflowBuilder.appendPreparStepList(items...)
}

// AppendDependencyStepList ...
func (builder *ConfigBuilderModel) AppendDependencyStepList(items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[PrimaryWorkflowID]
	if workflowBuilder == nil {
		workflowBuilder = &workflowBuilderModel{}
		builder.workflowBuilderMap[PrimaryWorkflowID] = workflowBuilder
	}
	workflowBuilder.appendDependencyStepList(items...)
}

// AppendMainStepList ...
func (builder *ConfigBuilderModel) AppendMainStepList(items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[PrimaryWorkflowID]
	if workflowBuilder == nil {
		workflowBuilder = &workflowBuilderModel{}
		builder.workflowBuilderMap[PrimaryWorkflowID] = workflowBuilder
	}
	workflowBuilder.appendMainStepList(items...)

}

// AppendDeployStepList ...
func (builder *ConfigBuilderModel) AppendDeployStepList(items ...bitriseModels.StepListItemModel) {
	workflowBuilder := builder.workflowBuilderMap[PrimaryWorkflowID]
	if workflowBuilder == nil {
		workflowBuilder = &workflowBuilderModel{}
		builder.workflowBuilderMap[PrimaryWorkflowID] = workflowBuilder
	}
	workflowBuilder.appendDeployStepList(items...)
}

// Generate ...
func (builder *ConfigBuilderModel) Generate(projectType string, appEnvs ...envmanModels.EnvironmentItemModel) (bitriseModels.BitriseDataModel, error) {
	primaryWorkflowBuilder, ok := builder.workflowBuilderMap[PrimaryWorkflowID]
	if !ok || primaryWorkflowBuilder == nil || len(primaryWorkflowBuilder.stepList()) == 0 {
		return bitriseModels.BitriseDataModel{}, errors.New("primary workflow not defined")
	}

	workflows := map[string]bitriseModels.WorkflowModel{}
	for workflowID, workflowBuilder := range builder.workflowBuilderMap {
		workflows[string(workflowID)] = workflowBuilder.generate()
	}

	triggerMap := []bitriseModels.TriggerMapItemModel{
		bitriseModels.TriggerMapItemModel{
			PushBranch: "*",
			WorkflowID: string(PrimaryWorkflowID),
		},
		bitriseModels.TriggerMapItemModel{
			PullRequestSourceBranch: "*",
			WorkflowID:              string(PrimaryWorkflowID),
		},
	}

	app := bitriseModels.AppModel{
		Environments: appEnvs,
	}

	return bitriseModels.BitriseDataModel{
		FormatVersion:        FormatVersion,
		DefaultStepLibSource: defaultSteplibSource,
		ProjectType:          projectType,
		TriggerMap:           triggerMap,
		Workflows:            workflows,
		App:                  app,
	}, nil
}

// ---

// AddError ...
func (result *ScanResultModel) AddError(platform string, errorMessage string) {
	if result.PlatformErrorsMap == nil {
		result.PlatformErrorsMap = map[string]Errors{}
	}
	if result.PlatformErrorsMap[platform] == nil {
		result.PlatformErrorsMap[platform] = []string{}
	}
	result.PlatformErrorsMap[platform] = append(result.PlatformErrorsMap[platform], errorMessage)
}
