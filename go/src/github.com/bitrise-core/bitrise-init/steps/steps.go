package steps

import (
	bitrise "github.com/bitrise-io/bitrise/models"
	envman "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pointers"
	stepman "github.com/bitrise-io/stepman/models"
)

const (
	// Common Step IDs
	activateSSHKeyID      = "activate-ssh-key"
	activateSSHKeyVersion = "3.1.1"

	gitCloneID      = "git-clone"
	gitCloneVersion = "3.3.4"

	certificateAndProfileInstallerID      = "certificate-and-profile-installer"
	certificateAndProfileInstallerVersion = "1.6.0"

	deployToBitriseIoID      = "deploy-to-bitrise-io"
	deployToBitriseIoVersion = "1.2.5"

	scriptID      = "script"
	scriptVersion = "1.1.3"

	// Android Step IDs
	gradleRunnerID      = "gradle-runner"
	gradleRunnerVersion = "1.4.1"

	// Fastlane Step IDs
	fastlaneID      = "fastlane"
	fastlaneVersion = "2.2.0"

	// iOS Step IDs
	cocoapodsInstallID      = "cocoapods-install"
	cocoapodsInstallVersion = "1.5.7"

	recreateUserSchemesID      = "recreate-user-schemes"
	recreateUserSchemesVersion = "0.9.4"

	xcodeArchiveID      = "xcode-archive"
	xcodeArchiveVersion = "1.9.1"

	xcodeTestID      = "xcode-test"
	xcodeTestVersion = "1.14.0"

	// Xamarin Step IDs
	xamarinUserManagementID      = "xamarin-user-management"
	xamarinUserManagementVersion = "1.0.2"

	nugetRestoreID      = "nuget-restore"
	nugetRestoreVersion = "0.9.3"

	xamarinComponentsRestoreID      = "xamarin-components-restore"
	xamarinComponentsRestoreVersion = "0.9.0"

	xamarinBuilderID      = "xamarin-builder"
	xamarinBuilderVersion = "1.4.0"
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
	stepIDComposite := stepIDComposite(activateSSHKeyID, activateSSHKeyVersion)
	runIf := `{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}`
	return stepListItem(stepIDComposite, "", runIf, nil)
}

// GitCloneStepListItem ...
func GitCloneStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(gitCloneID, gitCloneVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// CertificateAndProfileInstallerStepListItem ...
func CertificateAndProfileInstallerStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(certificateAndProfileInstallerID, certificateAndProfileInstallerVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// DeployToBitriseIoStepListItem ...
func DeployToBitriseIoStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(deployToBitriseIoID, deployToBitriseIoVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// ScriptSteplistItem ...
func ScriptSteplistItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(scriptID, scriptVersion)
	return stepListItem(stepIDComposite, "Do anything with Script step", "", nil)
}

//------------------------
// Android Step List Items
//------------------------

// GradleRunnerStepListItem ...
func GradleRunnerStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(gradleRunnerID, gradleRunnerVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

//------------------------
// Fastlane Step List Items
//------------------------

// FastlaneStepListItem ...
func FastlaneStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(fastlaneID, fastlaneVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

//------------------------
// iOS Step List Items
//------------------------

// CocoapodsInstallStepListItem ...
func CocoapodsInstallStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(cocoapodsInstallID, cocoapodsInstallVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// RecreateUserSchemesStepListItem ...
func RecreateUserSchemesStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(recreateUserSchemesID, recreateUserSchemesVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

// XcodeArchiveStepListItem ...
func XcodeArchiveStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(xcodeArchiveID, xcodeArchiveVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

// XcodeTestStepListItem ...
func XcodeTestStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(xcodeTestID, xcodeTestVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}

//------------------------
// Xamarin Step List Items
//------------------------

// XamarinUserManagementStepListItem ...
func XamarinUserManagementStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(xamarinUserManagementID, xamarinUserManagementVersion)
	runIf := ".IsCI"
	return stepListItem(stepIDComposite, "", runIf, inputs)
}

// NugetRestoreStepListItem ...
func NugetRestoreStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(nugetRestoreID, nugetRestoreVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// XamarinComponentsRestoreStepListItem ...
func XamarinComponentsRestoreStepListItem() bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(xamarinComponentsRestoreID, xamarinComponentsRestoreVersion)
	return stepListItem(stepIDComposite, "", "", nil)
}

// XamarinBuilderStepListItem ...
func XamarinBuilderStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := stepIDComposite(xamarinBuilderID, xamarinBuilderVersion)
	return stepListItem(stepIDComposite, "", "", inputs)
}
