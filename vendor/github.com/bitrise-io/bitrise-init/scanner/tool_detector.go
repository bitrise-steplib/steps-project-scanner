package scanner

import (
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/sliceutil"

	"github.com/bitrise-io/go-utils/log"
)

const maxDepth = 4

// UnknownToolDetector ...
type UnknownToolDetector interface {
	// ToolName is the human-readable name of a tool/framework/platform
	ToolName() string

	// DetectToolIn should search recursively in rootPath looking for the given tool
	DetectToolIn(rootPath string) (DetectionResult, error)
}

// DetectionResult ...
type DetectionResult struct {
	Detected    bool
	ProjectTree string
}

// UnknownToolDetectors ...
var UnknownToolDetectors = []UnknownToolDetector{
	toolDetector{toolName: "Tuist", primaryFile: "Project.swift"},
	toolDetector{toolName: "Xcodegen", primaryFile: "project.yml"},
	toolDetector{toolName: "Bazel", primaryFile: "WORKSPACE", optionalFiles: []string{"WORKSPACE.bazel", "BUILD", "BUILD.bazel", ".bazelrc", ".bazelversion", ".bazelignore"}},
	toolDetector{toolName: "Buck", primaryFile: "BUCK", optionalFiles: []string{".buckversion", ".buckconfig", ".buckjavaargs"}},
	kotlinMultiplatformDetector{},
}

var excludedDirs = []string{
	".git",
	".idea",
	"node_modules",
	"Pods",
	"Carthage",
	"CordovaLib",
	".framework",
}

// toolDetector first tries to detect primaryFile in the directory, then one of optionalFiles as a fallback.
// It detects the tool if primaryFile is found OR one of optionalFiles
type toolDetector struct {
	toolName      string
	primaryFile   string
	optionalFiles []string
}

func (d toolDetector) ToolName() string {
	return d.toolName
}

func (d toolDetector) DetectToolIn(rootPath string) (DetectionResult, error) {
	fileNames, _, tree, err := walkProjectDir(rootPath)
	if err != nil {
		return DetectionResult{}, err
	}

	if sliceutil.IsStringInSlice(d.primaryFile, fileNames) {
		return DetectionResult{
			Detected:    true,
			ProjectTree: tree,
		}, nil
	}

	optionalFileDetected := false
	for _, fileName := range d.optionalFiles {
		if sliceutil.IsStringInSlice(fileName, fileNames) {
			optionalFileDetected = true
			break
		}
	}

	return DetectionResult{
		Detected:    optionalFileDetected,
		ProjectTree: tree,
	}, nil

}

type kotlinMultiplatformDetector struct{}

func (d kotlinMultiplatformDetector) ToolName() string {
	return "Kotlin Multiplatform"
}

func (d kotlinMultiplatformDetector) DetectToolIn(rootPath string) (DetectionResult, error) {
	fileNames, filePaths, tree, err := walkProjectDir(rootPath)
	if err != nil {
		return DetectionResult{}, err
	}

	fileNamePattern := `.+\.gradle(\.kts)?$`
	re, err := regexp.Compile(fileNamePattern)
	if err != nil {
		return DetectionResult{}, err
	}
	var potentialFilePaths []string
	for index, fileName := range fileNames {
		if re.MatchString(fileName) {
			potentialFilePaths = append(potentialFilePaths, filePaths[index])
		}
	}

	detected := false
	for _, path := range potentialFilePaths {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			log.Warnf(err.Error())
			continue
		}

		if d.canFindPatternIn(string(bytes)) {
			detected = true
			break
		}
	}

	return DetectionResult{
		Detected:    detected,
		ProjectTree: tree,
	}, err
}

func (d kotlinMultiplatformDetector) canFindPatternIn(fileContent string) bool {
	return strings.Contains(fileContent, `kotlin("multiplatform")`) ||
		strings.Contains(fileContent, `org.jetbrains.kotlin.multiplatform`)
}

// walkProjectDir recursively walks through every file and directory up to the defined depth limit while ignoring some
// directories. It returns with a list of fileNames, a list of (absolute) filePaths and a visual tree representation
// of the directory structure (taking the depth limit and ignored folders into account)
func walkProjectDir(rootPath string) (fileNames []string, filePaths []string, tree string, err error) {
	treeBuilder := strings.Builder{}

	err = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Warnf("error while traversing %s: %s", path, err)
			return filepath.SkipDir
		}

		relativePath, err := filepath.Rel(rootPath, path)
		if err != nil {
			log.Warnf(err.Error())
			return filepath.SkipDir
		}

		depth := strings.Count(relativePath, string(filepath.Separator)) + 1

		if d.IsDir() {
			if sliceutil.IsStringInSlice(d.Name(), excludedDirs) {
				return filepath.SkipDir
			}
			if depth > maxDepth {
				return filepath.SkipDir
			}
		}

		fileNames = append(fileNames, d.Name())
		filePaths = append(filePaths, path)

		var treePrefix = ""
		if depth > 1 {
			treePrefix = strings.Repeat("· ", depth-1)
		}
		var entryName = d.Name()
		if d.IsDir() {
			entryName = entryName + "/"
		}
		if relativePath != "." {
			treeBuilder.WriteString(treePrefix + entryName + "\n")
		}

		return nil
	})

	return fileNames, filePaths, treeBuilder.String(), err
}
