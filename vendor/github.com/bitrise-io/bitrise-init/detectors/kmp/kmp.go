package kmp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise-init/detectors/gradle"
	"github.com/bitrise-io/bitrise-init/scanners/android"
	"github.com/bitrise-io/bitrise-init/scanners/ios"
	"github.com/bitrise-io/go-utils/log"
)

type Project struct {
	GradleProject          gradle.Project
	IOSAppDetectResult     *ios.DetectResult
	AndroidAppDetectResult *android.DetectResult
}

func ScanProject(gradleProject gradle.Project) (*Project, error) {
	log.TInfof("Searching for Kotlin Multiplatform dependencies...")
	kotlinMultiplatformDetected, err := gradleProject.DetectAnyDependencies([]string{
		"org.jetbrains.kotlin.multiplatform",
		`kotlin("multiplatform")`,
	})
	if err != nil {
		return nil, err
	}

	log.TDonef("Kotlin Multiplatform dependencies found: %v", kotlinMultiplatformDetected)
	if !kotlinMultiplatformDetected {
		return nil, nil
	}

	log.TInfof("Scanning Kotlin Multiplatform targets...")
	iosAppDetectResult, err := scanIOSAppProject(gradleProject)
	if err != nil {
		return nil, fmt.Errorf("failed to scan IOS project: %w", err)
	}

	androidAppDetectResult, err := scanAndroidAppProject(gradleProject)
	if err != nil {
		return nil, fmt.Errorf("failed to scan Android project: %w", err)
	}

	return &Project{
		GradleProject:          gradleProject,
		IOSAppDetectResult:     iosAppDetectResult,
		AndroidAppDetectResult: androidAppDetectResult,
	}, nil
}

func scanIOSAppProject(gradleProject gradle.Project) (*ios.DetectResult, error) {
	xcodeProjectFile := gradleProject.RootDirEntry.FindFirstFileEntryByExtension(".xcodeproj")
	if xcodeProjectFile == nil {
		return nil, nil
	}

	iosScanner := ios.NewScanner()
	detected, err := iosScanner.DetectPlatform(filepath.Dir(xcodeProjectFile.AbsPath))
	if err != nil {
		log.TWarnf("Failed to detect iOS project: %s", err)
	}
	if detected && len(iosScanner.DetectResult.Projects) > 0 {
		result := iosScanner.DetectResult
		if len(result.Projects) > 1 {
			log.TWarnf("%d iOS projects found in the Gradle project, using the first one: %s", len(result.Projects), result.Projects[0].RelPath)
		}
		result.Projects = result.Projects[:1]
		iosAppProjectRelPath := result.Projects[0].RelPath
		iosAppProjectRelPath = filepath.Join(filepath.Dir(xcodeProjectFile.RelPath), iosAppProjectRelPath)
		if !strings.HasPrefix(iosAppProjectRelPath, "./") {
			iosAppProjectRelPath = "./" + iosAppProjectRelPath
		}
		project := result.Projects[0]
		project.RelPath = iosAppProjectRelPath
		result.Projects[0] = project
		return &result, nil
	}
	return nil, nil
}

func scanAndroidAppProject(gradleProject gradle.Project) (*android.DetectResult, error) {
	androidApplicationPluginID, err := gradleProject.GetDependencyID(`com.android.application`)
	if err != nil {
		return nil, fmt.Errorf("failed to get Android application plugin ID: %w", err)
	}

	androidAppDependencies := []string{
		`"com.android.application"`,
	}
	if androidApplicationPluginID != "" {
		// alias(libs.plugins.androidApplication)
		// TODO?: make it more reliable by using a regex/glob matching (alias.*<androidApplicationPluginID>)
		androidAppDependencies = append(androidAppDependencies, fmt.Sprintf("alias(libs.plugins.%s)", androidApplicationPluginID))
	}

	androidProjects, err := gradleProject.FindSubProjectsWithAnyDependencies(androidAppDependencies)
	if err != nil {
		return nil, err
	}

	// Wear projects Manifest files contains this: <uses-feature android:name="android.hardware.type.watch" />
	var androidAppProjects []gradle.SubProject
	if len(androidProjects) > 0 {
		for _, androidProject := range androidProjects {
			androidProjectDir := androidProject.BuildScriptFileEntry.Parent()
			manifestFiles := androidProjectDir.FindAllEntriesByName("AndroidManifest.xml", false)
			isWearApp := false
			if len(manifestFiles) > 0 {
				for _, manifestFile := range manifestFiles {
					manifestContent, err := os.ReadFile(manifestFile.AbsPath)
					if err != nil {
						return nil, fmt.Errorf("failed to read AndroidManifest.xml file: %w", err)
					}
					if strings.Contains(string(manifestContent), "android.hardware.type.watch") {
						isWearApp = true
						break
					}
				}
			}

			if isWearApp {
				continue
			}

			androidAppProjects = append(androidAppProjects, androidProject)
		}
	}

	if len(androidAppProjects) > 0 {
		androidAppProject := &androidAppProjects[0]
		androidAppProjectDir := filepath.Dir(androidAppProject.BuildScriptFileEntry.RelPath)
		if len(androidAppProjects) > 1 {
			log.TWarnf("%d Android targets found in the Gradle project, using the first one: %s", len(androidAppProjects), androidAppProjectDir)
		}

		return &android.DetectResult{
			GradleProject: gradleProject,
			Modules: []android.GradleModule{{
				ModulePath:     androidAppProjectDir,
				BuildScriptPth: androidAppProject.BuildScriptFileEntry.RelPath,
				UsesKotlinDSL:  strings.HasSuffix(androidAppProject.BuildScriptFileEntry.RelPath, ".kts"),
			}},
			Icons: nil,
		}, nil
	}

	return nil, nil
}
