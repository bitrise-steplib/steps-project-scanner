package xcodeproj

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Extensions
const (
	// XCWorkspaceExt ...
	XCWorkspaceExt = ".xcworkspace"
	// XCodeProjExt ...
	XCodeProjExt = ".xcodeproj"
	// XCSchemeExt ...
	XCSchemeExt = ".xcscheme"
)

// IsXCodeProj ...
func IsXCodeProj(pth string) bool {
	return strings.HasSuffix(pth, XCodeProjExt)
}

// IsXCWorkspace ...
func IsXCWorkspace(pth string) bool {
	return strings.HasSuffix(pth, XCWorkspaceExt)
}

// SchemeNameFromPath ...
func SchemeNameFromPath(schemePth string) string {
	basename := filepath.Base(schemePth)
	ext := filepath.Ext(schemePth)
	if ext != XCSchemeExt {
		return ""
	}
	return strings.TrimSuffix(basename, ext)
}

// SchemeFileContainsXCTestBuildAction ...
func SchemeFileContainsXCTestBuildAction(schemeFilePth string) (bool, error) {
	content, err := fileutil.ReadStringFromFile(schemeFilePth)
	if err != nil {
		return false, err
	}

	return schemeFileContentContainsXCTestBuildAction(content)
}

// ProjectSharedSchemeFilePaths ...
func ProjectSharedSchemeFilePaths(projectPth string) ([]string, error) {
	return sharedSchemeFilePaths(projectPth)
}

// WorkspaceSharedSchemeFilePaths ...
func WorkspaceSharedSchemeFilePaths(workspacePth string) ([]string, error) {
	workspaceSchemeFilePaths, err := sharedSchemeFilePaths(workspacePth)
	if err != nil {
		return []string{}, err
	}

	projects, err := WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		projectSchemeFilePaths, err := sharedSchemeFilePaths(project)
		if err != nil {
			return []string{}, err
		}
		workspaceSchemeFilePaths = append(workspaceSchemeFilePaths, projectSchemeFilePaths...)
	}

	sort.Strings(workspaceSchemeFilePaths)

	return workspaceSchemeFilePaths, nil
}

// ProjectSharedSchemes ...
func ProjectSharedSchemes(projectPth string) (map[string]bool, error) {
	return sharedSchemes(projectPth)
}

// WorkspaceSharedSchemes ...
func WorkspaceSharedSchemes(workspacePth string) (map[string]bool, error) {
	schemeMap, err := sharedSchemes(workspacePth)
	if err != nil {
		return map[string]bool{}, err
	}

	projects, err := WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		projectSchemeMap, err := sharedSchemes(project)
		if err != nil {
			return map[string]bool{}, err
		}

		for name, hasXCtest := range projectSchemeMap {
			schemeMap[name] = hasXCtest
		}
	}

	return schemeMap, nil
}

// ProjectUserSchemeFilePaths ...
func ProjectUserSchemeFilePaths(projectPth string) ([]string, error) {
	return userSchemeFilePaths(projectPth)
}

// WorkspaceUserSchemeFilePaths ...
func WorkspaceUserSchemeFilePaths(workspacePth string) ([]string, error) {
	workspaceSchemeFilePaths, err := userSchemeFilePaths(workspacePth)
	if err != nil {
		return []string{}, err
	}

	projects, err := WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		projectSchemeFilePaths, err := userSchemeFilePaths(project)
		if err != nil {
			return []string{}, err
		}
		workspaceSchemeFilePaths = append(workspaceSchemeFilePaths, projectSchemeFilePaths...)
	}

	sort.Strings(workspaceSchemeFilePaths)

	return workspaceSchemeFilePaths, nil
}

// ProjectUserSchemes ...
func ProjectUserSchemes(projectPth string) (map[string]bool, error) {
	return userSchemes(projectPth)
}

// WorkspaceUserSchemes ...
func WorkspaceUserSchemes(workspacePth string) (map[string]bool, error) {
	schemeMap, err := userSchemes(workspacePth)
	if err != nil {
		return map[string]bool{}, err
	}

	projects, err := WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		projectSchemeMap, err := userSchemes(project)
		if err != nil {
			return map[string]bool{}, err
		}

		for name, hasXCtest := range projectSchemeMap {
			schemeMap[name] = hasXCtest
		}
	}

	return schemeMap, nil
}

