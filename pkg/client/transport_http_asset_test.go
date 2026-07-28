package client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
)

// The node's own answer for asset 1002000, verbatim. Every bytes field in it is
// hex: an address, a short name, a long description, a URL.
const liveAssetIssue = `{"owner_address": "4137fa1a56eb8c503624701d776d95f6dae1d9f0d6",` +
	`"name": "426974546f7272656e74","abbr": "425454","total_supply": 990000000000000000,` +
	`"trx_num": 1,"precision": 6,"num": 1,"start_time": 1548000000000,"end_time": 1548000001000,` +
	`"description": "4f6666696369616c20546f6b656e206f6620426974546f7272656e742050726f746f636f6c",` +
	`"url": "7777772e626974746f7272656e742e636f6d","id": "1002000"}`

const bttOwner = "TF5Bn4cJCT6GVeUgyCN4rBhDg42KBrpAjg"

// The name is a bytes field, so it goes over the wire hex-encoded. Sent plain it
// is not searched for and not found - the node refuses the request outright with
// "invalid characters encountered in Hex string", at HTTP 200, which read as an
// asset that does not exist.
func TestHTTPAssetIssueListByNameSendsHexName(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, `{"assetIssue":[`+liveAssetIssue+`]}`)

	_, err := tr.GetAssetIssueListByName(t.Context(), []byte("BitTorrent"))
	require.NoError(t, err)

	require.NotNil(t, *lastReq)
	require.Equal(t, "426974546f7272656e74", (*lastReq)["value"])
}

// The id is a string field and must not get the same treatment: hex-encoding it
// would search for the bytes of the digits instead of the asset.
func TestHTTPAssetIssueByIdSendsIdAsIs(t *testing.T) {
	tr, lastReq := newStubTransport(t, http.StatusOK, liveAssetIssue)

	_, err := tr.GetAssetIssueById(t.Context(), []byte("1002000"))
	require.NoError(t, err)

	require.NotNil(t, *lastReq)
	require.Equal(t, "1002000", (*lastReq)["value"])
}

// protojson reads a bytes field as base64, and an even-length hex string is
// valid base64 - so it decoded every one of these into other bytes and reported
// no error. The owner address is the damage that matters: 31 bytes of nonsense
// that EncodeCheck still turns into a plausible-looking address.
func TestHTTPAssetIssueDecodesHexFields(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, liveAssetIssue)

	asset, err := tr.GetAssetIssueById(t.Context(), []byte("1002000"))
	require.NoError(t, err)

	require.Equal(t, "BitTorrent", string(asset.GetName()))
	require.Equal(t, "BTT", string(asset.GetAbbr()))
	require.Equal(t, "Official Token of BitTorrent Protocol", string(asset.GetDescription()))
	require.Equal(t, "www.bittorrent.com", string(asset.GetUrl()))

	require.Len(t, asset.GetOwnerAddress(), 21)
	require.Equal(t, bttOwner, tronutils.EncodeCheck(asset.GetOwnerAddress()))

	// The scalars have to survive the switch away from protojson.
	require.Equal(t, "1002000", asset.GetId())
	require.Equal(t, int64(990_000_000_000_000_000), asset.GetTotalSupply())
	require.Equal(t, int32(6), asset.GetPrecision())
	require.Equal(t, int32(1), asset.GetTrxNum())
	require.Equal(t, int32(1), asset.GetNum())
	require.Equal(t, int64(1_548_000_000_000), asset.GetStartTime())
	require.Equal(t, int64(1_548_000_001_000), asset.GetEndTime())
}

// A list is the same records under a wrapper, and the wrapper key is "assetIssue".
func TestHTTPAssetIssueListDecodesEveryRecord(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{"assetIssue":[`+liveAssetIssue+`,`+
		`{"id":"1001927","name":"425454","abbr":"425454","owner_address":"4113189bb13f1ec4f45c88526bd05482f482c06a11",`+
		`"frozen_supply":[{"frozen_amount":1,"frozen_days":1}]}]}`)

	list, err := tr.GetAssetIssueListByName(t.Context(), []byte("x"))
	require.NoError(t, err)
	require.Len(t, list.GetAssetIssue(), 2)

	require.Equal(t, "BitTorrent", string(list.GetAssetIssue()[0].GetName()))

	second := list.GetAssetIssue()[1]
	require.Equal(t, "BTT", string(second.GetName()))
	require.Len(t, second.GetOwnerAddress(), 21)
	require.Len(t, second.GetFrozenSupply(), 1)
	require.Equal(t, int64(1), second.GetFrozenSupply()[0].GetFrozenAmount())
	require.Equal(t, int64(1), second.GetFrozenSupply()[0].GetFrozenDays())
}

// An asset nobody issued is an answer, not a failure: the node replies "{}".
func TestHTTPAssetIssueEmptyAnswer(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{}`)

	asset, err := tr.GetAssetIssueById(t.Context(), []byte("1"))
	require.NoError(t, err)
	require.Empty(t, asset.GetId())
	require.Nil(t, asset.GetName())
	require.Nil(t, asset.GetOwnerAddress())

	list, err := tr.GetAssetIssueListByName(t.Context(), []byte("nothing"))
	require.NoError(t, err)
	require.Empty(t, list.GetAssetIssue())
}

// Every field the proto declares has to be carried over. One left out of
// toProto reads as zero, which for an asset means a supply of nothing or a
// frozen period that ended - and no other test would notice, because a value
// that is never set looks exactly like a value the node did not send.
func TestHTTPAssetIssueCarriesEveryProtoField(t *testing.T) {
	// Every field non-zero, so an omission cannot hide behind a legitimate zero.
	tr, _ := newStubTransport(t, http.StatusOK, `{"id":"1002000",`+
		`"owner_address":"4137fa1a56eb8c503624701d776d95f6dae1d9f0d6","name":"42","abbr":"42",`+
		`"total_supply":1,"frozen_supply":[{"frozen_amount":1,"frozen_days":1}],"trx_num":1,`+
		`"precision":1,"num":1,"start_time":1,"end_time":1,"order":1,"vote_score":1,`+
		`"description":"42","url":"42","free_asset_net_limit":1,"public_free_asset_net_limit":1,`+
		`"public_free_asset_net_usage":1,"public_latest_free_net_time":1}`)

	asset, err := tr.GetAssetIssueById(t.Context(), []byte("1002000"))
	require.NoError(t, err)

	requireEveryFieldSet(t, asset)
}

// Nonsense in a bytes field is reported rather than dropped, so a field that
// stops being hex is not silently read as an asset with no name.
func TestHTTPAssetIssueRejectsMalformedHex(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{"id":"1002000","name":"zz"}`)

	_, err := tr.GetAssetIssueById(t.Context(), []byte("1002000"))
	require.ErrorContains(t, err, "decode name")
}
