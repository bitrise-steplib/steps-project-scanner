package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestFlutter(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__flutter__")
	require.NoError(t, err)

	t.Log("sample-apps-flutter-ios-android")
	{
		sampleAppDir := filepath.Join(tmpDir, "sample-apps-flutter-ios-android")
		sampleAppURL := "https://github.com/bitrise-samples/sample-apps-flutter-ios-android.git"
		gitClone(t, sampleAppDir, sampleAppURL)

		cmd := command.New(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)

		validateConfigExpectation(t, "sample-apps-flutter-ios-android", strings.TrimSpace(flutterSampleResultYML), strings.TrimSpace(result), flutterSampleVersions...)
	}
}

var flutterSampleVersions = []interface{}{
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.CertificateAndProfileInstallerVersion,
	steps.FlutterInstallVersion,
	steps.FlutterTestVersion,
	steps.FlutterBuildVersion,
	steps.XcodeArchiveVersion,
	steps.DeployToBitriseIoVersion,

	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.FlutterInstallVersion,
	steps.FlutterTestVersion,
	steps.DeployToBitriseIoVersion,
}

var flutterSampleResultYML = fmt.Sprintf(`options:
  flutter:
    title: Project Location
    env_key: BITRISE_FLUTTER_PROJECT_LOCATION
    value_map:
      .:
        title: Project (or Workspace) path
        env_key: BITRISE_PROJECT_PATH
        value_map:
          ios/Runner.xcworkspace:
            title: Scheme name
            env_key: BITRISE_SCHEME
            value_map:
              Runner:
                title: ipa export method
                env_key: BITRISE_EXPORT_METHOD
                value_map:
                  ad-hoc:
                    config: flutter-config
                  app-store:
                    config: flutter-config
                  development:
                    config: flutter-config
                  enterprise:
                    config: flutter-config
configs:
  flutter:
    flutter-config: |
      format_version: "%s"
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      project_type: flutter
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
          - flutter-installer@%s: {}
          - flutter-test@%s:
              inputs:
              - project_location: $BITRISE_FLUTTER_PROJECT_LOCATION
          - flutter-build@%s:
              inputs:
              - project_location: $BITRISE_FLUTTER_PROJECT_LOCATION
          - xcode-archive@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
              - export_method: $BITRISE_EXPORT_METHOD
              - configuration: Release
          - deploy-to-bitrise-io@%s: {}
        primary:
          steps:
          - activate-ssh-key@%s:
              run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
          - git-clone@%s: {}
          - script@%s:
              title: Do anything with Script step
          - flutter-installer@%s: {}
          - flutter-test@%s:
              inputs:
              - project_location: $BITRISE_FLUTTER_PROJECT_LOCATION
          - deploy-to-bitrise-io@%s: {}
warnings:
  flutter: []
`, flutterSampleVersions...)
