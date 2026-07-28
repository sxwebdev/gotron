package client

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"google.golang.org/protobuf/proto"
)

// The two accounts of a real Nile delegation, and the node's own answers for
// it. Every address in them is base58 because the request sets visible=true.
const (
	delegationOwner    = "TDsCgUWZkyBLEEJXP2B1X81bASueT1Uqtf"
	delegationReceiver = "TMHy1fRsYsD4gCgTZbysd9yH9AQFQnTZQS"

	liveDelegationIndex = `{"account": "TDsCgUWZkyBLEEJXP2B1X81bASueT1Uqtf","toAccounts": ["TMHy1fRsYsD4gCgTZbysd9yH9AQFQnTZQS"]}`

	liveDelegationRecord = `{"delegatedResource": [{"from": "TDsCgUWZkyBLEEJXP2B1X81bASueT1Uqtf",` +
		`"to": "TMHy1fRsYsD4gCgTZbysd9yH9AQFQnTZQS","frozen_balance_for_bandwidth": 308593660,` +
		`"frozen_balance_for_energy": 677377195,"expire_time_for_energy": 1785353340000}]}`
)

// A base58 address is made only of base64 characters, so protojson decodes one
// into 24 bytes of a different account without reporting anything. The index
// then names a receiver that does not exist, the record lookup for it comes
// back empty, and an account with a live delegation is reported as lending
// nothing at all - while the same read over gRPC returns it.
func TestHTTPDelegationIndexV2DecodesBase58Addresses(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/getdelegatedresourceaccountindexv2": liveDelegationIndex,
	})

	owner, err := tronutils.DecodeCheck(delegationOwner)
	require.NoError(t, err)

	index, err := tr.GetDelegatedResourceAccountIndexV2(t.Context(), owner)
	require.NoError(t, err)

	require.Equal(t, delegationOwner, tronutils.EncodeCheck(index.GetAccount()))
	require.Len(t, index.GetToAccounts(), 1)
	require.Equal(t, delegationReceiver, tronutils.EncodeCheck(index.GetToAccounts()[0]))
}

func TestHTTPDelegatedResourceV2DecodesBase58Addresses(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/getdelegatedresourcev2": liveDelegationRecord,
	})

	owner, err := tronutils.DecodeCheck(delegationOwner)
	require.NoError(t, err)
	receiver, err := tronutils.DecodeCheck(delegationReceiver)
	require.NoError(t, err)

	list, err := tr.GetDelegatedResourceV2(t.Context(), &api.DelegatedResourceMessage{
		FromAddress: owner,
		ToAddress:   receiver,
	})
	require.NoError(t, err)
	require.Len(t, list.GetDelegatedResource(), 1)

	got := list.GetDelegatedResource()[0]
	require.Equal(t, delegationOwner, tronutils.EncodeCheck(got.GetFrom()))
	require.Equal(t, delegationReceiver, tronutils.EncodeCheck(got.GetTo()))
	require.Equal(t, int64(308_593_660), got.GetFrozenBalanceForBandwidth())
	require.Equal(t, int64(677_377_195), got.GetFrozenBalanceForEnergy())
	// A lock the caller has to be told about: reclaiming before it passes is
	// refused by the chain.
	require.Equal(t, int64(1_785_353_340_000), got.GetExpireTimeForEnergy())
	require.Zero(t, got.GetExpireTimeForBandwidth())
}

// The whole walk, which is what the caller actually uses: the receiver read out
// of the index has to be the one the record lookup is made with, or the second
// request asks about an account nobody delegated to.
func TestHTTPGetDelegatedResourcesV2ReturnsTheDelegation(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/getdelegatedresourceaccountindexv2": liveDelegationIndex,
		"/wallet/getdelegatedresourcev2":             liveDelegationRecord,
	})

	c := &Client{transport: tr, config: Config{}}

	lent, err := c.GetDelegatedResourcesV2(t.Context(), delegationOwner)
	require.NoError(t, err)
	require.Len(t, lent, 1)

	require.Equal(t, delegationOwner, lent[0].From)
	require.Equal(t, delegationReceiver, lent[0].To)
	require.Equal(t, SUN(308_593_660), lent[0].Bandwidth)
	require.Equal(t, SUN(677_377_195), lent[0].Energy)
}

// An account that has lent nothing out is an answer, not a failure.
func TestHTTPGetDelegatedResourcesV2EmptyIndex(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/getdelegatedresourceaccountindexv2": `{}`,
	})

	c := &Client{transport: tr, config: Config{}}

	lent, err := c.GetDelegatedResourcesV2(t.Context(), delegationOwner)
	require.NoError(t, err)
	require.Empty(t, lent)
}

