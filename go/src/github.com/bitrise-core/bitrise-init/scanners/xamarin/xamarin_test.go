package xamarin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetProjectTestType(t *testing.T) {
	t.Log("Xamarin.UITest")
	{
		testType := projectTestType(`Include="Xamarin.UITest`)
		require.Equal(t, xamarinUITestType, testType)
	}

	t.Log("Xamarin.UITest")
	{
		testType := projectTestType(`include="xamarin.uitest`)
		require.Equal(t, xamarinUITestType, testType)
	}

	t.Log("NUnit test")
	{
		testType := projectTestType(`Include="nunit.framework`)
		require.Equal(t, nunitTestType, testType)
	}

	t.Log("NUnit test")
	{
		testType := projectTestType(`include="nunit.framework`)
		require.Equal(t, nunitTestType, testType)
	}

	t.Log("NUnitLite test")
	{
		testType := projectTestType(`<Reference Include="Xamarin.Android.NUnitLite" />`)
		require.Equal(t, nunitLiteTestType, testType)
	}

	t.Log("NUnitLite test")
	{
		testType := projectTestType(`<Reference Include="monotouch.nunitlite" />`)
		require.Equal(t, nunitLiteTestType, testType)
	}
}

func TestProjectType(t *testing.T) {
	t.Log("Xamarin.iOS")
	{
		projectType := projectType([]string{"E613F3A2-FE9C-494F-B74E-F63BCB86FEA6"})
		require.Equal(t, "Xamarin.iOS", projectType)
	}

	t.Log("Xamarin.Android")
	{
		projectType := projectType([]string{"10368E6C-D01B-4462-8E8B-01FC667A7035"})
		require.Equal(t, "Xamarin.Android", projectType)
	}

	t.Log("MonoMac")
	{
		projectType := projectType([]string{"1C533B1C-72DD-4CB1-9F6B-BF11D93BCFBE"})
		require.Equal(t, "MonoMac", projectType)
	}

	t.Log("Xamarin.Mac")
	{
		projectType := projectType([]string{"A3F8F2AB-B479-4A4A-A458-A89E7DC349F1"})
		require.Equal(t, "Xamarin.Mac", projectType)
	}

	t.Log("Xamarin.tvOS")
	{
		projectType := projectType([]string{"06FA79CB-D6CD-4721-BB4B-1BD202089C55"})
		require.Equal(t, "Xamarin.tvOS", projectType)
	}

	t.Log("Xamarin.iOS & Xamarin.Android - finds the first type")
	{
		projectType := projectType([]string{"E613F3A2-FE9C-494F-B74E-F63BCB86FEA6", "EFBA0AD7-5A72-4C68-AF49-83D382785DCF"})
		require.Equal(t, "Xamarin.iOS", projectType)
	}
}

func TestFilterSolutionFiles(t *testing.T) {
	t.Log(`Contains solution files`)
	{
		fileList := []string{
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-xamarin-ios/CreditCardValidator.iOS.sln",
			"/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-android/sln",
			"path/to/my/gradlew/file",
			"path/to/my",
		}

		files, err := filterSolutionFiles(fileList)
		require.NoError(t, err)
		require.Equal(t, 1, len(files))

		// Also sorts solution files by path components length
		require.Equal(t, "/Users/bitrise/Develop/bitrise/sample-apps/sample-apps-xamarin-ios/CreditCardValidator.iOS.sln", files[0])
	}

	t.Log(`Do not contains solution file`)
	{
		fileList := []string{
			"path/to/my/gradlew/build.",
			"path/to/my/gradle",
		}

		files, err := filterSolutionFiles(fileList)
		require.NoError(t, err)
		require.Equal(t, 0, len(files))
	}
}

func TestXamarinConfigName(t *testing.T) {
	require.Equal(t, "xamarin-config", configName(false, false))
	require.Equal(t, "xamarin-nuget-config", configName(true, false))
	require.Equal(t, "xamarin-components-config", configName(false, true))
	require.Equal(t, "xamarin-nuget-components-config", configName(true, true))
}
