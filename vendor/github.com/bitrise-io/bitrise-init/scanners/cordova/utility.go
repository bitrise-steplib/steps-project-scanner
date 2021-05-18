package cordova

import (
	"encoding/xml"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

const configXMLBasePath = "config.xml"

// WidgetModel ...
type WidgetModel struct {
	XMLNSCDV string `xml:"xmlns cdv,attr"`
}

func parseConfigXMLContent(content string) (WidgetModel, error) {
	widget := WidgetModel{}
	if err := xml.Unmarshal([]byte(content), &widget); err != nil {
		return WidgetModel{}, err
	}
	return widget, nil
}

// ParseConfigXML ...
func ParseConfigXML(pth string) (WidgetModel, error) {
	content, err := fileutil.ReadStringFromFile(pth)
	if err != nil {
		return WidgetModel{}, err
	}
	return parseConfigXMLContent(content)
}

// FilterRootConfigXMLFile ...
func FilterRootConfigXMLFile(fileList []string) (string, error) {
	allowConfigXMLBaseFilter := pathutil.BaseFilter(configXMLBasePath, true)
	configXMLs, err := pathutil.FilterPaths(fileList, allowConfigXMLBaseFilter)
	if err != nil {
		return "", err
	}

	if len(configXMLs) == 0 {
		return "", nil
	}

	return configXMLs[0], nil
}
