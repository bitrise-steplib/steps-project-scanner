package ios

import (
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/steps"
	bitriseModels "github.com/bitrise-io/bitrise/v2/models"
	envmanModels "github.com/bitrise-io/envman/v2/models"
)

const (
	TestRepetitionModeKey                 = "test_repetition_mode"
	TestRepetitionModeRetryOnFailureValue = "retry_on_failure"
	BuildForTestDestinationKey            = "destination"
	BuildForTestDestinationValue          = "platform=iOS Simulator,name=iPhone 8 Plus,OS=latest"
	GenericBuildForTestDestinationValue   = "generic/platform=iOS Simulator"
	AutomaticCodeSigningKey               = "automatic_code_signing"
	AutomaticCodeSigningValue             = "api-key"
	CacheLevelKey                         = "cache_level"
	CacheLevelNone                        = "none"
	ShardCountKey                         = "shard_count"
	ShardCountValue                       = "$TEST_SHARD_COUNT"
	ProductPathKey                        = "product_path"
	ProductPathValue                      = "$BITRISE_XCTESTRUN_FILE_PATH"
	PipelineIntermediateFilesKey          = "pipeline_intermediate_files"
	PipelineIntermediateFilesValue        = "BITRISE_TEST_SHARDS_PATH\nBITRISE_TEST_BUNDLE_PATH"
	OnlyTestingKey                        = "only_testing"
	OnlyTestingValue                      = "$BITRISE_TEST_SHARDS_PATH/$BITRISE_IO_PARALLEL_INDEX"
	XctestrunKey                          = "xctestrun"
	XctestrunValue                        = "$BITRISE_TEST_BUNDLE_PATH/all_tests.xctestrun"

	// test pipeline
	testPipelineID = "run_tests"

	// test workflow
	primaryWorkflowID = "primary"

	testWorkflowID          = "run_tests"
	testWorkflowSummary     = "Run your Xcode tests and get the test report."
	testWorkflowDescription = "The workflow will first clone your Git repository, cache and install your project's dependencies if any, run your Xcode tests and save the test results."

	buildWorkflowID          = "build"
	buildWorkflowSummary     = "Build your Xcode project."
	buildWorkflowDescription = "The workflow will first clone your Git repository, cache and install your project's dependencies if any and build your project."

	buildForTestingWorkflowID     = "build_for_testing"
	testWithoutBuildingWorkflowID = "test_without_building"

	// deploy workflow
	deployWorkflowID = "deploy"

	archiveAndExportWorkflowID = "archive_and_export_app"

	archiveAndExportWorkflowWithTestsSummary     = "Run your Xcode tests and create an IPA file to install your app on a device or share it with your team."
	archiveAndExportWorkflowWithTestsDescription = "The workflow will first clone your Git repository, cache and install your project's dependencies if any, run your Xcode tests, export an IPA file from the project and save it."

	archiveAndExportWorkflowWithoutTestsSummary     = "Create an IPA file to install your app on a device or share it with your team."
	archiveAndExportWorkflowWithoutTestsDescription = "The workflow will first clone your Git repository, cache and install your project's dependencies if any, export an IPA file from the project and save it."
)

type workflowSetupParams struct {
	projectType        XcodeProjectType
	configBuilder      *models.ConfigBuilderModel
	sshKeyActivation   models.SSHKeyActivation
	hasTests           bool
	hasAppClip         bool
	hasPodfile         bool
	hasSPMDependencies bool
	carthageCommand    string
	exportMethod       string
}

func createVerificationWorkflow(params workflowSetupParams) {
	id, summary, description := verificationWorkflowIDSummaryAndDescription(params.projectType, params.hasTests)

	addSharedSetupSteps(models.WorkflowID(id), params, false, true)

	if params.hasTests {
		addTestStep(models.WorkflowID(id), params.configBuilder, params.projectType)
	} else {
		addBuildStep(models.WorkflowID(id), params.configBuilder, params.projectType)
	}

	addSharedTeardownSteps(models.WorkflowID(id), params, true)
	addSummary(models.WorkflowID(id), params.configBuilder, summary)
	addDescription(models.WorkflowID(id), params.configBuilder, description)
}

func createDeployWorkflow(params workflowSetupParams) {
	id, summary, description := deployWorkflowIDSummaryAndDescription(params.projectType, params.hasTests)

	includeCertificateAndProfileInstallStep := params.projectType == XcodeProjectTypeMacOS
	addSharedSetupSteps(models.WorkflowID(id), params, includeCertificateAndProfileInstallStep, false)

	if params.hasTests {
		addTestStep(models.WorkflowID(id), params.configBuilder, params.projectType)
	}

	addArchiveStep(models.WorkflowID(id), params.configBuilder, params.projectType, params.hasAppClip, params.exportMethod)
	addSharedTeardownSteps(models.WorkflowID(id), params, false) // No cache in deploy workflows
	addSummary(models.WorkflowID(id), params.configBuilder, summary)
	addDescription(models.WorkflowID(id), params.configBuilder, description)
}

