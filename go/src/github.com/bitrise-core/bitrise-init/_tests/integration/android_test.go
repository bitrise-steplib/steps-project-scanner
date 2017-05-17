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

func TestAndroid(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__android__")
	require.NoError(t, err)

	t.Log("sample-apps-android-sdk22")
	{
		sampleAppDir := filepath.Join(tmpDir, "sample-apps-android-sdk22")
		sampleAppURL := "https://github.com/bitrise-samples/sample-apps-android-sdk22.git"
		require.NoError(t, git.Clone(sampleAppURL, sampleAppDir))

		cmd := command.New(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(sampleAppsAndroid22ResultYML), strings.TrimSpace(result))
	}

	t.Log("android-non-executable-gradlew")
	{
		sampleAppDir := filepath.Join(tmpDir, "android-non-executable-gradlew")
		sampleAppURL := "https://github.com/bitrise-samples/android-non-executable-gradlew.git"
		require.NoError(t, git.Clone(sampleAppURL, sampleAppDir))

		cmd := command.New(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(androidNonExecutableGradlewResultYML), strings.TrimSpace(result))
	}

	t.Log("android-sdk22-no-gradlew")
	{
		sampleAppDir := filepath.Join(tmpDir, "android-sdk22-no-gradlew")
		sampleAppURL := "https://github.com/bitrise-samples/android-sdk22-no-gradlew.git"
		require.NoError(t, git.Clone(sampleAppURL, sampleAppDir))

		cmd := command.New(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		_, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.EqualError(t, err, "exit status 1")

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(sampleAppsSDK22NoGradlewResultYML), strings.TrimSpace(result))
	}

	t.Log("android-sdk22-subdir")
	{
		sampleAppDir := filepath.Join(tmpDir, "android-sdk22-subdir")
		sampleAppURL := "https://github.com/bitrise-samples/sample-apps-android-sdk22-subdir"
		require.NoError(t, git.Clone(sampleAppURL, sampleAppDir))

		cmd := command.New(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		_, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(sampleAppsAndroidSDK22SubdirResultYML), strings.TrimSpace(result))
	}
}

var sampleAppsAndroidSDK22SubdirVersions = []interface{}{
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.InstallMissingAndroidToolsVersion,
	steps.GradleRunnerVersion,
	steps.DeployToBitriseIoVersion,
}

var sampleAppsAndroidSDK22SubdirResultYML = fmt.Sprintf(`options:
  android:
    title: Gradlew file path
    env_key: GRADLEW_PATH
    value_map:
      src/gradlew:
        title: Path to the gradle file to use
        env_key: GRADLE_BUILD_FILE_PATH
        value_map:
          src/build.gradle:
            title: Gradle task to run
            env_key: GRADLE_TASK
            value_map:
              assemble:
                config: android-config
              assembleDebug:
                config: android-config
              assembleRelease:
                config: android-config
configs:
  android:
    android-config: |
      format_version: "%s"
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      project_type: android
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
warnings:
  android: []
`, sampleAppsAndroidSDK22SubdirVersions...)

var sampleAppsSDK22NoGradlewResultYML = `warnings:
  android:
  - "<b>No Gradle Wrapper (gradlew) found.</b> \nUsing a Gradle Wrapper (gradlew)
    is required, as the wrapper is what makes sure\nthat the right Gradle version
    is installed and used for the build. More info/guide: <a>https://docs.gradle.org/current/userguide/gradle_wrapper.html</a>"
errors:
  general:
  - No known platform detected
`

var sampleAppsAndroid22Versions = []interface{}{
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.InstallMissingAndroidToolsVersion,
	steps.GradleRunnerVersion,
	steps.DeployToBitriseIoVersion,
}

var sampleAppsAndroid22ResultYML = fmt.Sprintf(`options:
  android:
    title: Gradlew file path
    env_key: GRADLEW_PATH
    value_map:
      ./gradlew:
        title: Path to the gradle file to use
        env_key: GRADLE_BUILD_FILE_PATH
        value_map:
          build.gradle:
            title: Gradle task to run
            env_key: GRADLE_TASK
            value_map:
              assemble:
                config: android-config
              assembleDebug:
                config: android-config
              assembleRelease:
                config: android-config
configs:
  android:
    android-config: |
      format_version: "%s"
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      project_type: android
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
warnings:
  android: []
`, sampleAppsAndroid22Versions...)

var androidNonExecutableGradlewVersions = []interface{}{
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.InstallMissingAndroidToolsVersion,
	steps.GradleRunnerVersion,
	steps.DeployToBitriseIoVersion,
}

var androidNonExecutableGradlewResultYML = fmt.Sprintf(`options:
  android:
    title: Gradlew file path
    env_key: GRADLEW_PATH
    value_map:
      ./gradlew:
        title: Path to the gradle file to use
        env_key: GRADLE_BUILD_FILE_PATH
        value_map:
          build.gradle:
            title: Gradle task to run
            env_key: GRADLE_TASK
            value_map:
              assemble:
                config: android-config
              assembleDebug:
                config: android-config
              assembleRelease:
                config: android-config
configs:
  android:
    android-config: |
      format_version: "%s"
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      project_type: android
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
warnings:
  android: []
`, androidNonExecutableGradlewVersions...)
