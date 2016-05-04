package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOptionModel(t *testing.T) {
	actual := NewOptionModel("project_path", "Project (or Workspace) path", "BITRISE_PROJECT_PATH")
	expected := OptionModel{
		Key:      "project_path",
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

func TestAddValueMapItems(t *testing.T) {
	t.Log("Without nested options")
	{
		actual := NewEmptyOptionModel()
		actual.AddValueMapItems("assembleAndroidTest")
		actual.AddValueMapItems("assembleDebug")
		actual.AddValueMapItems("assembleRelease")

		expected := NewEmptyOptionModel()
		expected.ValueMap = OptionValueMap{
			"assembleAndroidTest": nil,
			"assembleDebug":       nil,
			"assembleRelease":     nil,
		}

		require.Equal(t, expected, actual)
	}

	t.Log("With nested options")
	{
		actual := NewEmptyOptionModel()
		actual.AddValueMapItems("assembleAndroidTest", OptionModel{ValueMap: OptionValueMap{"bitrise.json": nil}})

		expected := NewEmptyOptionModel()
		expected.ValueMap = OptionValueMap{
			"assembleAndroidTest": []OptionModel{
				OptionModel{
					ValueMap: OptionValueMap{
						"bitrise.json": nil,
					},
				},
			},
		}

		require.Equal(t, expected, actual)
	}
}

func TestGetValues(t *testing.T) {
	option := NewEmptyOptionModel()
	option.AddValueMapItems("assembleAndroidTest")
	option.AddValueMapItems("assembleDebug")
	option.AddValueMapItems("assembleRelease")

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
