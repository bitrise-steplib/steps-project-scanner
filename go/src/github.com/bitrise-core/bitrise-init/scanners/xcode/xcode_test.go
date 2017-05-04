package xcode

import (
	"testing"

	"github.com/bitrise-core/bitrise-init/utility"
	"github.com/stretchr/testify/require"
)

func TestNewConfigDescriptor(t *testing.T) {
	descriptor := NewConfigDescriptor(false, "", false, true)
	require.Equal(t, false, descriptor.HasPodfile)
	require.Equal(t, false, descriptor.HasTest)
	require.Equal(t, true, descriptor.MissingSharedSchemes)
	require.Equal(t, "", descriptor.CarthageCommand)
}

func TestConfigName(t *testing.T) {
	{
		descriptor := NewConfigDescriptor(false, "", false, false)
		require.Equal(t, "ios-config", descriptor.ConfigName(utility.XcodeProjectTypeIOS))
	}

	{
		descriptor := NewConfigDescriptor(true, "", false, false)
		require.Equal(t, "ios-pod-config", descriptor.ConfigName(utility.XcodeProjectTypeIOS))
	}

	{
		descriptor := NewConfigDescriptor(false, "bootsrap", false, false)
		require.Equal(t, "ios-carthage-config", descriptor.ConfigName(utility.XcodeProjectTypeIOS))
	}

	{
		descriptor := NewConfigDescriptor(false, "", true, false)
		require.Equal(t, "ios-test-config", descriptor.ConfigName(utility.XcodeProjectTypeIOS))
	}

	{
		descriptor := NewConfigDescriptor(false, "", false, true)
		require.Equal(t, "ios-missing-shared-schemes-config", descriptor.ConfigName(utility.XcodeProjectTypeIOS))
	}

	{
		descriptor := NewConfigDescriptor(true, "bootstrap", false, false)
		require.Equal(t, "ios-pod-carthage-config", descriptor.ConfigName(utility.XcodeProjectTypeIOS))
	}

	{
		descriptor := NewConfigDescriptor(true, "bootstrap", true, false)
		require.Equal(t, "ios-pod-carthage-test-config", descriptor.ConfigName(utility.XcodeProjectTypeIOS))
	}

	{
		descriptor := NewConfigDescriptor(true, "bootstrap", true, true)
		require.Equal(t, "ios-pod-carthage-test-missing-shared-schemes-config", descriptor.ConfigName(utility.XcodeProjectTypeIOS))
	}
}
