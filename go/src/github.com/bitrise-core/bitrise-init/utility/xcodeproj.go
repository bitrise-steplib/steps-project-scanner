package utility

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bitrise-tools/go-xcode/xcodeproj"
)

var (
	embeddedWorkspacePathRegexp    = regexp.MustCompile(`.+\.xcodeproj/.+\.xcworkspace`)
	scanProjectPathRegexpBlackList = []*regexp.Regexp{embeddedWorkspacePathRegexp}

	gitFolderName           = ".git"
	podsFolderName          = "Pods"
	carthageFolderName      = "Carthage"
	scanFolderNameBlackList = []string{gitFolderName, podsFolderName, carthageFolderName}

	frameworkExt           = ".framework"
	scanFolderExtBlackList = []string{frameworkExt}
)

func isPathMatchRegexp(pth string, regexp *regexp.Regexp) bool {
	return (regexp.FindString(pth) != "")
}

func isPathContainsComponent(pth, component string) bool {
	pathComponents := strings.Split(pth, string(filepath.Separator))
	for _, c := range pathComponents {
		if c == component {
			return true
		}
	}
	return false
}

func isPathContainsComponentWithExtension(pth, ext string) bool {
	pathComponents := strings.Split(pth, string(filepath.Separator))
	for _, c := range pathComponents {
		e := filepath.Ext(c)
		if e == ext {
			return true
		}
	}
	return false
}

func isDir(pth string) (bool, error) {
	fileInf, err := os.Lstat(pth)
	if err != nil {
		return false, err
	}
	if fileInf == nil {
		return false, errors.New("no file info available")
	}
	return fileInf.IsDir(), nil
}

func isRelevantProject(pth string, isTest bool) (bool, error) {
	// xcodeproj & xcworkspace should be a dir
	if !isTest {
		if is, err := isDir(pth); err != nil {
			return false, err
		} else if !is {
			return false, nil
		}
	}

	for _, regexp := range scanProjectPathRegexpBlackList {
		if isPathMatchRegexp(pth, regexp) {
			return false, nil
		}
	}

	for _, folderName := range scanFolderNameBlackList {
		if isPathContainsComponent(pth, folderName) {
			return false, nil
		}
	}

	for _, folderExt := range scanFolderExtBlackList {
		if isPathContainsComponentWithExtension(pth, folderExt) {
			return false, nil
		}
	}

	return true, nil
}

// FilterRelevantXcodeProjectFiles ...
func FilterRelevantXcodeProjectFiles(fileList []string, isTest bool) ([]string, error) {
	filteredFiles := FilterFilesWithExtensions(fileList, xcodeproj.XCodeProjExt, xcodeproj.XCWorkspaceExt)
	relevantFiles := []string{}

	for _, file := range filteredFiles {
		is, err := isRelevantProject(file, isTest)
		if err != nil {
			return []string{}, err
		} else if !is {
			continue
		}

		relevantFiles = append(relevantFiles, file)
	}

	sort.Sort(ByComponents(relevantFiles))

	return relevantFiles, nil
}

func isRelevantPodfile(pth string) bool {
	basename := filepath.Base(pth)
	if !CaseInsensitiveEquals(basename, "podfile") {
		return false
	}

	for _, folderName := range scanFolderNameBlackList {
		if isPathContainsComponent(pth, folderName) {
			return false
		}
	}

	for _, folderExt := range scanFolderExtBlackList {
		if isPathContainsComponentWithExtension(pth, folderExt) {
			return false
		}
	}

	return true
}

// FilterRelevantPodFiles ...
func FilterRelevantPodFiles(fileList []string) []string {
	podfiles := []string{}

	for _, file := range fileList {
		if isRelevantPodfile(file) {
			podfiles = append(podfiles, file)
		}
	}

	if len(podfiles) == 0 {
		return []string{}
	}

	sort.Sort(ByComponents(podfiles))

	return podfiles
}
