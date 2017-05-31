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

func TestIonic(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__ionic__")
	require.NoError(t, err)

	t.Log("sample-apps-ionic")
	{
		sampleAppDir := filepath.Join(tmpDir, "sample-apps-ionic")
		sampleAppURL := "https://github.com/driftyco/ionic-conference-app.git"
		require.NoError(t, git.Clone(sampleAppURL, sampleAppDir))

		cmd := command.New(binPath(), "--ci", "config", "--dir", sampleAppDir, "--output-dir", sampleAppDir)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)

		scanResultPth := filepath.Join(sampleAppDir, "result.yml")

		result, err := fileutil.ReadStringFromFile(scanResultPth)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSpace(sampleAppsIonicResultYML), strings.TrimSpace(result))
	}
}

var sampleAppsIonicVersions = []interface{}{
	models.FormatVersion,
	steps.ActivateSSHKeyVersion,
	steps.GitCloneVersion,
	steps.ScriptVersion,
	steps.IonicBuildVersion,
	steps.DeployToBitriseIoVersion,
}

var sampleAppsIonicResultYML = fmt.Sprintf(`options:
  ionic:
    title: Platform to use in ionic-cli commands
    env_key: IONIC_PLATFORM
    value_map:
      android:
        config: ionic-config
      ios:
        config: ionic-config
configs:
  ionic:
    ionic-config: |
      format_version: "%s"
      default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
      project_type: ionic
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
          - ionic-build@%s:
              inputs:
              - build_for_platform: $IONIC_PLATFORM
          - deploy-to-bitrise-io@%s: {}
warnings:
  ionic: []
`, sampleAppsIonicVersions...)
