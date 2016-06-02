package cli

import (
	"fmt"
	"log"

	"github.com/bitrise-core/bitrise-init/output"
	"github.com/bitrise-core/bitrise-init/version"
	"github.com/codegangsta/cli"
)

// VersionOutputModel ...
type VersionOutputModel struct {
	Version     string `json:"version" yaml:"version"`
	BuildNumber string `json:"build_number" yaml:"build_number"`
	Commit      string `json:"commit" yaml:"commit"`
}

func printVersionCmd(c *cli.Context) {
	fullVersion := c.Bool("full")
	formatStr := c.String("format")

	if formatStr == "" {
		formatStr = output.YAMLFormat.String()
	}
	format, err := output.ParseFormat(formatStr)
	if err != nil {
		log.Fatalf("Failed to parse format, error: %s", err)
	}

	versionOutput := VersionOutputModel{
		Version: version.VERSION,
	}

	if fullVersion {
		versionOutput.BuildNumber = version.BuildNumber
		versionOutput.Commit = version.Commit
	}

	var out interface{}

	if format == output.RawFormat {
		if fullVersion {
			out = fmt.Sprintf("version: %v\nbuild_number: %v\ncommit: %v\n", versionOutput.Version, versionOutput.BuildNumber, versionOutput.Commit)
		} else {
			out = fmt.Sprintf("%v\n", versionOutput.Version)
		}
	} else {
		out = versionOutput
	}

	if err := output.Print(out, format, ""); err != nil {
		log.Fatalf("Failed to print version, error: %s", err)
	}
}
