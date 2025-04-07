package gradle

import (
	"io"
	"os"
	"sort"
	"strings"

	"github.com/bitrise-io/bitrise-init/direntry"
	"github.com/bitrise-io/go-utils/log"
)

/*
Relevant Gradle project files:

Gradle wrapper scripts (gradlew and gradlew.bat):
	The presence of the gradlew and gradlew.bat files in the root directory of a project is a clear indicator that Gradle is used.

Settings File (settings.gradle[.kts]): The settings file is the entry point of every Gradle project.
	The primary purpose of the settings file is to add subprojects to your build.
	Gradle supports single and multi-project builds.
	- For single-project builds, the settings file is optional.
	- For multi-project builds, the settings file is mandatory and declares all subprojects.
*/

type SubProject struct {
	Name                 string
	BuildScriptFileEntry direntry.DirEntry
}

type Project struct {
	RootDirEntry            direntry.DirEntry
	GradlewFileEntry        direntry.DirEntry
	ConfigDirEntry          *direntry.DirEntry
	VersionCatalogFileEntry *direntry.DirEntry
	SettingsGradleFileEntry *direntry.DirEntry

	IncludedProjects          []SubProject
	AllBuildScriptFileEntries []direntry.DirEntry
}

func ScanProject(searchDir string) (*Project, error) {
	rootEntry, err := direntry.ListEntries(searchDir, 4)
	if err != nil {
		return nil, err
	}

	projectRoot, err := detectGradleProjectRoot(*rootEntry)
	if err != nil {
		return nil, err
	}
	if projectRoot == nil {
		return nil, nil
	}
	projects, err := detectIncludedProjects(*projectRoot)
	if err != nil {
		return nil, err
	}

	project := Project{
		RootDirEntry:            projectRoot.rootDirEntry,
		GradlewFileEntry:        projectRoot.gradlewFileEntry,
		ConfigDirEntry:          projectRoot.configDirEntry,
		VersionCatalogFileEntry: projectRoot.versionCatalogFileEntry,
		SettingsGradleFileEntry: projectRoot.settingsGradleFileEntry,

		IncludedProjects:          projects.includedProjects,
		AllBuildScriptFileEntries: projects.allBuildScriptEntries,
	}

	return &project, nil
}

func (proj Project) DetectAnyDependencies(dependencies []string) (bool, error) {
	detected, err := proj.detectAnyDependenciesInVersionCatalogFile(dependencies)
	if err != nil {
		return false, err
	}
	if detected {
		return true, nil
	}

	detected, err = proj.detectAnyDependenciesInIncludedProjectBuildScripts(dependencies)
	if err != nil {
		return false, err
	}
	if detected {
		return true, nil
	}

	return proj.detectAnyDependenciesInBuildScripts(dependencies)
}

func (proj Project) detectAnyDependenciesInVersionCatalogFile(dependencies []string) (bool, error) {
	if proj.VersionCatalogFileEntry == nil {
		return false, nil
	}
	return proj.detectAnyDependencies(proj.VersionCatalogFileEntry.AbsPath, dependencies)
}

func (proj Project) detectAnyDependenciesInIncludedProjectBuildScripts(dependencies []string) (bool, error) {
	for _, includedProject := range proj.IncludedProjects {
		detected, err := proj.detectAnyDependencies(includedProject.BuildScriptFileEntry.AbsPath, dependencies)
		if err != nil {
			return false, err
		}
		if detected {
			return true, nil
		}
	}
	return false, nil
}

func (proj Project) detectAnyDependenciesInBuildScripts(dependencies []string) (bool, error) {
	for _, BuildScriptFileEntry := range proj.AllBuildScriptFileEntries {
		detected, err := proj.detectAnyDependencies(BuildScriptFileEntry.AbsPath, dependencies)
		if err != nil {
			return false, err
		}
		if detected {
			return true, nil
		}
	}
	return false, nil
}

func (proj Project) detectAnyDependencies(pth string, dependencies []string) (bool, error) {
	file, err := os.Open(pth)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.TWarnf("Unable to close file %s: %s", pth, err)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return false, err
	}

	for _, dependency := range dependencies {
		if strings.Contains(string(content), dependency) {
			return true, nil
		}
	}

	return false, nil
}

type gradleProjectRootEntry struct {
	rootDirEntry            direntry.DirEntry
	gradlewFileEntry        direntry.DirEntry
	configDirEntry          *direntry.DirEntry
	versionCatalogFileEntry *direntry.DirEntry
	settingsGradleFileEntry *direntry.DirEntry
}

