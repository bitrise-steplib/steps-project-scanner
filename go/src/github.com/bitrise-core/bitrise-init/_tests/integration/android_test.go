package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestAndroid(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__android__")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	}()

	t.Log("sample-apps-android-sdk22")
	{
		sampleAppDir := filepath.Join(tmpDir, "sample-apps-android-sdk22")
		sampleAppURL := "https://github.com/bitrise-samples/sample-apps-android-sdk22.git"
		require.NoError(t, cmdex.GitClone(sampleAppURL, sampleAppDir))

		cmd := cmdex.NewCommand(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
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
		require.NoError(t, cmdex.GitClone(sampleAppURL, sampleAppDir))

		cmd := cmdex.NewCommand(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(androidNonExecutableGradlewResultYML), strings.TrimSpace(result))
	}
}

var sampleAppsAndroid22Versions = []interface{}{
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.ScriptVersion,
	steps.GradleRunnerVersion,
	steps.DeployToBitriseIoVersion,
}

var sampleAppsAndroid22ResultYML = fmt.Sprintf(`options:
  android:
    title: Path to the gradle file to use
    env_key: GRADLE_BUILD_FILE_PATH
    value_map:
      build.gradle:
        title: Gradle task to run
        env_key: GRADLE_TASK
        value_map:
          assemble:
            title: Gradlew file path
            env_key: GRADLEW_PATH
            value_map:
              ./gradlew:
                config: android-config
          assembleDebug:
            title: Gradlew file path
            env_key: GRADLEW_PATH
            value_map:
              ./gradlew:
                config: android-config
          assembleRelease:
            title: Gradlew file path
            env_key: GRADLEW_PATH
            value_map:
              ./gradlew:
                config: android-config
configs:
  android:
    android-config: |
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
          - script@%s:
              title: Update Android Extra packages
              inputs:
              - content: |
                  #!/bin/bash
                  set -ex

                  echo y | android update sdk --no-ui --all --filter platform-tools | grep 'package installed'

                  echo y | android update sdk --no-ui --all --filter extra-android-m2repository | grep 'package installed'
                  echo y | android update sdk --no-ui --all --filter extra-google-m2repository | grep 'package installed'
                  echo y | android update sdk --no-ui --all --filter extra-google-google_play_services | grep 'package installed'
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
	steps.ScriptVersion,
	steps.GradleRunnerVersion,
	steps.DeployToBitriseIoVersion,
}

var androidNonExecutableGradlewResultYML = fmt.Sprintf(`options:
  android:
    title: Path to the gradle file to use
    env_key: GRADLE_BUILD_FILE_PATH
    value_map:
      build.gradle:
        title: Gradle task to run
        env_key: GRADLE_TASK
        value_map:
          assemble:
            title: Gradlew file path
            env_key: GRADLEW_PATH
            value_map:
              ./gradlew:
                config: android-config
          assembleDebug:
            title: Gradlew file path
            env_key: GRADLEW_PATH
            value_map:
              ./gradlew:
                config: android-config
          assembleRelease:
            title: Gradlew file path
            env_key: GRADLEW_PATH
            value_map:
              ./gradlew:
                config: android-config
configs:
  android:
    android-config: |
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
          - script@%s:
              title: Update Android Extra packages
              inputs:
              - content: |
                  #!/bin/bash
                  set -ex

                  echo y | android update sdk --no-ui --all --filter platform-tools | grep 'package installed'

                  echo y | android update sdk --no-ui --all --filter extra-android-m2repository | grep 'package installed'
                  echo y | android update sdk --no-ui --all --filter extra-google-m2repository | grep 'package installed'
                  echo y | android update sdk --no-ui --all --filter extra-google-google_play_services | grep 'package installed'
          - gradle-runner@%s:
              inputs:
              - gradle_file: $GRADLE_BUILD_FILE_PATH
              - gradle_task: $GRADLE_TASK
              - gradlew_path: $GRADLEW_PATH
          - deploy-to-bitrise-io@%s: {}
warnings:
  android: []
`, androidNonExecutableGradlewVersions...)
