package client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// A live /wallet/getaccount answer (requested without "visible", so addresses
// are hex), trimmed to the fields that need converting but otherwise verbatim -
// including the shapes protojson cannot read: a protobuf map rendered as an
// array of key/value objects, and hex where protojson expects base64.
const liveAccountWithPermissions = `{
	"address": "41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e",
	"balance": 123456,
	"net_window_size": 28800000,
	"net_window_optimized": true,
	"asset_optimized": true,
	"account_resource": {
		"energy_usage": 7,
		"latest_consume_time_for_energy": 1778753271000,
		"energy_window_size": 28800000,
		"delegated_frozenV2_balance_for_energy": 5000000,
		"acquired_delegated_frozenV2_balance_for_energy": 1000000,
		"energy_window_optimized": true
	},
	"owner_permission": {
		"permission_name": "owner",
		"threshold": 1,
		"keys": [{"address": "41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e", "weight": 1}]
	},
	"witness_permission": {
		"type": "Witness",
		"id": 1,
		"permission_name": "witness",
		"threshold": 1,
		"keys": [{"address": "418440ffd578f7a5abf3537b5f46a6980d382db581", "weight": 1}]
	},
	"active_permission": [{
		"type": "Active",
		"id": 2,
		"permission_name": "active",
		"threshold": 2,
		"operations": "7fff1fc0033e0300000000000000000000000000000000000000000000000000",
		"keys": [
			{"address": "41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e", "weight": 1},
			{"address": "41a614f803b6fd780986a42c78ec9c7f77e6ded13c", "weight": 1}
		]
	}],
	"assetV2": [{"key": "1004977", "value": 8888880000}, {"key": "1005026", "value": 970000}],
	"free_asset_net_usageV2": [{"key": "1004977", "value": 3}]
}`

func mustDecode(t *testing.T, addr string) []byte {
	t.Helper()

	decoded, err := tronutils.DecodeCheck(addr)
	require.NoError(t, err)

	return decoded
}

