package utility

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/go-utils/pathutil"
)

// CaseInsensitiveContains ...
func CaseInsensitiveContains(s, substr string) bool {
	s, substr = strings.ToUpper(s), strings.ToUpper(substr)
	return strings.Contains(s, substr)
}

// FileList ...
func FileList(searchDir string) ([]string, error) {
	searchDir, err := filepath.Abs(searchDir)
	if err != nil {
		return []string{}, err
	}

	fileList := []string{}

	if err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		rel, err := filepath.Rel(searchDir, path)
		if err != nil {
			return err
		}

		fileList = append(fileList, rel)

		return nil
	}); err != nil {
		return []string{}, err
	}
	return fileList, nil
}

// FilterFilesWithBasPaths ...
func FilterFilesWithBasPaths(fileList []string, basePath ...string) []string {
	filteredFileList := []string{}

	for _, file := range fileList {
		base := filepath.Base(file)

		for _, desiredBasePath := range basePath {
			if strings.EqualFold(base, desiredBasePath) {
				filteredFileList = append(filteredFileList, file)
				break
			}
		}
	}

	return filteredFileList
}

// FilterFilesWithExtensions ...
func FilterFilesWithExtensions(fileList []string, extension ...string) []string {
	filteredFileList := []string{}

	for _, file := range fileList {
		ext := filepath.Ext(file)

		for _, desiredExt := range extension {
			if ext == desiredExt {
				filteredFileList = append(filteredFileList, file)
				break
			}
		}
	}

	return filteredFileList
}

// PathDept ...
func PathDept(pth string) (int, error) {
	abs, err := pathutil.AbsPath(pth)
	if err != nil {
		return 0, err
	}
	comp := strings.Split(abs, string(os.PathSeparator))

	fixedComp := []string{}
	for _, c := range comp {
		if c != "" {
			fixedComp = append(fixedComp, c)
		}
	}

	return len(fixedComp), nil
}

// MapStringStringHasValue ...
func MapStringStringHasValue(mapStringString map[string]string, value string) bool {
	for _, v := range mapStringString {
		if v == value {
			return true
		}
	}
	return false
}

//--------------------------------------------------
// Sorting
//--------------------------------------------------

// ByComponents ..
type ByComponents []string

func (s ByComponents) Len() int {
	return len(s)
}
func (s ByComponents) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByComponents) Less(i, j int) bool {
	path1 := s[i]
	path2 := s[j]

	d1, err := PathDept(path1)
	if err != nil {
		log.Warnf("failed to calculate path depth (%s), error: %s", path1, err)
		return false
	}

	d2, err := PathDept(path2)
	if err != nil {
		log.Warnf("failed to calculate path depth (%s), error: %s", path1, err)
		return false
	}

	if d1 < d2 {
		return true
	} else if d1 > d2 {
		return false
	}

	// if same component size,
	// do alphabetic sort based on the last component
	base1 := filepath.Base(path1)
	base2 := filepath.Base(path2)

	return base1 < base2
}
