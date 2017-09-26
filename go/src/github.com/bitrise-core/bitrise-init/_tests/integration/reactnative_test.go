package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/git"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestReactNative(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__reactnative__")
	require.NoError(t, err)

	t.Log("sample-apps-react-native-ios-and-android")
	{
		sampleAppDir := filepath.Join(tmpDir, "sample-apps-react-native-ios-and-android")
		sampleAppURL := "https://github.com/bitrise-samples/sample-apps-react-native-ios-and-android.git"
		require.NoError(t, git.Clone(sampleAppURL, sampleAppDir))

		cmd := command.New(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(sampleAppsReactNativeIosAndAndroidResultYML), strings.TrimSpace(result))
	}

	t.Log("sample-apps-react-native-subdir")
	{
		sampleAppDir := filepath.Join(tmpDir, "sample-apps-react-native-subdir")
		sampleAppURL := "https://github.com/bitrise-samples/sample-apps-react-native-subdir.git"
		require.NoError(t, git.Clone(sampleAppURL, sampleAppDir))

		cmd := command.New(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(sampleAppsReactNativeSubdirResultYML), strings.TrimSpace(result))
	}
}

var sampleAppsReactNativeSubdirVersions = []interface{}{
	models.FormatVersion,

	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.NpmVersion,
	steps.InstallMissingAndroidToolsVersion,
	steps.GradleRunnerVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.XcodeArchiveVersion,
	steps.DeployToBitriseIoVersion,

	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.NpmVersion,
	steps.NpmVersion,
	steps.DeployToBitriseIoVersion,
}

var sampleAppsReactNativeSubdirResultYML = fmt.Sprintf(`options:
  react-native:
    title: Path to the gradle file to use
    env_key: GRADLE_BUILD_FILE_PATH
    value_map:
      project/android/build.gradle:
        title: Gradlew file path
        env_key: GRADLEW_PATH
        value_map:
          project/android/gradlew:
            title: Project (or Workspace) path
            env_key: BITRISE_PROJECT_PATH
            value_map:
              project/ios/SampleAppsReactNativeAndroid.xcodeproj:
                title: Scheme name
                env_key: BITRISE_SCHEME
                value_map:
                  SampleAppsReactNativeAndroid:
                    config: react-native-android-ios-test-config
                  SampleAppsReactNativeAndroid-tvOS:
                    config: react-native-android-ios-test-config
configs:
  react-native:
    react-native-android-ios-test-config: |
      format_version: "%s"
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      project_type: react-native
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
          - npm@%s:
              inputs:
              - workdir: project
              - command: install
          - install-missing-android-tools@%s: {}
          - gradle-runner@%s:
              inputs:
              - gradle_file: $GRADLE_BUILD_FILE_PATH
              - gradle_task: assembleRelease
              - gradlew_path: $GRADLEW_PATH
          - certificate-and-profile-installer@%s: {}
          - xcode-archive@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
              - configuration: Release
          - deploy-to-bitrise-io@%s: {}
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - npm@%s:
              inputs:
              - workdir: project
              - command: install
          - npm@%s:
              inputs:
              - workdir: project
              - command: test
          - deploy-to-bitrise-io@%s: {}
warnings:
  react-native: []
`, sampleAppsReactNativeSubdirVersions...)

var sampleAppsReactNativeIosAndAndroidVersions = []interface{}{
	models.FormatVersion,

	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.NpmVersion,
	steps.InstallMissingAndroidToolsVersion,
	steps.GradleRunnerVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.XcodeArchiveVersion,
	steps.DeployToBitriseIoVersion,

	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.NpmVersion,
	steps.NpmVersion,
	steps.DeployToBitriseIoVersion,
}

var sampleAppsReactNativeIosAndAndroidResultYML = fmt.Sprintf(`options:
  react-native:
    title: Path to the gradle file to use
    env_key: GRADLE_BUILD_FILE_PATH
    value_map:
      android/build.gradle:
        title: Gradlew file path
        env_key: GRADLEW_PATH
        value_map:
          android/gradlew:
            title: Project (or Workspace) path
            env_key: BITRISE_PROJECT_PATH
            value_map:
              ios/SampleAppsReactNativeAndroid.xcodeproj:
                title: Scheme name
                env_key: BITRISE_SCHEME
                value_map:
                  SampleAppsReactNativeAndroid:
                    config: react-native-android-ios-test-config
                  SampleAppsReactNativeAndroid-tvOS:
                    config: react-native-android-ios-test-config
configs:
  react-native:
    react-native-android-ios-test-config: |
      format_version: "%s"
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      project_type: react-native
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
          - npm@%s:
              inputs:
              - command: install
          - install-missing-android-tools@%s: {}
          - gradle-runner@%s:
              inputs:
              - gradle_file: $GRADLE_BUILD_FILE_PATH
              - gradle_task: assembleRelease
              - gradlew_path: $GRADLEW_PATH
          - certificate-and-profile-installer@%s: {}
          - xcode-archive@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
              - configuration: Release
          - deploy-to-bitrise-io@%s: {}
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - npm@%s:
              inputs:
              - command: install
          - npm@%s:
              inputs:
              - command: test
          - deploy-to-bitrise-io@%s: {}
warnings:
  react-native: []
`, sampleAppsReactNativeIosAndAndroidVersions...)
