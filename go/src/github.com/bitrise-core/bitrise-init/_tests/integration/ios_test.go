package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"strings"

	"github.com/bitrise-core/bitrise-init/models"
	"github.com/bitrise-core/bitrise-init/steps"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestIOS(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__ios__")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	}()

	t.Log("ios-no-shared-schemes")
	{
		sampleAppDir := filepath.Join(tmpDir, "ios-no-shared-scheme")
		sampleAppURL := "https://github.com/bitrise-samples/ios-no-shared-schemes.git"
		require.NoError(t, cmdex.GitClone(sampleAppURL, sampleAppDir))

		cmd := cmdex.NewCommand(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(iosNoSharedSchemesResultYML), strings.TrimSpace(result))
	}

	t.Log("ios-cocoapods-at-root")
	{
		sampleAppDir := filepath.Join(tmpDir, "ios-cocoapods-at-root")
		sampleAppURL := "https://github.com/bitrise-samples/ios-cocoapods-at-root.git"
		require.NoError(t, cmdex.GitClone(sampleAppURL, sampleAppDir))

		cmd := cmdex.NewCommand(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(iosCocoapodsAtRootResultYML), strings.TrimSpace(result))
	}

	t.Log("sample-apps-ios-watchkit")
	{
		sampleAppDir := filepath.Join(tmpDir, "sample-apps-ios-watchkit")
		sampleAppURL := "https://github.com/bitrise-io/sample-apps-ios-watchkit.git"
		require.NoError(t, cmdex.GitClone(sampleAppURL, sampleAppDir))

		cmd := cmdex.NewCommand(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(sampleAppsIosWatchkitResultYML), strings.TrimSpace(result))
	}
}

var sampleAppsIosWatchkitResultYML = fmt.Sprintf(`options:
  ios:
    title: Project (or Workspace) path
    env_key: BITRISE_PROJECT_PATH
    value_map:
      watch-test.xcodeproj:
        title: Scheme name
        env_key: BITRISE_SCHEME
        value_map:
          Complication - watch-test WatchKit App:
            config: ios-config
          Glance - watch-test WatchKit App:
            config: ios-config
          Notification - watch-test WatchKit App:
            config: ios-config
          watch-test:
            config: ios-test-config
          watch-test WatchKit App:
            config: ios-config
configs:
  ios:
    ios-config: |
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
          - deploy-to-bitrise-io@%s: {}
    ios-test-config: |
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
          - xcode-test@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - deploy-to-bitrise-io@%s: {}
warnings:
  ios: []
`, models.FormatVersion,
	steps.ActivateSSHKeyVersion, steps.GitCloneVersion, steps.ScriptVersion, steps.CertificateAndProfileInstallerVersion, steps.XcodeArchiveVersion, steps.DeployToBitriseIoVersion,
	steps.ActivateSSHKeyVersion, steps.GitCloneVersion, steps.ScriptVersion, steps.CertificateAndProfileInstallerVersion, steps.DeployToBitriseIoVersion,
	models.FormatVersion,
	steps.ActivateSSHKeyVersion, steps.GitCloneVersion, steps.ScriptVersion, steps.CertificateAndProfileInstallerVersion, steps.XcodeTestVersion, steps.XcodeArchiveVersion, steps.DeployToBitriseIoVersion,
	steps.ActivateSSHKeyVersion, steps.GitCloneVersion, steps.ScriptVersion, steps.CertificateAndProfileInstallerVersion, steps.XcodeTestVersion, steps.DeployToBitriseIoVersion)

var iosCocoapodsAtRootResultYML = fmt.Sprintf(`options:
  ios:
    title: Project (or Workspace) path
    env_key: BITRISE_PROJECT_PATH
    value_map:
      iOSMinimalCocoaPodsSample.xcodeproj:
        title: Scheme name
        env_key: BITRISE_SCHEME
        value_map:
          iOSMinimalCocoaPodsSample:
            config: ios-test-config
configs:
  ios:
    ios-test-config: |
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
          - xcode-test@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - deploy-to-bitrise-io@%s: {}
warnings:
  ios: []
`, models.FormatVersion,
	steps.ActivateSSHKeyVersion, steps.GitCloneVersion, steps.ScriptVersion, steps.CertificateAndProfileInstallerVersion, steps.XcodeTestVersion, steps.XcodeArchiveVersion, steps.DeployToBitriseIoVersion,
	steps.ActivateSSHKeyVersion, steps.GitCloneVersion, steps.ScriptVersion, steps.CertificateAndProfileInstallerVersion, steps.XcodeTestVersion, steps.DeployToBitriseIoVersion)

var iosNoSharedSchemesResultYML = fmt.Sprintf(`options:
  ios:
    title: Project (or Workspace) path
    env_key: BITRISE_PROJECT_PATH
    value_map:
      BitriseXcode7Sample.xcodeproj:
        title: Scheme name
        env_key: BITRISE_SCHEME
        value_map:
          BitriseXcode7Sample:
            config: ios-test-missing-shared-schemes-config
configs:
  ios:
    ios-test-missing-shared-schemes-config: |
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
          - recreate-user-schemes@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
          - xcode-test@%s:
              inputs:
              - project_path: $BITRISE_PROJECT_PATH
              - scheme: $BITRISE_SCHEME
          - deploy-to-bitrise-io@%s: {}
warnings:
  ios:
  - "No shared schemes found for project: BitriseXcode7Sample.xcodeproj.\n\tAutomatically
    generated schemes for this project.\n\tThese schemes may differ from the ones
    in your project.\n\tMake sure to <a href=\"https://developer.apple.com/library/ios/recipes/xcode_help-scheme_editor/Articles/SchemeManage.html\">share
    your schemes</a> for the expected behaviour."
`, models.FormatVersion,
	steps.ActivateSSHKeyVersion, steps.GitCloneVersion, steps.ScriptVersion, steps.CertificateAndProfileInstallerVersion, steps.RecreateUserSchemesVersion, steps.XcodeTestVersion, steps.XcodeArchiveVersion, steps.DeployToBitriseIoVersion,
	steps.ActivateSSHKeyVersion, steps.GitCloneVersion, steps.ScriptVersion, steps.CertificateAndProfileInstallerVersion, steps.RecreateUserSchemesVersion, steps.XcodeTestVersion, steps.DeployToBitriseIoVersion)