// A /wallet endpoint reports a request it would not process at all as HTTP 200
// with an "Error" field. Dropped, it becomes an empty result with a nil error -
// which reads as "nothing is delegated" and sends the caller looking at the
// wrong account.
func TestHTTPDelegationSurfacesNodeRefusal(t *testing.T) {
	const refusal = `{"Error":"class org.tron.core.services.http.JsonFormat$ParseException : ` +
		`1:65: invalid address for field: protocol.DelegatedResourceMessage.toAddress"}`

	owner, err := tronutils.DecodeCheck(delegationOwner)
	require.NoError(t, err)
	receiver, err := tronutils.DecodeCheck(delegationReceiver)
	require.NoError(t, err)

	t.Run("record", func(t *testing.T) {
		tr := newRoutedTransport(t, map[string]string{"/wallet/getdelegatedresourcev2": refusal})

		_, err := tr.GetDelegatedResourceV2(t.Context(), &api.DelegatedResourceMessage{
			FromAddress: owner,
			ToAddress:   receiver,
		})
		require.ErrorContains(t, err, "invalid address for field")
	})

	t.Run("index", func(t *testing.T) {
		tr := newRoutedTransport(t, map[string]string{
			"/wallet/getdelegatedresourceaccountindexv2": refusal,
		})

		_, err := tr.GetDelegatedResourceAccountIndexV2(t.Context(), owner)
		require.ErrorContains(t, err, "invalid address for field")
	})
}

// Both delegation responses are hand-copied field by field, so a field added
// by a future `make genproto` would stay permanently zero over HTTP with no
// error and no failing test - which for a delegation reads as "nothing is
// locked" or "nothing was lent out". Every field non-zero here, so an omission
// cannot hide behind a legitimate zero.
func TestHTTPDelegationCarriesEveryProtoField(t *testing.T) {
	owner, err := tronutils.DecodeCheck(delegationOwner)
	require.NoError(t, err)
	receiver, err := tronutils.DecodeCheck(delegationReceiver)
	require.NoError(t, err)

	tr := newRoutedTransport(t, map[string]string{
		"/wallet/getdelegatedresourcev2": `{"delegatedResource":[{"from":"` + delegationOwner + `",` +
			`"to":"` + delegationReceiver + `","frozen_balance_for_bandwidth":1,"frozen_balance_for_energy":1,` +
			`"expire_time_for_bandwidth":1,"expire_time_for_energy":1}]}`,
		"/wallet/getdelegatedresourceaccountindexv2": `{"account":"` + delegationOwner + `",` +
			`"fromAccounts":["` + delegationReceiver + `"],"toAccounts":["` + delegationReceiver + `"],` +
			`"timestamp":1}`,
	})

	list, err := tr.GetDelegatedResourceV2(t.Context(), &api.DelegatedResourceMessage{
		FromAddress: owner,
		ToAddress:   receiver,
	})
	require.NoError(t, err)
	require.Len(t, list.GetDelegatedResource(), 1)
	requireEveryFieldSet(t, list.GetDelegatedResource()[0])

	index, err := tr.GetDelegatedResourceAccountIndexV2(t.Context(), owner)
	require.NoError(t, err)
	requireEveryFieldSet(t, index)
}

// requireEveryFieldSet fails naming any field of msg the conversion left unset.
func requireEveryFieldSet(t *testing.T, msg proto.Message) {
	t.Helper()

	reflected := msg.ProtoReflect()
	fields := reflected.Descriptor().Fields()

	var missing []string
	for i := range fields.Len() {
		if fd := fields.Get(i); !reflected.Has(fd) {
			missing = append(missing, string(fd.Name()))
		}
	}

	require.Empty(t, missing, "fields dropped between the JSON and %s", reflected.Descriptor().FullName())
}

// A malformed address is reported rather than passed on: EncodeCheck turns any
// bytes into a plausible-looking address, so a record kept with a half-decoded
// one names an account that simply does not exist.
func TestHTTPDelegationRejectsMalformedAddresses(t *testing.T) {
	tr := newRoutedTransport(t, map[string]string{
		"/wallet/getdelegatedresourceaccountindexv2": `{"account":"TDsCgUWZkyBLEEJXP2B1X81bASueT1Uqtf","toAccounts":["not-an-address"]}`,
	})

	owner, err := tronutils.DecodeCheck(delegationOwner)
	require.NoError(t, err)

	_, err = tr.GetDelegatedResourceAccountIndexV2(t.Context(), owner)
	require.ErrorIs(t, err, ErrInvalidAddress)
}
