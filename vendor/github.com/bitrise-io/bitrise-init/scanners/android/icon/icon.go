package icon

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/beevik/etree"
	"github.com/bitrise-io/bitrise-init/models"
	"github.com/bitrise-io/bitrise-init/utility"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

type icon struct {
	prefix       string
	fileNameBase string
}

func lookupIcon(manifestPth, resPth string) (string, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(manifestPth); err != nil {
		return "", err
	}

	log.Debugf("Looking for app icons. Manifest path: %s", manifestPth)
	icon, err := parseIconName(doc)
	if err != nil {
		return "", err
	}

	var resourceSuffixes = [...]string{"xxxhdpi", "xxhdpi", "xhdpi", "hdpi", "mdpi", "ldpi"}
	resourceDirs := make([]string, len(resourceSuffixes))
	for _, mipmapSuffix := range resourceSuffixes {
		resourceDirs = append(resourceDirs, icon.prefix+"-"+mipmapSuffix)
	}

	for _, dir := range resourceDirs {
		filePath := path.Join(resPth, dir, icon.fileNameBase+".png")
		if exists, err := pathutil.IsPathExists(filePath); err != nil {
			return "", err
		} else if exists {
			return filePath, nil
		}
	}
	return "", nil
}

// parseIconName fetches icon name from AndroidManifest.xml
func parseIconName(doc *etree.Document) (icon, error) {
	man := doc.SelectElement("manifest")
	if man == nil {
		log.Debugf("Key manifest not found in manifest file")
		return icon{}, nil
	}
	app := man.SelectElement("application")
	if app == nil {
		log.Debugf("Key application not found in manifest file")
		return icon{}, nil
	}
	ic := app.SelectAttr("android:icon")
	if ic == nil {
		log.Debugf("Attribute not found in manifest file")
		return icon{}, nil
	}

	iconPathParts := strings.Split(strings.TrimPrefix(ic.Value, "@"), "/")
	if len(iconPathParts) != 2 {
		return icon{}, fmt.Errorf("unsupported icon key")
	}
	return icon{
		prefix:       iconPathParts[0],
		fileNameBase: iconPathParts[1],
	}, nil
}

func lookupPossibleMatches(projectDir string, basepath string) ([]string, error) {
	manifestPaths, err := filepath.Glob(filepath.Join(regexp.QuoteMeta(projectDir), "*", "src", "*", "AndroidManifest.xml"))
	if err != nil {
		return nil, err
	}

	var iconPaths []string
	for _, manifestPath := range manifestPaths {
		resourcesPath, err := filepath.Abs(filepath.Join(manifestPath, "..", "res"))
		if err != nil {
			return nil, err
		}
		if exist, err := pathutil.IsPathExists(resourcesPath); err != nil {
			return nil, err
		} else if !exist {
			log.Debugf("Resource path %s does not exist.", resourcesPath)
		}

		iconPath, err := lookupIcon(manifestPath, resourcesPath)
		if err != nil {
			return nil, err
		}
		if iconPath != "" {
			iconPaths = append(iconPaths, iconPath)
		}
	}
	return iconPaths, nil
}

// LookupPossibleMatches returns the largest resolution for all potential android icons
// It does look up all possible files project_dir/*/src/*/AndroidManifest.xml,
// then looks up the icon referenced in the res directory
func LookupPossibleMatches(projectDir string, basepath string) (models.Icons, error) {
	iconPaths, err := lookupPossibleMatches(projectDir, basepath)
	if err != nil {
		return nil, err
	}

	icons, err := utility.ConvertPathsToUniqueFileNames(iconPaths, basepath)
	if err != nil {
		return nil, err
	}
	return icons, nil
}
