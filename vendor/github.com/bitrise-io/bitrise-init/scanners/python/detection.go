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

type pyprojectInfo struct {
	poetryPackageModeDisabled bool
	poetryHasPackagesField bool
	poetryName             string
	projectName            string
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

// detectFramework logs which Python web framework the project uses, if any.
// The result is only surfaced through scan logs; it doesn't affect the
// generated workflow.
func detectFramework(projectDir string) {
	log.TPrintf("Checking framework")

	frameworks := []string{"fastapi", "django", "flask"}

	content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, "requirements.txt"))
	if err != nil {
		log.TPrintf("- framework - requirements.txt not found")
		return
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
				return
			}
		}
	}

	log.TPrintf("- framework - not detected")
}

// detectPoetryNeedsNoRoot decides whether `poetry install` should be invoked
// with --no-root. Plain `poetry install` fails when poetry is in package mode
// but the project doesn't ship an installable package, which is the common app
// case (FastAPI/Django/Flask services). For library projects with a real
// package layout we skip --no-root so the package is installed for tests that
// rely on entry points or installed metadata.
//
// Heuristics in priority order:
//  1. `package-mode = false` in [tool.poetry]                              -> no --no-root
//  2. explicit `packages = ...` in [tool.poetry]                           -> no --no-root
//  3. project name resolves to <dir>/__init__.py or src/<dir>/__init__.py  -> no --no-root
//  4. otherwise                                                            -> use --no-root
func detectPoetryNeedsNoRoot(projectDir string) bool {
	log.TPrintf("Checking Poetry --no-root requirement")

	content, err := fileutil.ReadStringFromFile(filepath.Join(projectDir, "pyproject.toml"))
	if err != nil {
		log.TPrintf("- pyproject.toml - not found, using --no-root")
		return true
	}

	info := parsePyproject(content)
	if info.poetryPackageModeDisabled {
		log.TPrintf("- package-mode = false - found, plain install")
		return false
	}
	if info.poetryHasPackagesField {
		log.TPrintf("- [tool.poetry] packages - declared, plain install")
		return false
	}

	name := info.poetryName
	if name == "" {
		name = info.projectName
	}
	if name == "" {
		log.TPrintf("- project name - not found, using --no-root")
		return true
	}

	pkgDir := strings.ReplaceAll(name, "-", "_")
	if utility.FileExists(filepath.Join(projectDir, pkgDir, "__init__.py")) {
		log.TPrintf("- %s/__init__.py - found, plain install", pkgDir)
		return false
	}
	if utility.FileExists(filepath.Join(projectDir, "src", pkgDir, "__init__.py")) {
		log.TPrintf("- src/%s/__init__.py - found, plain install", pkgDir)
		return false
	}

	log.TPrintf("- installable package layout - not found, using --no-root")
	return true
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

// packageName extracts the package name from a requirements.txt line by stripping version specifiers.
func packageName(line string) string {
	i := strings.IndexAny(line, "=<>![ ;#\t")
	if i == -1 {
		return line
	}
	return strings.TrimSpace(line[:i])
}

func parsePyproject(content string) pyprojectInfo {
	info := pyprojectInfo{}
	section := ""
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = trimmed
			continue
		}
		switch section {
		case "[tool.poetry]":
			if strings.HasPrefix(trimmed, "package-mode") {
				if parts := strings.SplitN(trimmed, "=", 2); len(parts) == 2 && strings.TrimSpace(parts[1]) == "false" {
					info.poetryPackageModeDisabled = true
				}
			}
			if strings.HasPrefix(trimmed, "packages") && strings.Contains(trimmed, "=") {
				info.poetryHasPackagesField = true
			}
			if strings.HasPrefix(trimmed, "name") {
				if v := pyprojectStringValue(trimmed); v != "" {
					info.poetryName = v
				}
			}
		case "[project]":
			if strings.HasPrefix(trimmed, "name") {
				if v := pyprojectStringValue(trimmed); v != "" {
					info.projectName = v
				}
			}
		}
	}
	return info
}

func pyprojectStringValue(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(parts[1]), `"'`)
}
