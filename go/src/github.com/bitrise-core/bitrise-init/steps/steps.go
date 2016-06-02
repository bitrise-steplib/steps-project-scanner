package steps

import (
	bitrise "github.com/bitrise-io/bitrise/models"
	envman "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/pointers"
	stepman "github.com/bitrise-io/stepman/models"
)

const (
	// Common Step IDs
	activateSSHKeyID                 = "activate-ssh-key"
	gitCloneID                       = "git-clone"
	certificateAndProfileInstallerID = "certificate-and-profile-installer"
	deployToBitriseIoID              = "deploy-to-bitrise-io"

	// Android Step IDs
	gradleRunnerID = "gradle-runner"

	// Fastlane Step IDs
	fastlaneID = "fastlane"

	// iOS Step IDs
	cocoapodsInstallID = "cocoapods-install"
	xcodeArchiveID     = "xcode-archive"
	xcodeTestID        = "xcode-test"

	// Xamarin Step IDs
	xamarinUserManagementID    = "xamarin-user-management"
	nugetRestoreID             = "nuget-restore"
	xamarinComponentsRestoreID = "xamarin-components-restore"
	xamarinBuilderID           = "xamarin-builder"
)

const (
	// Common Step Versions
	activateSSHKeyVersion                 = "3.1.0"
	gitCloneVersion                       = "3.2.0"
	certificateAndProfileInstallerVersion = "1.5.0"
	deployToBitriseIoVersion              = "1.2.3"

	// Android Step Versions
	gradleRunnerVersion = "1.3.1"

	// Fatslane Step Versions
	fastlaneVersion = "2.2.0"

	// iOS Step Versions
	cocoapodsInstallVersion = "1.4.0"
	xcodeArchiveVersion     = "1.7.3"
	xcodeTestVersion        = "1.13.7"

	// Xamarin Step Versions
	xamarinUserManagementVersion    = "1.0.2"
	nugetRestoreVersion             = "0.9.1"
	xamarinComponentsRestoreVersion = "0.9.0"
	xamarinBuilderVersion           = "1.3.0"
)

func setpIDComposite(ID, version string) string {
	return ID + "@" + version
}

func stepListItem(stepIDComposite, runIf string, inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	step := stepman.StepModel{}
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
	stepIDComposite := setpIDComposite(activateSSHKeyID, activateSSHKeyVersion)
	runIf := `{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}`
	return stepListItem(stepIDComposite, runIf, nil)
}

// GitCloneStepListItem ...
func GitCloneStepListItem() bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(gitCloneID, gitCloneVersion)
	return stepListItem(stepIDComposite, "", nil)
}

// CertificateAndProfileInstallerStepListItem ...
func CertificateAndProfileInstallerStepListItem() bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(certificateAndProfileInstallerID, certificateAndProfileInstallerVersion)
	return stepListItem(stepIDComposite, "", nil)
}

// DeployToBitriseIoStepListItem ...
func DeployToBitriseIoStepListItem() bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(deployToBitriseIoID, deployToBitriseIoVersion)
	return stepListItem(stepIDComposite, "", nil)
}

//------------------------
// Android Step List Items
//------------------------

// GradleRunnerStepListItem ...
func GradleRunnerStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(gradleRunnerID, gradleRunnerVersion)
	return stepListItem(stepIDComposite, "", inputs)
}

//------------------------
// Fastlane Step List Items
//------------------------

// FastlaneStepListItem ...
func FastlaneStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(fastlaneID, fastlaneVersion)
	return stepListItem(stepIDComposite, "", inputs)
}

//------------------------
// iOS Step List Items
//------------------------

// CocoapodsInstallStepListItem ...
func CocoapodsInstallStepListItem() bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(cocoapodsInstallID, cocoapodsInstallVersion)
	return stepListItem(stepIDComposite, "", nil)
}

// XcodeArchiveStepListItem ...
func XcodeArchiveStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(xcodeArchiveID, xcodeArchiveVersion)
	return stepListItem(stepIDComposite, "", inputs)
}

// XcodeTestStepListItem ...
func XcodeTestStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(xcodeTestID, xcodeTestVersion)
	return stepListItem(stepIDComposite, "", inputs)
}

//------------------------
// Xamarin Step List Items
//------------------------

// XamarinUserManagementStepListItem ...
func XamarinUserManagementStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(xamarinUserManagementID, xamarinUserManagementVersion)
	runIf := ".IsCI"
	return stepListItem(stepIDComposite, runIf, inputs)
}

// NugetRestoreStepListItem ...
func NugetRestoreStepListItem() bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(nugetRestoreID, nugetRestoreVersion)
	return stepListItem(stepIDComposite, "", nil)
}

// XamarinComponentsRestoreStepListItem ...
func XamarinComponentsRestoreStepListItem() bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(xamarinComponentsRestoreID, xamarinComponentsRestoreVersion)
	return stepListItem(stepIDComposite, "", nil)
}

// XamarinBuilderStepListItem ...
func XamarinBuilderStepListItem(inputs []envman.EnvironmentItemModel) bitrise.StepListItemModel {
	stepIDComposite := setpIDComposite(xamarinBuilderID, xamarinBuilderVersion)
	return stepListItem(stepIDComposite, "", inputs)
}
