package steps

import (
	bitriseModels "github.com/bitrise-io/bitrise/models"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pointers"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

func stepIDComposite(ID, version string) string {
	if version != "" {
		return ID + "@" + version
	}
	return ID
}

func stepListItem(stepIDComposite, title, runIf string, inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	step := stepmanModels.StepModel{}
	if title != "" {
		step.Title = pointers.NewStringPtr(title)
	}
	if runIf != "" {
		step.RunIf = pointers.NewStringPtr(runIf)
	}
	if len(inputs) > 0 {
		step.Inputs = inputs
	}

	return bitriseModels.StepListItemModel{
		stepIDComposite: step,
	}
}

// DefaultPrepareStepList ...
func DefaultPrepareStepList() []bitriseModels.StepListItemModel {
	return []bitriseModels.StepListItemModel{
		ActivateSSHKeyStepListItem(),
		GitCloneStepListItem(),
		ScriptSteplistItem(ScriptDefaultTitle),
	}
}

// DefaultDeployStepList ...
func DefaultDeployStepList() []bitriseModels.StepListItemModel {
	return []bitriseModels.StepListItemModel{
		DeployToBitriseIoStepListItem(),
	}
}

// ActivateSSHKeyStepListItem ...
func ActivateSSHKeyStepListItem() bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(ActivateSSHKeyID, ActivateSSHKeyVersion)
	runIf := `{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}`
	return stepListItem(stepIDComposite, "", runIf)
}

// ChangeWorkDirStepListItem ...
func ChangeWorkDirStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(ChangeWorkDirID, ChangeWorkDirVersion)
	inputs = append(inputs, envmanModels.EnvironmentItemModel{"is_create_path": "false"})
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// GitCloneStepListItem ...
func GitCloneStepListItem() bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(GitCloneID, GitCloneVersion)
	return stepListItem(stepIDComposite, "", "")
}

// CertificateAndProfileInstallerStepListItem ...
func CertificateAndProfileInstallerStepListItem() bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(CertificateAndProfileInstallerID, CertificateAndProfileInstallerVersion)
	return stepListItem(stepIDComposite, "", "")
}

// DeployToBitriseIoStepListItem ...
func DeployToBitriseIoStepListItem() bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(DeployToBitriseIoID, DeployToBitriseIoVersion)
	return stepListItem(stepIDComposite, "", "")
}

// ScriptSteplistItem ...
func ScriptSteplistItem(title string, inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(ScriptID, ScriptVersion)
	return stepListItem(stepIDComposite, title, "", inputs...)
}

// InstallMissingAndroidToolsStepListItem ....
func InstallMissingAndroidToolsStepListItem() bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(InstallMissingAndroidToolsID, InstallMissingAndroidToolsVersion)
	return stepListItem(stepIDComposite, "", "")
}

// GradleRunnerStepListItem ...
func GradleRunnerStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(GradleRunnerID, GradleRunnerVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// FastlaneStepListItem ...
func FastlaneStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(FastlaneID, FastlaneVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// CocoapodsInstallStepListItem ...
func CocoapodsInstallStepListItem() bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(CocoapodsInstallID, CocoapodsInstallVersion)
	return stepListItem(stepIDComposite, "", "")
}

// CarthageStepListItem ...
func CarthageStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(CarthageID, CarthageVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// RecreateUserSchemesStepListItem ...
func RecreateUserSchemesStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(RecreateUserSchemesID, RecreateUserSchemesVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// XcodeArchiveStepListItem ...
func XcodeArchiveStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(XcodeArchiveID, XcodeArchiveVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// XcodeTestStepListItem ...
func XcodeTestStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(XcodeTestID, XcodeTestVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// XamarinUserManagementStepListItem ...
func XamarinUserManagementStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(XamarinUserManagementID, XamarinUserManagementVersion)
	runIf := ".IsCI"
	return stepListItem(stepIDComposite, "", runIf, inputs...)
}

// NugetRestoreStepListItem ...
func NugetRestoreStepListItem() bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(NugetRestoreID, NugetRestoreVersion)
	return stepListItem(stepIDComposite, "", "")
}

// XamarinComponentsRestoreStepListItem ...
func XamarinComponentsRestoreStepListItem() bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(XamarinComponentsRestoreID, XamarinComponentsRestoreVersion)
	return stepListItem(stepIDComposite, "", "")
}

// XamarinArchiveStepListItem ...
func XamarinArchiveStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(XamarinArchiveID, XamarinArchiveVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// XcodeArchiveMacStepListItem ...
func XcodeArchiveMacStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(XcodeArchiveMacID, XcodeArchiveMacVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// XcodeTestMacStepListItem ...
func XcodeTestMacStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(XcodeTestMacID, XcodeTestMacVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// GenerateGradleWrapperStepListItem ...
func GenerateGradleWrapperStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(GenerateGradleWrapperID, GenerateGradleWrapperVersion)
	return stepListItem(stepIDComposite, "", "")
}

// CordovaArchiveStepListItem ...
func CordovaArchiveStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(CordovaArchiveID, CordovaArchiveVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// GenerateCordovaBuildConfigStepListItem ...
func GenerateCordovaBuildConfigStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(GenerateCordovaBuildConfigID, GenerateCordovaBuildConfigVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// JasmineTestRunnerStepListItem ...
func JasmineTestRunnerStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(JasmineTestRunnerID, JasmineTestRunnerVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}

// KarmaJasmineTestRunnerStepListItem ...
func KarmaJasmineTestRunnerStepListItem(inputs ...envmanModels.EnvironmentItemModel) bitriseModels.StepListItemModel {
	stepIDComposite := stepIDComposite(KarmaJasmineTestRunnerID, KarmaJasmineTestRunnerVersion)
	return stepListItem(stepIDComposite, "", "", inputs...)
}
