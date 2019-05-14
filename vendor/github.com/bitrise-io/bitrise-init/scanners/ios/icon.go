package ios

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/xcode-project/xcodeproj"
)

// lookupIconBySchemeName returns possible ios app icons for a scheme.
func lookupIconBySchemeName(projectPath string, schemeName string, basepath string) (models.Icons, error) {
	project, err := xcodeproj.Open(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open project file: %s, error: %s", projectPath, err)
	}

	scheme, found := project.Scheme(schemeName)
	if !found {
		return nil, fmt.Errorf("scheme (%s) not found in project", schemeName)
	}

	mainTarget, err := mainTargetOfScheme(project, scheme.Name)

	return lookupIconByTarget(projectPath, mainTarget, basepath)
}

// lookupIconByTargetName returns possible ios app icons for a target.
func lookupIconByTargetName(projectPath string, targetName string, basepath string) (models.Icons, error) {
	target, err := nameToTarget(projectPath, targetName)
	if err != nil {
		return nil, nil
	}

	return lookupIconByTarget(projectPath, target, basepath)
}

func nameToTarget(projectPath string, targetName string) (xcodeproj.Target, error) {
	project, err := xcodeproj.Open(projectPath)
	if err != nil {
		return xcodeproj.Target{}, fmt.Errorf("failed to open project file: %s, error: %s", projectPath, err)
	}

	target, found, err := targetByName(project, targetName)
	if err != nil {
		return xcodeproj.Target{}, err
	} else if !found {
		return xcodeproj.Target{}, fmt.Errorf("not found target: %s, in project: %s", targetName, projectPath)
	}
	return target, nil
}

func lookupIconByTarget(projectPath string, target xcodeproj.Target, basepath string) (models.Icons, error) {
	targetToAppIconSetPaths, err := xcodeproj.AppIconSetPaths(projectPath)
	appIconSetPaths, ok := targetToAppIconSetPaths[target.ID]
	if !ok {
		return nil, nil
	}

	iconPaths := []string{}
	for _, appIconSetPath := range appIconSetPaths {
		icon, found, err := parseResourceSet(appIconSetPath)
		if err != nil {
			return nil, fmt.Errorf("could not get icon, error: %s", err)
		} else if !found {
			return nil, nil
		}
		log.Debugf("App icons: %s", icon)

		iconPath := filepath.Join(appIconSetPath, icon.Filename)

		if _, err := os.Stat(iconPath); err != nil && os.IsNotExist(err) {
			return nil, fmt.Errorf("icon file does not exist: %s, error: %err", iconPath, err)
		}
		iconPaths = append(iconPaths, iconPath)
	}

	icons, err := utility.CreateIconDescriptors(iconPaths, basepath)
	if err != nil {
		return nil, err
	}
	return icons, nil
}

func mainTargetOfScheme(proj xcodeproj.XcodeProj, scheme string) (xcodeproj.Target, error) {
	projTargets := proj.Proj.Targets
	sch, ok := proj.Scheme(scheme)
	if !ok {
		return xcodeproj.Target{}, fmt.Errorf("Failed to find scheme (%s) in project", scheme)
	}

	var blueIdent string
	for _, entry := range sch.BuildAction.BuildActionEntries {
		if entry.BuildableReference.IsAppReference() {
			blueIdent = entry.BuildableReference.BlueprintIdentifier
			break
		}
	}

	// Search for the main target
	for _, t := range projTargets {
		if t.ID == blueIdent {
			return t, nil

		}
	}
	return xcodeproj.Target{}, fmt.Errorf("failed to find the project's main target for scheme (%s)", scheme)
}

func targetByName(proj xcodeproj.XcodeProj, target string) (xcodeproj.Target, bool, error) {
	projTargets := proj.Proj.Targets
	for _, t := range projTargets {
		if t.Name == target {
			return t, true, nil
		}
	}
	return xcodeproj.Target{}, false, nil
}
