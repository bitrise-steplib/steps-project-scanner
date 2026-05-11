package python

import (
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

var markerFiles = []string{
	"requirements.txt",
	"pyproject.toml",
	"Pipfile",
	"setup.py",
}

// requirementsFiles are checked in order when looking for pytest as a dependency.
var requirementsFiles = []string{
	"requirements.txt",
	"requirements-dev.txt",
	"requirements-test.txt",
	"requirements_dev.txt",
	"requirements_test.txt",
}

func collectPythonProjectDirs(searchDir string) ([]string, error) {
	fileList, err := pathutil.ListPathInDirSortedByComponents(searchDir, false)
	if err != nil {
		return nil, err
	}

	excludeFilters := []pathutil.FilterFunc{
		pathutil.ComponentFilter(".git", false),
		pathutil.ComponentFilter("node_modules", false),
		pathutil.ComponentFilter(".venv", false),
		pathutil.ComponentFilter("venv", false),
		pathutil.ComponentFilter("__pycache__", false),
	}

	seenDirs := map[string]bool{}
	var projectDirs []string

	for _, markerFile := range markerFiles {
		filters := append(excludeFilters, pathutil.BaseFilter(markerFile, true))
		paths, err := pathutil.FilterPaths(fileList, filters...)
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			dir := filepath.Dir(p)
			if !seenDirs[dir] {
				seenDirs[dir] = true
				projectDirs = append(projectDirs, dir)
			}
		}
	}

	return projectDirs, nil
}

func detectPackageManager(projectDir string) string {
	log.TPrintf("Checking package manager")

	if utility.FileExists(filepath.Join(projectDir, "uv.lock")) {
		log.TPrintf("- uv.lock - found")
		return "uv"
	}

	if utility.FileExists(filepath.Join(projectDir, "poetry.lock")) {
		log.TPrintf("- poetry.lock - found")
		return "poetry"
	}

	if utility.FileExists(filepath.Join(projectDir, "requirements.txt")) {
		log.TPrintf("- requirements.txt - found")
		return "pip"
	}

	log.TPrintf("- package manager - not detected")
	return ""
}

func detectPythonVersion(projectDir string) string {
	log.TPrintf("Checking Python version")

	// .python-version — single line (e.g. "3.12")
	if content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, ".python-version")); err == nil {
		version := strings.TrimSpace(content)
		if version != "" {
			log.TPrintf("- .python-version - found (%s)", version)
			return version
		}
	}

	// .tool-versions — asdf/mise format: "python 3.12.x"
	if content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, ".tool-versions")); err == nil {
		for _, line := range strings.Split(content, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "python" {
				log.TPrintf("- .tool-versions - found python %s", fields[1])
				return fields[1]
			}
		}
	}

	// pyproject.toml — requires-python field (e.g. requires-python = ">=3.12")
	if version := pyprojectRequiresPython(projectDir); version != "" {
		log.TPrintf("- pyproject.toml requires-python - found (%s)", version)
		return version
	}

	log.TPrintf("- Python version - not found")
	return ""
}

// pyprojectRequiresPython extracts a version string from the requires-python field in pyproject.toml.
func pyprojectRequiresPython(projectDir string) string {
	content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, "pyproject.toml"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "requires-python") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}
		raw := strings.Trim(strings.TrimSpace(parts[1]), `"' `)
		// Strip leading operators: >=, ==, ~=, ^, >
		i := 0
		for i < len(raw) {
			c := raw[i]
			if c == '>' || c == '<' || c == '=' || c == '~' || c == '^' || c == ' ' {
				i++
			} else {
				break
			}
		}
		version := strings.TrimSpace(raw[i:])
		// Take the first token in case of compound ranges like ">=3.12,<4"
		if parts := strings.FieldsFunc(version, func(r rune) bool { return r == ',' || r == ' ' }); len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

func detectTestRunner(projectDir string) bool {
	log.TPrintf("Checking test runner")

	if utility.FileExists(filepath.Join(projectDir, "pytest.ini")) {
		log.TPrintf("- pytest.ini - found")
		return true
	}

	if utility.FileExists(filepath.Join(projectDir, "conftest.py")) {
		log.TPrintf("- conftest.py - found")
		return true
	}

	if hasPytestInPyprojectToml(projectDir) {
		log.TPrintf("- [tool.pytest] in pyproject.toml - found")
		return true
	}

	if hasPytestInRequirementsFiles(projectDir) {
		log.TPrintf("- pytest in requirements files - found")
		return true
	}

	log.TPrintf("- test runner - not detected")
	return false
}

func hasPytestInPyprojectToml(projectDir string) bool {
	content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, "pyproject.toml"))
	if err != nil {
		return false
	}
	return strings.Contains(content, "[tool.pytest")
}

func hasPytestInRequirementsFiles(projectDir string) bool {
	for _, name := range requirementsFiles {
		content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, name))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
				continue
			}
			pkg := packageName(line)
			if strings.EqualFold(pkg, "pytest") {
				return true
			}
		}
	}
	return false
}

// detectDevRequirementsFile returns the first dev/test requirements file found in projectDir, or "".
func detectDevRequirementsFile(projectDir string) string {
	devFiles := []string{
		"requirements-dev.txt",
		"requirements-test.txt",
		"requirements_dev.txt",
		"requirements_test.txt",
	}
	for _, name := range devFiles {
		if utility.FileExists(filepath.Join(projectDir, name)) {
			log.TPrintf("- dev requirements: %s - found", name)
			return name
		}
	}
	return ""
}

// packageName extracts the package name from a requirements.txt line by stripping version specifiers.
func packageName(line string) string {
	i := strings.IndexAny(line, "=<>![ ;#\t")
	if i == -1 {
		return line
	}
	return strings.TrimSpace(line[:i])
}

func detectFramework(projectDir string) string {
	log.TPrintf("Checking framework")

	frameworks := []string{"fastapi", "django", "flask"}

	content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, "requirements.txt"))
	if err != nil {
		log.TPrintf("- framework - requirements.txt not found")
		return ""
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pkg := strings.ToLower(packageName(line))
		for _, fw := range frameworks {
			if pkg == fw {
				log.TPrintf("- framework: %s", fw)
				return fw
			}
		}
	}

	log.TPrintf("- framework - not detected")
	return ""
}