// Every one of these fields is something gRPC returns. The hand-written struct
// HTTP used to parse into dropped them, so the same account read over the two
// transports disagreed - silently, because a field nobody sets looks exactly
// like a field the node did not send.
func TestHTTPGetAccountKeepsResourceAndPermissions(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, liveAccountWithPermissions)

	acc, err := tr.GetAccount(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.NoError(t, err)

	res := acc.GetAccountResource()
	require.NotNil(t, res, "account_resource dropped")
	require.Equal(t, int64(7), res.GetEnergyUsage())
	require.Equal(t, int64(1_778_753_271_000), res.GetLatestConsumeTimeForEnergy())
	require.Equal(t, int64(28_800_000), res.GetEnergyWindowSize())
	require.Equal(t, int64(5_000_000), res.GetDelegatedFrozenV2BalanceForEnergy())
	require.Equal(t, int64(1_000_000), res.GetAcquiredDelegatedFrozenV2BalanceForEnergy())
	require.True(t, res.GetEnergyWindowOptimized())

	owner := acc.GetOwnerPermission()
	require.NotNil(t, owner, "owner_permission dropped")
	require.Equal(t, core.Permission_Owner, owner.GetType())
	require.Equal(t, "owner", owner.GetPermissionName())
	require.Equal(t, int64(1), owner.GetThreshold())
	require.Len(t, owner.GetKeys(), 1)
	require.Equal(t, mustDecode(t, "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"), owner.GetKeys()[0].GetAddress())

	witness := acc.GetWitnessPermission()
	require.NotNil(t, witness, "witness_permission dropped")
	require.Equal(t, core.Permission_Witness, witness.GetType())
	require.Equal(t, WitnessPermissionID, witness.GetId())
	require.Equal(t, "witness", witness.GetPermissionName())
	require.Equal(t, mustDecode(t, "TN2W4cc7a4dsYyTLiLMWa9m7jVpdLjGvYs"), witness.GetKeys()[0].GetAddress())

	require.Len(t, acc.GetActivePermission(), 1)
	active := acc.GetActivePermission()[0]
	require.Equal(t, core.Permission_Active, active.GetType())
	require.Equal(t, int32(2), active.GetId())
	require.Equal(t, int64(2), active.GetThreshold())
	require.Len(t, active.GetOperations(), 32)

	// Both signers, in order: a permission short of one key reads as a lower
	// signing quorum than the account actually has.
	require.Len(t, active.GetKeys(), 2)
	require.Equal(t, mustDecode(t, "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"), active.GetKeys()[0].GetAddress())
	require.Equal(t, mustDecode(t, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"), active.GetKeys()[1].GetAddress())
	require.Equal(t, int64(1), active.GetKeys()[1].GetWeight())
}

// Tron renders a protobuf map as an array of {key,value} objects, which is not
// a map to protojson either - the TRC10 balances came back empty.
func TestHTTPGetAccountKeepsAssetMaps(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, liveAccountWithPermissions)

	acc, err := tr.GetAccount(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.NoError(t, err)

	require.True(t, acc.GetAssetOptimized())
	require.Equal(t, map[string]int64{"1004977": 8_888_880_000, "1005026": 970_000}, acc.GetAssetV2())
	require.Equal(t, map[string]int64{"1004977": 3}, acc.GetFreeAssetNetUsageV2())
}

// An account with none of these is the common case and must stay an answer
// rather than an empty message full of zero-valued sub-messages.
func TestHTTPGetAccountWithoutOptionalSections(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"address":"41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e","balance":1}`)

	acc, err := tr.GetAccount(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.NoError(t, err)

	require.Equal(t, int64(1), acc.GetBalance())
	require.Nil(t, acc.GetAccountResource())
	require.Nil(t, acc.GetOwnerPermission())
	require.Nil(t, acc.GetWitnessPermission())
	require.Empty(t, acc.GetActivePermission())
	require.Empty(t, acc.GetAssetV2())
}

// The address the node echoes back is what Client.GetAccount compares against
// the one it asked for. Dropping a decode failure left it nil, which that
// comparison turns into ErrAccountNotFound for an account that is right there.
func TestHTTPGetAccountRejectsUnreadableAddress(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{"address":"not-an-address","balance":1}`)

	_, err := tr.GetAccount(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.ErrorContains(t, err, "Account.address")
}

// A permission key that will not decode is refused too: EncodeCheck turns any
// bytes back into a plausible address, so a mangled signer reads as a real one
// the account owner does not control.
func TestHTTPGetAccountRejectsUnreadablePermissionKey(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"address":"41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e",`+
			`"owner_permission":{"permission_name":"owner","keys":[{"address":"nope","weight":1}]}}`)

	_, err := tr.GetAccount(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.ErrorContains(t, err, "Key.address")
}

// The same map-as-array shape, one message over.
func TestHTTPGetAccountResourceKeepsAssetAndPowerFields(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{
		"freeNetLimit": 600, "NetLimit": 100, "TotalNetLimit": 43200000000,
		"EnergyLimit": 50, "TotalEnergyLimit": 180000000000,
		"TotalTronPowerWeight": 40000000000, "tronPowerUsed": 3, "tronPowerLimit": 7,
		"storageUsed": 11, "storageLimit": 13,
		"assetNetUsed": [{"key": "1004977", "value": 0}, {"key": "1005026", "value": 5}],
		"assetNetLimit": [{"key": "1004977", "value": 1}]
	}`)

	res, err := tr.GetAccountResource(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.NoError(t, err)

	require.Equal(t, map[string]int64{"1004977": 0, "1005026": 5}, res.GetAssetNetUsed())
	require.Equal(t, map[string]int64{"1004977": 1}, res.GetAssetNetLimit())

	require.Equal(t, int64(40_000_000_000), res.GetTotalTronPowerWeight())
	require.Equal(t, int64(3), res.GetTronPowerUsed())
	require.Equal(t, int64(7), res.GetTronPowerLimit())
	require.Equal(t, int64(11), res.GetStorageUsed())
	require.Equal(t, int64(13), res.GetStorageLimit())

	// The fields that already worked must keep working.
	require.Equal(t, int64(600), res.GetFreeNetLimit())
	require.Equal(t, int64(100), res.GetNetLimit())
	require.Equal(t, int64(50), res.GetEnergyLimit())
}

// is_witness governs whether a witness permission is legal at all
// (AccountPermissionUpdateActuator: "account isn't witness can't set witness
// permission"). Dropping it made the same SR read as a witness over gRPC and
// as an ordinary account over HTTP.
func TestHTTPGetAccountKeepsIsWitness(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"address":"41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e","balance":1,"is_witness":true}`)

	acc, err := tr.GetAccount(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.NoError(t, err)
	require.True(t, acc.GetIsWitness())
}

// A permission type the SDK does not know used to fall through the value map to
// 0, which is Owner - the one type PermissionAllows grants everything.
func TestHTTPGetAccountRejectsUnknownPermissionType(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"address":"41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e","active_permission":[`+
			`{"type":"Delegated","id":2,"permission_name":"a","threshold":1,"operations":"`+
			`0000000000000000000000000000000000000000000000000000000000000000",`+
			`"keys":[{"address":"41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e","weight":1}]}]}`)

	_, err := tr.GetAccount(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.ErrorContains(t, err, `unknown protocol.Permission.PermissionType value "Delegated"`)
}

// Tron omits "type" for the owner permission because Owner is the protobuf zero
// value, so an absent type must still parse.
func TestHTTPGetAccountAcceptsOwnerPermissionWithoutType(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK,
		`{"address":"41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e","owner_permission":`+
			`{"permission_name":"owner","threshold":1,`+
			`"keys":[{"address":"41fd49eda0f23ff7ec1d03b52c3a45991c24cd440e","weight":1}]}}`)

	acc, err := tr.GetAccount(t.Context(), &core.Account{Address: mustDecode(t, testAddr)})
	require.NoError(t, err)
	require.Equal(t, core.Permission_Owner, acc.GetOwnerPermission().GetType())
}
