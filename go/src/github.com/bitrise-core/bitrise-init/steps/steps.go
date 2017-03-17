package steps

import (
	bitrise "github.com/bitrise-io/bitrise/models"
	envman "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pointers"
	stepman "github.com/bitrise-io/stepman/models"
)

const (
	// Common Steps

	// ActivateSSHKeyID ...
	ActivateSSHKeyID = "activate-ssh-key"
	// ActivateSSHKeyVersion ...
	ActivateSSHKeyVersion = "3.1.1"

	// GitCloneID ...
	GitCloneID = "git-clone"
	// GitCloneVersion ...
	GitCloneVersion = "3.4.2"

	// CertificateAndProfileInstallerID ...
	CertificateAndProfileInstallerID = "certificate-and-profile-installer"
	// CertificateAndProfileInstallerVersion ...
	CertificateAndProfileInstallerVersion = "1.8.4"

	// DeployToBitriseIoID ...
	DeployToBitriseIoID = "deploy-to-bitrise-io"
	// DeployToBitriseIoVersion ...
	DeployToBitriseIoVersion = "1.2.9"

	// ScriptID ...
	ScriptID = "script"
	// ScriptVersion ...
	ScriptVersion = "1.1.3"
	// ScriptDefaultTitle ...
	ScriptDefaultTitle = "Do anything with Script step"

	// Android Steps

	// InstallMissingAndroidToolsID ...
	InstallMissingAndroidToolsID = "install-missing-android-tools"
	// InstallMissingAndroidToolsVersion ...
	InstallMissingAndroidToolsVersion = "0.9.2"

	// GradleRunnerID ...
	GradleRunnerID = "gradle-runner"
	// GradleRunnerVersion ...
	GradleRunnerVersion = "1.5.4"

	// Fastlane Steps

	// FastlaneID ...
	FastlaneID = "fastlane"
	// FastlaneVersion ...
	FastlaneVersion = "2.3.7"

	// iOS Steps

	// CocoapodsInstallID ...
	CocoapodsInstallID = "cocoapods-install"
	// CocoapodsInstallVersion ...
	CocoapodsInstallVersion = "1.6.1"

	// CarthageID ...
	CarthageID = "carthage"
	// CarthageVersion ...
	CarthageVersion = "3.0.6"

	// RecreateUserSchemesID ...
	RecreateUserSchemesID = "recreate-user-schemes"
	// RecreateUserSchemesVersion ...
	RecreateUserSchemesVersion = "0.9.5"

	// XcodeArchiveID ...
	XcodeArchiveID = "xcode-archive"
	// XcodeArchiveVersion ...
	XcodeArchiveVersion = "2.0.5"

	// XcodeTestID ...
	XcodeTestID = "xcode-test"
	// XcodeTestVersion ...
	XcodeTestVersion = "1.18.1"

	// Xamarin Steps

	// XamarinUserManagementID ...
	XamarinUserManagementID = "xamarin-user-management"
	// XamarinUserManagementVersion ...
	XamarinUserManagementVersion = "1.0.3"

	// NugetRestoreID ...
	NugetRestoreID = "nuget-restore"
	// NugetRestoreVersion ...
	NugetRestoreVersion = "1.0.3"

	// XamarinComponentsRestoreID ...
	XamarinComponentsRestoreID = "xamarin-components-restore"
	// XamarinComponentsRestoreVersion ...
	XamarinComponentsRestoreVersion = "0.9.0"

	// XamarinArchiveID ...
	XamarinArchiveID = "xamarin-archive"
	// XamarinArchiveVersion ...
	XamarinArchiveVersion = "1.3.2"

	// macOS Setps

	// XcodeArchiveMacID ...
	XcodeArchiveMacID = "xcode-archive-mac"
	// XcodeArchiveMacVersion ...
	XcodeArchiveMacVersion = "1.4.0"

	// XcodeTestMacID ...
	XcodeTestMacID = "xcode-test-mac"
	// XcodeTestMacVersion ...
	XcodeTestMacVersion = "1.1.0"
)

func stepIDComposite(ID, version string) string {
	return ID + "@" + version
}

func stepListItem(stepIDComposite, title, runIf string, inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	step := stepman.StepModel{}
	if title != "" {
		step.Title = pointers.NewStringPtr(title)
	}
	if runIf != "" {
		step.RunIf = pointers.NewStringPtr(runIf)
	}
	if inputs != nil && len(inputs) > 0 {
		step.Inputs = inputs
	}

	return bitrise.StepListItemModel{
		stepIDComposite: step,
	}
}

//------------------------
// Common Step List Items
//------------------------

// ActivateSSHKeyStepListItem ...
func ActivateSSHKeyStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(ActivateSSHKeyID, ActivateSSHKeyVersion)
	runIf := `{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}`
	return stepListItem(stepIDComposite, "", runIf, nil)
}

// GitCloneStepListItem ...
func GitCloneStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(GitCloneID, GitCloneVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// CertificateAndProfileInstallerStepListItem ...
func CertificateAndProfileInstallerStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(CertificateAndProfileInstallerID, CertificateAndProfileInstallerVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// DeployToBitriseIoStepListItem ...
func DeployToBitriseIoStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(DeployToBitriseIoID, DeployToBitriseIoVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// ScriptSteplistItem ...
func ScriptSteplistItem(title string, inputs ...envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(ScriptID, ScriptVersion)
	return stepListItem(stepIDComposite, title, "", inputs)
}

//------------------------
// Android Step List Items
//------------------------

// InstallMissingAndroidToolsStepListItem ....
func InstallMissingAndroidToolsStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(InstallMissingAndroidToolsID, InstallMissingAndroidToolsVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// GradleRunnerStepListItem ...
func GradleRunnerStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(GradleRunnerID, GradleRunnerVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

//------------------------
// Fastlane Step List Items
//------------------------

// FastlaneStepListItem ...
func FastlaneStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(FastlaneID, FastlaneVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

//------------------------
// iOS Step List Items
//------------------------

// CocoapodsInstallStepListItem ...
func CocoapodsInstallStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(CocoapodsInstallID, CocoapodsInstallVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// CarthageStepListItem ...
func CarthageStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(CarthageID, CarthageVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

// RecreateUserSchemesStepListItem ...
func RecreateUserSchemesStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(RecreateUserSchemesID, RecreateUserSchemesVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

// XcodeArchiveStepListItem ...
func XcodeArchiveStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(XcodeArchiveID, XcodeArchiveVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

// XcodeTestStepListItem ...
func XcodeTestStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(XcodeTestID, XcodeTestVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

//------------------------
// Xamarin Step List Items
//------------------------

// XamarinUserManagementStepListItem ...
func XamarinUserManagementStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(XamarinUserManagementID, XamarinUserManagementVersion)
	runIf := ".IsCI"
	return stepListItem(stepIDComposite, "", runIf, inputs)
}

// NugetRestoreStepListItem ...
func NugetRestoreStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(NugetRestoreID, NugetRestoreVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// XamarinComponentsRestoreStepListItem ...
func XamarinComponentsRestoreStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(XamarinComponentsRestoreID, XamarinComponentsRestoreVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// XamarinArchiveStepListItem ...
func XamarinArchiveStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(XamarinArchiveID, XamarinArchiveVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

//------------------------
// macOS Step List Items
//------------------------

// XcodeArchiveMacStepListItem ...
func XcodeArchiveMacStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(XcodeArchiveMacID, XcodeArchiveMacVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

// XcodeTestMacStepListItem ...
func XcodeTestMacStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(XcodeTestMacID, XcodeTestMacVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}