func runRubyScriptForOutput(scriptContent, gemfileContent, inDir string, withEnvs []string) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("bitrise")
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
		cmd.SetEnvs(withEnvs)

		var outBuffer bytes.Buffer
		outWriter := bufio.NewWriter(&outBuffer)
		cmd.SetStdout(outWriter)

		var errBuffer bytes.Buffer
		errWriter := bufio.NewWriter(&errBuffer)
		cmd.SetStderr(errWriter)

		if err := cmd.Run(); err != nil {
			if errorutil.IsExitStatusError(err) {
				errMsg := ""
				if errBuffer.String() != "" {
					errMsg += fmt.Sprintf("error: %s\n", errBuffer.String())
				}
				if outBuffer.String() != "" {
					errMsg += fmt.Sprintf("output: %s", outBuffer.String())
				}
				if errMsg == "" {
					return "", err
				}

				return "", errors.New(errMsg)
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
		cmd.SetEnvs(withEnvs)
	}

	var outBuffer bytes.Buffer
	outWriter := bufio.NewWriter(&outBuffer)
	cmd.SetStdout(outWriter)

	var errBuffer bytes.Buffer
	errWriter := bufio.NewWriter(&errBuffer)
	cmd.SetStderr(errWriter)

	if err := cmd.Run(); err != nil {
		if errorutil.IsExitStatusError(err) {
			errMsg := ""
			if errBuffer.String() != "" {
				errMsg += fmt.Sprintf("error: %s\n", errBuffer.String())
			}
			if outBuffer.String() != "" {
				errMsg += fmt.Sprintf("output: %s", outBuffer.String())
			}
			if errMsg == "" {
				return "", err
			}

			return "", errors.New(errMsg)
		}
		return "", err
	}

	return outBuffer.String(), nil
}

func runRubyScript(scriptContent, gemfileContent, inDir string, withEnvs []string) error {
	_, err := runRubyScriptForOutput(scriptContent, gemfileContent, inDir, withEnvs)
	return err
}

// ReCreateProjectUserSchemes ....
func ReCreateProjectUserSchemes(projectPth string) error {
	projectDir := filepath.Dir(projectPth)

	projectBase := filepath.Base(projectPth)
	envs := append(os.Environ(), "project_path="+projectBase, "LC_ALL=en_US.UTF-8")

	return runRubyScript(recreateUserSchemesRubyScriptContent, xcodeprojGemfileContent, projectDir, envs)
}

// ReCreateWorkspaceUserSchemes ...
func ReCreateWorkspaceUserSchemes(workspacePth string) error {
	projects, err := WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return err
	}

	for _, project := range projects {
		if err := ReCreateProjectUserSchemes(project); err != nil {
			return err
		}
	}

	return nil
}

// ProjectTargets ...
func ProjectTargets(projectPth string) (map[string]bool, error) {
	pbxProjPth := filepath.Join(projectPth, "project.pbxproj")
	if exist, err := pathutil.IsPathExists(pbxProjPth); err != nil {
		return map[string]bool{}, err
	} else if !exist {
		return map[string]bool{}, fmt.Errorf("project.pbxproj does not exist at: %s", pbxProjPth)
	}

	content, err := fileutil.ReadStringFromFile(pbxProjPth)
	if err != nil {
		return map[string]bool{}, err
	}

	return pbxprojContentTartgets(content)

}

// WorkspaceTargets ...
func WorkspaceTargets(workspacePth string) (map[string]bool, error) {
	projects, err := WorkspaceProjectReferences(workspacePth)
	if err != nil {
		return nil, err
	}

	targetMap := map[string]bool{}
	for _, project := range projects {
		projectTargetMap, err := ProjectTargets(project)
		if err != nil {
			return map[string]bool{}, err
		}

		for name, hasXCTest := range projectTargetMap {
			targetMap[name] = hasXCTest
		}
	}

	return targetMap, nil
}

// WorkspaceProjectReferences ...
func WorkspaceProjectReferences(workspace string) ([]string, error) {
	projects := []string{}

	workspaceDir := filepath.Dir(workspace)

	xcworkspacedataPth := path.Join(workspace, "contents.xcworkspacedata")
	if exist, err := pathutil.IsPathExists(xcworkspacedataPth); err != nil {
		return []string{}, err
	} else if !exist {
		return []string{}, fmt.Errorf("contents.xcworkspacedata does not exist at: %s", xcworkspacedataPth)
	}

	xcworkspacedataStr, err := fileutil.ReadStringFromFile(xcworkspacedataPth)
	if err != nil {
		return []string{}, err
	}

	xcworkspacedataLines := strings.Split(xcworkspacedataStr, "\n")
	fileRefStart := false
	regexp := regexp.MustCompile(`location = "(.+):(.+).xcodeproj"`)

	for _, line := range xcworkspacedataLines {
		if strings.Contains(line, "<FileRef") {
			fileRefStart = true
			continue
		}

		if fileRefStart {
			fileRefStart = false
			matches := regexp.FindStringSubmatch(line)
			if len(matches) == 3 {
				projectName := matches[2]
				project := filepath.Join(workspaceDir, projectName+".xcodeproj")
				projects = append(projects, project)
			}
		}
	}

	sort.Strings(projects)

	return projects, nil
}

