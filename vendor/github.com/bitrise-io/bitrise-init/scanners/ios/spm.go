package ios

import (
	"fmt"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/pathfilters"
)

func HasSPMDependencies(fileList []string) (bool, error) {
	// Pure SPM projects: find project-level `Package.swift` file
	pureSwiftFilters := []pathutil.FilterFunc{
		pathutil.BaseFilter("Package.swift", true), // match nested project folders too
		pathfilters.ForbidPodsDirComponentFilter,   // don't match dependency source checkouts
		pathfilters.ForbidCarthageDirComponentFilter,
		pathfilters.ForbidNodeModulesComponentFilter,
	}
	matches, err := pathutil.FilterPaths(fileList, pureSwiftFilters...)
	if err != nil {
		return false, fmt.Errorf("couldn't detect SPM dependencies: %w", err)
	}

	if len(matches) > 0 {
		return true, nil
	}

	// Xcode projects: find lockfile inside `xcodeproj`
	xcodeFilters := []pathutil.FilterFunc{
		pathutil.BaseFilter("Package.resolved", true), // match nested project folders too
		pathfilters.ForbidPodsDirComponentFilter,      // don't match dependency source checkouts
		pathfilters.ForbidCarthageDirComponentFilter,
		pathfilters.ForbidNodeModulesComponentFilter,
	}
	matches, err = pathutil.FilterPaths(fileList, xcodeFilters...)
	if err != nil {
		return false, fmt.Errorf("couldn't detect SPM dependencies: %w", err)
	}
	return len(matches) > 0, nil
}
