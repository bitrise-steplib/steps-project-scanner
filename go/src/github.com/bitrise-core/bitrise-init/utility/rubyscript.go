package utility

import (
	"errors"
	"path"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

func runRubyScriptForOutput(scriptContent, gemfileContent, inDir string, withEnvs []string) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__bitrise-init__")
	if err != nil {
		return "", err
	}

	// Write Gemfile to file and install
	if gemfileContent != "" {
		gemfilePth := path.Join(tmpDir, "Gemfile")
		if err := fileutil.WriteStringToFile(gemfilePth, gemfileContent); err != nil {
			return "", err
		}

		cmd := cmdex.NewCommand("bundle", "install")

		if inDir != "" {
			cmd.SetDir(inDir)
		}

		withEnvs = append(withEnvs, "BUNDLE_GEMFILE="+gemfilePth)
		cmd.AppendEnvs(withEnvs)

		if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			if errorutil.IsExitStatusError(err) {
				return "", errors.New(out)
			}
			return "", err
		}
	}

	// Write script to file and run
	rubyScriptPth := path.Join(tmpDir, "script.rb")
	if err := fileutil.WriteStringToFile(rubyScriptPth, scriptContent); err != nil {
		return "", err
	}

	var cmd *cmdex.CommandModel

	if gemfileContent != "" {
		cmd = cmdex.NewCommand("bundle", "exec", "ruby", rubyScriptPth)
	} else {
		cmd = cmdex.NewCommand("ruby", rubyScriptPth)
	}

	if inDir != "" {
		cmd.SetDir(inDir)
	}

	if len(withEnvs) > 0 {
		cmd.AppendEnvs(withEnvs)
	}

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return "", errors.New(out)
		}
		return "", err
	}

	return out, nil
}