// ------------------------------
// Utility

func filesInDir(dir string) ([]string, error) {
	files := []string{}
	if err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	}); err != nil {
		return []string{}, err
	}
	return files, nil
}

func isUserSchemeFilePath(pth string) bool {
	regexpPattern := filepath.Join(".*[/]?xcuserdata", ".*[.]xcuserdatad", "xcschemes", ".+[.]xcscheme")
	regexp := regexp.MustCompile(regexpPattern)
	return (regexp.FindString(pth) != "")
}

func filterUserSchemeFilePaths(paths []string) []string {
	filteredPaths := []string{}
	for _, pth := range paths {
		if isUserSchemeFilePath(pth) {
			filteredPaths = append(filteredPaths, pth)
		}
	}

	sort.Strings(filteredPaths)

	return filteredPaths
}

func userSchemeFilePaths(projectOrWorkspacePth string) ([]string, error) {
	paths, err := filesInDir(projectOrWorkspacePth)
	if err != nil {
		return []string{}, err
	}
	return filterUserSchemeFilePaths(paths), nil
}

func userSchemes(projectOrWorkspacePth string) (map[string]bool, error) {
	schemePaths, err := userSchemeFilePaths(projectOrWorkspacePth)
	if err != nil {
		return map[string]bool{}, err
	}

	schemeMap := map[string]bool{}
	for _, schemePth := range schemePaths {
		schemeName := SchemeNameFromPath(schemePth)
		hasXCtest, err := SchemeFileContainsXCTestBuildAction(schemePth)
		if err != nil {
			return map[string]bool{}, err
		}
		schemeMap[schemeName] = hasXCtest
	}

	return schemeMap, nil
}

func isSharedSchemeFilePath(pth string) bool {
	regexpPattern := filepath.Join(".*[/]?xcshareddata", "xcschemes", ".+[.]xcscheme")
	regexp := regexp.MustCompile(regexpPattern)
	return (regexp.FindString(pth) != "")
}

func filterSharedSchemeFilePaths(paths []string) []string {
	filteredPaths := []string{}
	for _, pth := range paths {
		if isSharedSchemeFilePath(pth) {
			filteredPaths = append(filteredPaths, pth)
		}
	}

	sort.Strings(filteredPaths)

	return filteredPaths
}

func sharedSchemeFilePaths(projectOrWorkspacePth string) ([]string, error) {
	paths, err := filesInDir(projectOrWorkspacePth)
	if err != nil {
		return []string{}, err
	}
	return filterSharedSchemeFilePaths(paths), nil
}

func sharedSchemes(projectOrWorkspacePth string) (map[string]bool, error) {
	schemePaths, err := sharedSchemeFilePaths(projectOrWorkspacePth)
	if err != nil {
		return map[string]bool{}, err
	}

	schemeMap := map[string]bool{}
	for _, schemePth := range schemePaths {
		schemeName := SchemeNameFromPath(schemePth)
		hasXCTest, err := SchemeFileContainsXCTestBuildAction(schemePth)
		if err != nil {
			return map[string]bool{}, err
		}

		schemeMap[schemeName] = hasXCTest
	}

	return schemeMap, nil
}

