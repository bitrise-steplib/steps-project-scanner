package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"os"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestManualConfig(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__manual-config__")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	}()

	t.Log("manual-config")
	{
		manualConfigDir := filepath.Join(tmpDir, "manual-config")
		require.NoError(t, os.MkdirAll(manualConfigDir, 0777))

		cmd := command.New(binPath(), "--ci", "manual-config", "--output-dir", manualConfigDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(manualConfigDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(customConfigResultYML), strings.TrimSpace(result))
	}
}

var customConfigVersions = []interface{}{
	// android
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.InstallMissingAndroidToolsVersion,
	steps.GradleRunnerVersion,
	steps.DeployToBitriseIoVersion,

	// fastlane
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.FastlaneVersion,
	steps.DeployToBitriseIoVersion,

	// ios
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.CocoapodsInstallVersion,
	steps.RecreateUserSchemesVersion,
	steps.XcodeTestVersion,
	steps.XcodeArchiveVersion,
	steps.DeployToBitriseIoVersion,

	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.CocoapodsInstallVersion,
	steps.RecreateUserSchemesVersion,
	steps.XcodeTestVersion,
	steps.DeployToBitriseIoVersion,

	// macos
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.CocoapodsInstallVersion,
	steps.RecreateUserSchemesVersion,
	steps.XcodeTestMacVersion,
	steps.XcodeArchiveMacVersion,
	steps.DeployToBitriseIoVersion,

	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.CocoapodsInstallVersion,
	steps.RecreateUserSchemesVersion,
	steps.XcodeTestMacVersion,
	steps.DeployToBitriseIoVersion,

	// other
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,

	// xamarin
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.XamarinUserManagementVersion,
	steps.NugetRestoreVersion,
	steps.XamarinComponentsRestoreVersion,
	steps.XamarinArchiveVersion,
	steps.DeployToBitriseIoVersion,
}

var customConfigResultYML = fmt.Sprintf(`options:
  android:
    title: Path to the gradle file to use
    env_key: GRADLE_BUILD_FILE_PATH
    value_map:
      _:
        title: Gradle task to run
        env_key: GRADLE_TASK
        value_map:
          _:
            title: Gradlew file path
            env_key: GRADLEW_PATH
            value_map:
              _:
                config: default-android-config
  fastlane:
    title: Working directory
    env_key: FASTLANE_WORK_DIR
    value_map:
      _:
        title: Fastlane lane
        env_key: FASTLANE_LANE
        value_map:
          _:
            config: default-fastlane-config
  ios:
    title: Project (or Workspace) path
    env_key: BITRISE_PROJECT_PATH
    value_map:
      _:
        title: Scheme name
        env_key: BITRISE_SCHEME
        value_map:
          _:
            config: default-ios-config
  macos:
    title: Project (or Workspace) path
    env_key: BITRISE_PROJECT_PATH
    value_map:
      _:
        title: Scheme name
        env_key: BITRISE_SCHEME
        value_map:
          _:
            config: default-macos-config
  xamarin:
    title: Path to the Xamarin Solution file
    env_key: BITRISE_PROJECT_PATH
    value_map:
      _:
        title: Xamarin solution configuration
        env_key: BITRISE_XAMARIN_CONFIGURATION
        value_map:
          _:
            title: Xamarin solution platform
            env_key: BITRISE_XAMARIN_PLATFORM
            value_map:
              _:
                config: default-xamarin-config
configs:
  android:
    default-android-config: |
      format_version: %s
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      trigger_map:
      - push_branch: '*'
        workflow: primary
      - pull_request_source_branch: '*'
        workflow: primary
      workflows:
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - install-missing-android-tools@%s: {}
          - gradle-runner@%s:
              inputs:
              - gradle_file: $GRADLE_BUILD_FILE_PATH
              - gradle_task: $GRADLE_TASK
              - gradlew_path: $GRADLEW_PATH
          - deploy-to-bitrise-io@%s: {}
  fastlane:
    default-fastlane-config: |
      format_version: %s
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      app:
        envs:
        - FASTLANE_XCODE_LIST_TIMEOUT: "120"
      trigger_map:
      - push_branch: '*'
        workflow: primary
      - pull_request_source_branch: '*'
        workflow: primary
      workflows:
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - certificate-and-profile-installer@%s: {}
          - fastlane@%s:
              inputs:
              - lane: $FASTLANE_LANE
              - work_dir: $FASTLANE_WORK_DIR
          - deploy-to-bitrise-io@%s: {}
  ios:
    default-ios-config: |
      format_version: %s
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      trigger_map:
      - push_branch: '*'
        workflow: primary
      - pull_request_source_branch: '*'
        workflow: primary
      workflows:
        deploy:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - certificate-and-profile-installer@%s: {}
          - cocoapods-install@%s: {}
          - recreate-user-schemes@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
          - xcode-test@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - xcode-archive@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - deploy-to-bitrise-io@%s: {}
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - certificate-and-profile-installer@%s: {}
          - cocoapods-install@%s: {}
          - recreate-user-schemes@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
          - xcode-test@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - deploy-to-bitrise-io@%s: {}
  macos:
    default-macos-config: |
      format_version: %s
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      app: {}
      trigger_map:
      - push_branch: '*'
        workflow: primary
      - pull_request_source_branch: '*'
        workflow: primary
      workflows:
        deploy:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - certificate-and-profile-installer@%s: {}
          - cocoapods-install@%s: {}
          - recreate-user-schemes@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
          - xcode-test-mac@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - xcode-archive-mac@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - deploy-to-bitrise-io@%s: {}
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - certificate-and-profile-installer@%s: {}
          - cocoapods-install@%s: {}
          - recreate-user-schemes@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
          - xcode-test-mac@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - deploy-to-bitrise-io@%s: {}
  other:
    other-config: |
      format_version: %s
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      trigger_map:
      - push_branch: '*'
        workflow: primary
      - pull_request_source_branch: '*'
        workflow: primary
      workflows:
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
  xamarin:
    default-xamarin-config: |
      format_version: %s
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      trigger_map:
      - push_branch: '*'
        workflow: primary
      - pull_request_source_branch: '*'
        workflow: primary
      workflows:
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - certificate-and-profile-installer@%s: {}
          - xamarin-user-management@%s:
              run_if: .IsCI
          - nuget-restore@%s: {}
          - xamarin-components-restore@%s: {}
          - xamarin-archive@%s:
              inputs:
              - xamarin_solution: $BITRISE_PROJECT_PATH
              - xamarin_configuration: $BITRISE_XAMARIN_CONFIGURATION
              - xamarin_platform: $BITRISE_XAMARIN_PLATFORM
          - deploy-to-bitrise-io@%s: {}
`, customConfigVersions...)
