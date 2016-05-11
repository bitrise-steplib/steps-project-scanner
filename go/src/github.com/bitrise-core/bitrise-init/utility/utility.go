package utility

import (
	"os"
	"path/filepath"
	"strings"
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

		rel = filepath.Join("./", rel)

		// fmt.Println(rel)

		fileList = append(fileList, rel)
		// fileList = append(fileList, path)

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
	c1 := strings.Split(s[i], string(os.PathSeparator))
	c2 := strings.Split(s[j], string(os.PathSeparator))

	return len(c1) < len(c2)
}