func schemeFileContentContainsXCTestBuildAction(schemeFileContent string) (bool, error) {
	testActionStartPattern := "<TestAction"
	testActionEndPattern := "</TestAction>"
	isTestableAction := false

	testableReferenceStartPattern := "<TestableReference"
	testableReferenceSkippedRegexp := regexp.MustCompile(`skipped = "(?P<skipped>.+)"`)
	testableReferenceEndPattern := "</TestableReference>"
	isTestableReference := false

	xctestBuildableReferenceNameRegexp := regexp.MustCompile(`BuildableName = ".+.xctest"`)

	scanner := bufio.NewScanner(strings.NewReader(schemeFileContent))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == testActionEndPattern {
			break
		}

		if strings.TrimSpace(line) == testActionStartPattern {
			isTestableAction = true
			continue
		}

		if !isTestableAction {
			continue
		}

		// TestAction

		if strings.TrimSpace(line) == testableReferenceEndPattern {
			isTestableReference = false
			continue
		}

		if strings.TrimSpace(line) == testableReferenceStartPattern {
			isTestableReference = true
			continue
		}

		if !isTestableReference {
			continue
		}

		// TestableReference

		if matches := testableReferenceSkippedRegexp.FindStringSubmatch(line); len(matches) > 1 {
			skipped := matches[1]
			if skipped != "NO" {
				break
			}
		}

		if match := xctestBuildableReferenceNameRegexp.FindString(line); match != "" {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

// PBXTargetDependency ...
type PBXTargetDependency struct {
	id     string
	isa    string
	target string
}

func parsePBXTargetDependencies(pbxprojContent string) ([]PBXTargetDependency, error) {
	pbxTargetDependencies := []PBXTargetDependency{}

	id := ""
	isa := ""
	target := ""

	beginPBXTargetDependencySectionPattern := `/* Begin PBXTargetDependency section */`
	endPBXTargetDependencySectionPattern := `/* End PBXTargetDependency section */`
	isPBXTargetDependencySection := false

	// BAAFFEEF19EE788800F3AC91 /* PBXTargetDependency */ = {
	beginPBXTargetDependencyRegexp := regexp.MustCompile(`\s*(?P<id>[A-Z0-9]+) /\* (?P<isa>.*) \*/ = {`)
	endPBXTargetDependencyPattern := `};`
	isPBXTargetDependency := false

	// isa = PBXTargetDependency;
	isaRegexp := regexp.MustCompile(`\s*isa = (?P<isa>.*);`)
	// target = BAAFFED019EE788800F3AC91 /* SampleAppWithCocoapods */;
	targetRegexp := regexp.MustCompile(`\s*target = (?P<id>[A-Z0-9]+) /\* (?P<name>.*) \*/;`)

	scanner := bufio.NewScanner(strings.NewReader(pbxprojContent))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == endPBXTargetDependencySectionPattern {
			break
		}

		if strings.TrimSpace(line) == beginPBXTargetDependencySectionPattern {
			isPBXTargetDependencySection = true
			continue
		}

		if !isPBXTargetDependencySection {
			continue
		}

		// PBXTargetDependency section

		if strings.TrimSpace(line) == endPBXTargetDependencyPattern {
			pbxTargetDependency := PBXTargetDependency{
				id:     id,
				isa:    isa,
				target: target,
			}
			pbxTargetDependencies = append(pbxTargetDependencies, pbxTargetDependency)

			id = ""
			isa = ""
			target = ""

			isPBXTargetDependency = false
			continue
		}

		if matches := beginPBXTargetDependencyRegexp.FindStringSubmatch(line); len(matches) == 3 {
			id = matches[1]
			isa = matches[2]

			isPBXTargetDependency = true
			continue
		}

		if !isPBXTargetDependency {
			continue
		}

		// PBXTargetDependency item

		if matches := isaRegexp.FindStringSubmatch(line); len(matches) == 2 {
			isa = strings.Trim(matches[1], `"`)
		}

		if matches := targetRegexp.FindStringSubmatch(line); len(matches) == 3 {
			targetID := strings.Trim(matches[1], `"`)
			// targetName := strings.Trim(matches[2], `"`)

			target = targetID
		}
	}

	return pbxTargetDependencies, nil
}

// PBXNativeTarget ...
type PBXNativeTarget struct {
	id           string
	isa          string
	dependencies []string
	name         string
	productPath  string
	productType  string
}

func parsePBXNativeTargets(pbxprojContent string) ([]PBXNativeTarget, error) {
	pbxNativeTargets := []PBXNativeTarget{}

	id := ""
	isa := ""
	dependencies := []string{}
	name := ""
	productPath := ""
	productType := ""

	beginPBXNativeTargetSectionPattern := `/* Begin PBXNativeTarget section */`
	endPBXNativeTargetSectionPattern := `/* End PBXNativeTarget section */`
	isPBXNativeTargetSection := false

	// BAAFFED019EE788800F3AC91 /* SampleAppWithCocoapods */ = {
	beginPBXNativeTargetRegexp := regexp.MustCompile(`\s*(?P<id>[A-Z0-9]+) /\* (?P<name>.*) \*/ = {`)
	endPBXNativeTargetPattern := `};`
	isPBXNativeTarget := false

	// isa = PBXNativeTarget;
	isaRegexp := regexp.MustCompile(`\s*isa = (?P<isa>.*);`)

	beginDependenciesPattern := `dependencies = (`
	dependencieRegexp := regexp.MustCompile(`\s*(?P<id>[A-Z0-9]+) /\* (?P<isa>.*) \*/,`)
	endDependenciesPattern := `);`
	isDependencies := false

	// name = SampleAppWithCocoapods;
	nameRegexp := regexp.MustCompile(`\s*name = (?P<name>.*);`)
	// productReference = BAAFFEED19EE788800F3AC91 /* SampleAppWithCocoapodsTests.xctest */;
	productReferenceRegexp := regexp.MustCompile(`\s*productReference = (?P<id>[A-Z0-9]+) /\* (?P<path>.*) \*/;`)
	// productType = "com.apple.product-type.bundle.unit-test";
	productTypeRegexp := regexp.MustCompile(`\s*productType = (?P<productType>.*);`)

	scanner := bufio.NewScanner(strings.NewReader(pbxprojContent))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == endPBXNativeTargetSectionPattern {
			break
		}

		if strings.TrimSpace(line) == beginPBXNativeTargetSectionPattern {
			isPBXNativeTargetSection = true
			continue
		}

		if !isPBXNativeTargetSection {
			continue
		}

		// PBXNativeTarget section

		if strings.TrimSpace(line) == endPBXNativeTargetPattern {
			pbxNativeTarget := PBXNativeTarget{
				id:           id,
				isa:          isa,
				dependencies: dependencies,
				name:         name,
				productPath:  productPath,
				productType:  productType,
			}
			pbxNativeTargets = append(pbxNativeTargets, pbxNativeTarget)

			id = ""
			isa = ""
			name = ""
			productPath = ""
			productType = ""
			dependencies = []string{}

			isPBXNativeTarget = false
			continue
		}

		if matches := beginPBXNativeTargetRegexp.FindStringSubmatch(line); len(matches) == 3 {
			id = matches[1]
			name = matches[2]

			isPBXNativeTarget = true
			continue
		}

		if !isPBXNativeTarget {
			continue
		}

		// PBXNativeTarget item

		if matches := isaRegexp.FindStringSubmatch(line); len(matches) == 2 {
			isa = strings.Trim(matches[1], `"`)
		}

		if matches := nameRegexp.FindStringSubmatch(line); len(matches) == 2 {
			name = strings.Trim(matches[1], `"`)
		}

		if matches := productTypeRegexp.FindStringSubmatch(line); len(matches) == 2 {
			productType = strings.Trim(matches[1], `"`)
		}

		if matches := productReferenceRegexp.FindStringSubmatch(line); len(matches) == 3 {
			// productId := strings.Trim(matches[1], `"`)
			productPath = strings.Trim(matches[2], `"`)
		}

		if isDependencies && strings.TrimSpace(line) == endDependenciesPattern {
			isDependencies = false
			continue
		}

		if strings.TrimSpace(line) == beginDependenciesPattern {
			isDependencies = true
			continue
		}

		if !isDependencies {
			continue
		}

		// dependencies
		if matches := dependencieRegexp.FindStringSubmatch(line); len(matches) == 3 {
			dependencieID := strings.Trim(matches[1], `"`)
			dependencieIsa := strings.Trim(matches[2], `"`)

			if dependencieIsa == "PBXTargetDependency" {
				dependencies = append(dependencies, dependencieID)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return []PBXNativeTarget{}, err
	}

	return pbxNativeTargets, nil
}

func targetDependencieWithID(dependencies []PBXTargetDependency, id string) (PBXTargetDependency, bool) {
	for _, dependencie := range dependencies {
		if dependencie.id == id {
			return dependencie, true
		}
	}
	return PBXTargetDependency{}, false
}

func targetWithID(targets []PBXNativeTarget, id string) (PBXNativeTarget, bool) {
	for _, target := range targets {
		if target.id == id {
			return target, true
		}
	}
	return PBXNativeTarget{}, false
}

func pbxprojContentTartgets(pbxprojContent string) (map[string]bool, error) {
	targetMap := map[string]bool{}

	targets, err := parsePBXNativeTargets(pbxprojContent)
	if err != nil {
		return map[string]bool{}, err
	}

	targetDependencies, err := parsePBXTargetDependencies(pbxprojContent)
	if err != nil {
		return map[string]bool{}, err
	}

	// Add targets which has test targets
	for _, target := range targets {
		if path.Ext(target.productPath) == ".xctest" {
			if len(target.dependencies) > 0 {
				for _, dependencieID := range target.dependencies {
					dependency, found := targetDependencieWithID(targetDependencies, dependencieID)
					if found {
						dependentTarget, found := targetWithID(targets, dependency.target)
						if found {
							targetMap[dependentTarget.name] = true
						}
					}
				}
			}
		}
	}

	// Add targets which has NO test targets
	for _, target := range targets {
		if path.Ext(target.productPath) != ".xctest" {
			_, found := targetMap[target.name]
			if !found {
				targetMap[target.name] = false
			}
		}
	}

	return targetMap, nil
}
