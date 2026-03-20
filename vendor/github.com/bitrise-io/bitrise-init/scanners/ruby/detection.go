package ruby

import (
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

type testFramework struct {
	name           string
	detectionFiles []string
}

var testFrameworks = []testFramework{
	{"rspec", []string{"spec/spec_helper.rb", ".rspec"}},
	{"minitest", []string{"test/test_helper.rb"}},
}

func collectGemfiles(searchDir string) ([]string, error) {
	fileList, err := pathutil.ListPathInDirSortedByComponents(searchDir, false)
	if err != nil {
		return nil, err
	}

	filters := []pathutil.FilterFunc{
		pathutil.BaseFilter("Gemfile", true),
		pathutil.ComponentFilter("node_modules", false),
		pathutil.ComponentFilter("Pods", false),
		pathutil.ComponentFilter("Carthage", false),
		pathutil.ComponentFilter(".git", false),
	}
	return pathutil.FilterPaths(fileList, filters...)
}

func checkBundler(searchDir string) bool {
	log.TPrintf("Checking for Bundler")
	hasGemfileLock := utility.FileExists(filepath.Join(searchDir, "Gemfile.lock"))

	if !hasGemfileLock {
		log.TPrintf("- Gemfile.lock - not found")
		return false
	}

	log.TPrintf("- Gemfile.lock - found")
	log.TPrintf("Bundler: detected")
	return true
}

func checkRakefile(searchDir string) bool {
	log.TPrintf("Checking for Rakefile")
	hasRakefile := utility.FileExists(filepath.Join(searchDir, "Rakefile"))

	if !hasRakefile {
		log.TPrintf("- Rakefile - not found")
		return false
	}

	log.TPrintf("- Rakefile - found")
	return true
}

// readRubyVersion returns the Ruby version declared in .ruby-version or .tool-versions,
// or an empty string if no version file is found.
func readRubyVersion(searchDir string) string {
	log.TPrintf("Checking for Ruby version file")

	// .ruby-version: single line containing the version (e.g. "3.3.0" or "ruby-3.3.0")
	rubyVersionPath := filepath.Join(searchDir, ".ruby-version")
	if content, err := fileutil.ReadStringFromFile(rubyVersionPath); err == nil {
		version := strings.TrimSpace(content)
		version = strings.TrimPrefix(version, "ruby-")
		if version != "" {
			log.TPrintf("- .ruby-version - found (%s)", version)
			return version
		}
	}

	// .tool-versions: asdf format, one tool per line (e.g. "ruby 3.3.0")
	toolVersionsPath := filepath.Join(searchDir, ".tool-versions")
	if content, err := fileutil.ReadStringFromFile(toolVersionsPath); err == nil {
		for _, line := range strings.Split(content, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "ruby" {
				log.TPrintf("- .tool-versions - found ruby %s", fields[1])
				return fields[1]
			}
		}
	}

	log.TPrintf("- Ruby version file - not found")
	return ""
}

func detectTestFramework(searchDir string) string {
	log.TPrintf("Checking test framework")

	for _, fw := range testFrameworks {
		for _, detectionFile := range fw.detectionFiles {
			if utility.FileExists(filepath.Join(searchDir, detectionFile)) {
				log.TPrintf("- %s - found (%s)", fw.name, detectionFile)
				return fw.name
			}
		}
	}

	log.TPrintf("- test framework - not detected")
	return ""
}

func detectRails(searchDir string) bool {
	gemfilePath := filepath.Join(searchDir, "Gemfile")
	content, err := fileutil.ReadStringFromFile(gemfilePath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(content, "\n") {
		match := gemDeclPattern.FindStringSubmatch(line)
		if len(match) >= 2 && match[1] == "rails" {
			return true
		}
	}
	return false
}