func createBuildForTestingWorkflow(params workflowSetupParams) {
	if (params.projectType != XcodeProjectTypeIOS) || !params.hasTests {
		return
	}

	workflow := models.WorkflowID(buildForTestingWorkflowID)

	addSharedSetupSteps(workflow, params, false, true)
	params.configBuilder.AppendStepListItemsTo(workflow, steps.XcodeBuildForTestStepListItem(genericXcodeBuildForTestStepInputModels()...))
	addCacheTeardownStep(workflow, params)

	params.configBuilder.AppendStepListItemsTo(workflow,
		steps.XcodeTestShardCalculationStepListItem(xcodeTestShardCalculationStepInputModels()...),
		steps.DeployToBitriseIoStepListItem(buildForTestingDeployToBitriseIoStepInputModels()...),
	)
}

func createTestWithoutBuildingWorkflow(params workflowSetupParams) {
	if (params.projectType != XcodeProjectTypeIOS) || !params.hasTests {
		return
	}

	workflow := models.WorkflowID(testWithoutBuildingWorkflowID)

	params.configBuilder.AppendStepListItemsTo(
		workflow,
		steps.PullIntermediateFilesStepListItem(),
		steps.XcodeTestWithoutBuildingStepListItem(xcodeTestWithoutBuildingStepInputModels()...),
	)
}

func createRunTestsParallelPipeline(params workflowSetupParams) {
	if (params.projectType != XcodeProjectTypeIOS) || !params.hasTests {
		return
	}

	params.configBuilder.SetGraphPipelineWorkflowTo(testPipelineID, buildForTestingWorkflowID, bitriseModels.GraphPipelineWorkflowModel{})
	params.configBuilder.SetGraphPipelineWorkflowTo(testPipelineID, testWithoutBuildingWorkflowID, bitriseModels.GraphPipelineWorkflowModel{
		Parallel:  ShardCountValue,
		DependsOn: []string{buildForTestingWorkflowID},
	})
}

func verificationWorkflowIDSummaryAndDescription(projectType XcodeProjectType, hasTests bool) (string, string, string) {
	var id string
	var summary string
	var description string

	if projectType == XcodeProjectTypeMacOS {
		id = primaryWorkflowID
		summary = ""
		description = ""
	} else {
		if hasTests {
			id = testWorkflowID
			summary = testWorkflowSummary
			description = testWorkflowDescription
		} else {
			id = buildWorkflowID
			summary = buildWorkflowSummary
			description = buildWorkflowDescription
		}
	}

	return id, summary, description
}

func deployWorkflowIDSummaryAndDescription(projectType XcodeProjectType, hasTests bool) (string, string, string) {
	var id string
	var summary string
	var description string

	if projectType == XcodeProjectTypeMacOS {
		id = deployWorkflowID
		summary = ""
		description = ""
	} else {
		id = archiveAndExportWorkflowID
		if hasTests {
			summary = archiveAndExportWorkflowWithTestsSummary
			description = archiveAndExportWorkflowWithTestsDescription
		} else {
			summary = archiveAndExportWorkflowWithoutTestsSummary
			description = archiveAndExportWorkflowWithoutTestsDescription
		}
	}

	return id, summary, description
}

// Add steps

func addTestStep(workflow models.WorkflowID, configBuilder *models.ConfigBuilderModel, projectType XcodeProjectType) {
	switch projectType {
	case XcodeProjectTypeIOS:
		configBuilder.AppendStepListItemsTo(workflow, steps.XcodeTestStepListItem(xcodeTestStepInputModels()...))
	case XcodeProjectTypeMacOS:
		configBuilder.AppendStepListItemsTo(workflow, steps.XcodeTestMacStepListItem(baseXcodeStepInputModels()...))
	}
}

func addBuildStep(workflow models.WorkflowID, configBuilder *models.ConfigBuilderModel, projectType XcodeProjectType) {
	if projectType != XcodeProjectTypeIOS {
		return
	}

	configBuilder.AppendStepListItemsTo(workflow, steps.XcodeBuildForTestStepListItem(xcodeBuildForTestStepInputModels()...))
}

func addArchiveStep(workflow models.WorkflowID, configBuilder *models.ConfigBuilderModel, projectType XcodeProjectType, hasAppClip bool, exportMethod string) {
	inputModels := xcodeArchiveStepInputModels(projectType)

	switch projectType {
	case XcodeProjectTypeIOS:
		configBuilder.AppendStepListItemsTo(workflow, steps.XcodeArchiveStepListItem(inputModels...))

		if shouldAppendExportAppClipStep(hasAppClip, exportMethod) {
			appendExportAppClipStep(configBuilder, workflow)
		}
	case XcodeProjectTypeMacOS:
		configBuilder.AppendStepListItemsTo(workflow, steps.XcodeArchiveMacStepListItem(inputModels...))
	}
}

