package client

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

func TestResourceTypeValidate(t *testing.T) {
	require.NoError(t, ResourceTypeBandwidth.Validate())
	require.NoError(t, ResourceTypeEnergy.Validate())
	require.ErrorIs(t, ResourceType(7).Validate(), ErrInvalidResourceType)
}

func TestResourceTypeString(t *testing.T) {
	require.Equal(t, "BANDWIDTH", ResourceTypeBandwidth.String())
	require.Equal(t, "ENERGY", ResourceTypeEnergy.String())
	require.Equal(t, "UNKNOWN", ResourceType(7).String())
}

func TestResourceTypeToProto(t *testing.T) {
	require.Equal(t, core.ResourceCode_BANDWIDTH, ResourceTypeBandwidth.ToProto())
	require.Equal(t, core.ResourceCode_ENERGY, ResourceTypeEnergy.ToProto())
	require.Equal(t, core.ResourceCode(-1), ResourceType(7).ToProto())
}

func TestNetworkValidate(t *testing.T) {
	for _, n := range []Network{NetworkMainnet, NetworkShasta, NetworkNile} {
		require.NoError(t, n.Validate(), "network %s", n)
	}
	require.Error(t, Network("eth").Validate())
}

func TestNetworkString(t *testing.T) {
	require.Equal(t, "mainnet", NetworkMainnet.String())
}