func detectGradleProjectRoot(rootEntry direntry.DirEntry) (*gradleProjectRootEntry, error) {
	gradlewFileEntry := rootEntry.FindEntryByName("gradlew", false)
	if gradlewFileEntry == nil {
		if len(rootEntry.Entries) == 0 {
			return nil, nil
		}

		for _, entry := range rootEntry.Entries {
			if entry.IsDir {
				return detectGradleProjectRoot(entry)
			}
		}
		return nil, nil
	}

	projectRoot := gradleProjectRootEntry{
		rootDirEntry:     rootEntry,
		gradlewFileEntry: *gradlewFileEntry,
	}

	configDirEntry := rootEntry.FindEntryByName("gradle", true)
	if configDirEntry != nil {
		projectRoot.configDirEntry = configDirEntry

		versionCatalogFileEntry := configDirEntry.FindEntryByName("libs.versions.toml", false)
		if versionCatalogFileEntry != nil {
			projectRoot.versionCatalogFileEntry = versionCatalogFileEntry
		}
	}

	settingsFileEntry := rootEntry.FindEntryByName("settings.gradle", false)
	if settingsFileEntry == nil {
		settingsFileEntry = rootEntry.FindEntryByName("settings.gradle.kts", false)
	}
	if settingsFileEntry != nil {
		projectRoot.settingsGradleFileEntry = settingsFileEntry
	}

	return &projectRoot, nil
}

type includedProjects struct {
	allBuildScriptEntries []direntry.DirEntry
	includedProjects      []SubProject
}

func detectIncludedProjects(projectRootEntry gradleProjectRootEntry) (*includedProjects, error) {
	projects := includedProjects{}
	projects.allBuildScriptEntries = projectRootEntry.rootDirEntry.FindAllEntriesByName("build.gradle", false)
	projects.allBuildScriptEntries = append(projects.allBuildScriptEntries, projectRootEntry.rootDirEntry.FindAllEntriesByName("build.gradle.kts", false)...)
	sort.Slice(projects.allBuildScriptEntries, func(i, j int) bool {
		if len(projects.allBuildScriptEntries[i].AbsPath) == len(projects.allBuildScriptEntries[j].AbsPath) {
			return projects.allBuildScriptEntries[i].AbsPath < projects.allBuildScriptEntries[j].AbsPath
		}
		return len(projects.allBuildScriptEntries[i].AbsPath) < len(projects.allBuildScriptEntries[j].AbsPath)
	})

	if projectRootEntry.settingsGradleFileEntry != nil {
		var subprojects []SubProject

		includes, err := detectProjectIncludes(*projectRootEntry.settingsGradleFileEntry)
		if err != nil {
			return nil, err
		}

		for _, include := range includes {
			var components []string

			trimmedInclude := strings.TrimPrefix(include, ":")
			includeComponents := strings.Split(trimmedInclude, ":")
			for _, includeComponent := range includeComponents {
				if includeComponent == "" {
					continue
				}
				includeComponent = strings.TrimSpace(includeComponent)
				components = append(components, includeComponent)
			}

			projectBuildScript := projectRootEntry.rootDirEntry.FindEntryByPath(false, append(components, "build.gradle")...)
			if projectBuildScript != nil {
				subprojects = append(subprojects, SubProject{
					Name:                 include,
					BuildScriptFileEntry: *projectBuildScript,
				})
				continue
			}

			projectBuildScript = projectRootEntry.rootDirEntry.FindEntryByPath(false, append(components, "build.gradle.kts")...)
			if projectBuildScript != nil {
				subprojects = append(subprojects, SubProject{
					Name:                 include,
					BuildScriptFileEntry: *projectBuildScript,
				})
			} else {
				log.TWarnf("Unable to find build script for %s", include)
			}

			projects.includedProjects = subprojects
		}
	}

	return &projects, nil
}

func detectProjectIncludes(settingGradleFile direntry.DirEntry) ([]string, error) {
	file, err := os.Open(settingGradleFile.AbsPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.TWarnf("Unable to close file %s: %s", settingGradleFile.AbsPath, err)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return detectProjectIncludesInContent(string(content)), nil
}

func detectProjectIncludesInContent(settingGradleFileContent string) []string {
	var projects []string
	lines := strings.Split(settingGradleFileContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "include(") && !strings.HasPrefix(line, "include ") {
			continue
		}

		includedModules := strings.TrimPrefix(line, "include")
		includedModules = strings.Trim(includedModules, "()")
		includedModulesSplit := strings.Split(includedModules, ",")

		for _, includedModule := range includedModulesSplit {
			includedModule = strings.TrimSpace(includedModule)
			includedModule = strings.Trim(includedModule, `"'`)
			if !strings.HasPrefix(includedModule, ":") {
				includedModule = ":" + includedModule
			}
			projects = append(projects, includedModule)
		}
	}
	sort.Slice(projects, func(i, j int) bool {
		if len(projects[i]) == len(projects[j]) {
			return projects[i] < projects[j]
		}
		return len(projects[i]) < len(projects[j])
	})

	return projects
}