func addSharedSetupSteps(workflow models.WorkflowID, params workflowSetupParams, includeCertificateAndProfileInstallStep, includeCache bool) {
	params.configBuilder.AppendStepListItemsTo(workflow, steps.DefaultPrepareStepList(steps.PrepareListParams{
		SSHKeyActivation: params.sshKeyActivation,
	})...)

	if includeCache {
		if params.hasPodfile {
			params.configBuilder.AppendStepListItemsTo(workflow, steps.RestoreCocoapodsCache())
		}
		if params.carthageCommand != "" {
			params.configBuilder.AppendStepListItemsTo(workflow, steps.RestoreCarthageCache())
		}
		if params.hasSPMDependencies {
			params.configBuilder.AppendStepListItemsTo(workflow, steps.RestoreSPMCache())
		}
	}

	if includeCertificateAndProfileInstallStep {
		params.configBuilder.AppendStepListItemsTo(workflow, steps.CertificateAndProfileInstallerStepListItem())
	}

	if params.hasPodfile {
		params.configBuilder.AppendStepListItemsTo(workflow, steps.CocoapodsInstallStepListItem())
	}

	if params.carthageCommand != "" {
		params.configBuilder.AppendStepListItemsTo(workflow, steps.CarthageStepListItem(
			envmanModels.EnvironmentItemModel{CarthageCommandInputKey: params.carthageCommand},
		))
	}
}

func addCacheTeardownStep(workflow models.WorkflowID, params workflowSetupParams) {
	if params.hasPodfile {
		params.configBuilder.AppendStepListItemsTo(workflow, steps.SaveCocoapodsCache())
	}
	if params.carthageCommand != "" {
		params.configBuilder.AppendStepListItemsTo(workflow, steps.SaveCarthageCache())
	}
	if params.hasSPMDependencies {
		params.configBuilder.AppendStepListItemsTo(workflow, steps.SaveSPMCache())
	}
}

func addSharedTeardownSteps(workflow models.WorkflowID, params workflowSetupParams, includeCache bool) {
	if includeCache {
		addCacheTeardownStep(workflow, params)
	}

	params.configBuilder.AppendStepListItemsTo(workflow, steps.DefaultDeployStepList()...)
}

func addDescription(workflow models.WorkflowID, configBuilder *models.ConfigBuilderModel, description string) {
	configBuilder.SetWorkflowDescriptionTo(workflow, description)
}

func addSummary(workflow models.WorkflowID, configBuilder *models.ConfigBuilderModel, summary string) {
	configBuilder.SetWorkflowSummaryTo(workflow, summary)
}

// Helpers

func baseXcodeStepInputModels() []envmanModels.EnvironmentItemModel {
	return []envmanModels.EnvironmentItemModel{
		{ProjectPathInputKey: "$" + ProjectPathInputEnvKey},
		{SchemeInputKey: "$" + SchemeInputEnvKey},
	}
}

func xcodeTestStepInputModels() []envmanModels.EnvironmentItemModel {
	inputModels := []envmanModels.EnvironmentItemModel{
		{TestRepetitionModeKey: TestRepetitionModeRetryOnFailureValue},
		{CacheLevelKey: CacheLevelNone},
	}

	return append(baseXcodeStepInputModels(), inputModels...)
}

func xcodeBuildForTestStepInputModels() []envmanModels.EnvironmentItemModel {
	inputModels := []envmanModels.EnvironmentItemModel{
		{BuildForTestDestinationKey: BuildForTestDestinationValue},
		{CacheLevelKey: CacheLevelNone},
	}

	return append(baseXcodeStepInputModels(), inputModels...)
}

func genericXcodeBuildForTestStepInputModels() []envmanModels.EnvironmentItemModel {
	inputModels := []envmanModels.EnvironmentItemModel{
		{BuildForTestDestinationKey: GenericBuildForTestDestinationValue},
		{CacheLevelKey: CacheLevelNone},
	}

	return append(baseXcodeStepInputModels(), inputModels...)
}

func xcodeArchiveStepInputModels(projectType XcodeProjectType) []envmanModels.EnvironmentItemModel {
	var inputModels []envmanModels.EnvironmentItemModel

	if projectType == XcodeProjectTypeIOS {
		inputModels = append(inputModels, []envmanModels.EnvironmentItemModel{
			{DistributionMethodInputKey: "$" + DistributionMethodEnvKey},
			{AutomaticCodeSigningKey: AutomaticCodeSigningValue},
			{CacheLevelKey: CacheLevelNone},
		}...)
	} else {
		inputModels = []envmanModels.EnvironmentItemModel{
			{ExportMethodInputKey: "$" + ExportMethodEnvKey},
		}
	}

	return append(baseXcodeStepInputModels(), inputModels...)
}

func xcodeTestShardCalculationStepInputModels() []envmanModels.EnvironmentItemModel {
	return []envmanModels.EnvironmentItemModel{
		{ShardCountKey: ShardCountValue},
		{ProductPathKey: ProductPathValue},
	}
}

func buildForTestingDeployToBitriseIoStepInputModels() []envmanModels.EnvironmentItemModel {
	return []envmanModels.EnvironmentItemModel{
		{PipelineIntermediateFilesKey: PipelineIntermediateFilesValue},
	}
}

func xcodeTestWithoutBuildingStepInputModels() []envmanModels.EnvironmentItemModel {
	return []envmanModels.EnvironmentItemModel{
		{OnlyTestingKey: OnlyTestingValue},
		{XctestrunKey: XctestrunValue},
	}
}
