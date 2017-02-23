package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOptionModel(t *testing.T) {
	actual := NewOptionModel("Project (or Workspace) path", "BITRISE_PROJECT_PATH")
	expected := OptionModel{
		Title:    "Project (or Workspace) path",
		EnvKey:   "BITRISE_PROJECT_PATH",
		ValueMap: OptionValueMap{},
	}

	require.Equal(t, expected, actual)
}

func TestNewEmptyOptionModel(t *testing.T) {
	actual := NewEmptyOptionModel()
	expected := OptionModel{
		ValueMap: OptionValueMap{},
	}

	require.Equal(t, expected, actual)
}

func TestGetValues(t *testing.T) {
	option := NewEmptyOptionModel()
	option.ValueMap["assembleAndroidTest"] = OptionModel{}
	option.ValueMap["assembleDebug"] = OptionModel{}
	option.ValueMap["assembleRelease"] = OptionModel{}

	values := option.GetValues()

	expectedMap := map[string]bool{
		"assembleAndroidTest": false,
		"assembleDebug":       false,
		"assembleRelease":     false,
	}

	for _, value := range values {
		delete(expectedMap, value)
	}

	require.Equal(t, 0, len(expectedMap))
}
