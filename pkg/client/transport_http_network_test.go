package client

import (
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// The hosts come from a live /wallet/listnodes answer (3.111.147.19:8090) and
// cover all three residues of the hex length mod 4: base64 decoding a 28-char
// string loses nothing, but a 26- or 30-char one drops trailing bits, so those
// are the hosts a protojson-based decoder could not even recover afterwards.
func TestHTTPListNodesDecodesHexHosts(t *testing.T) {
	const body = `{"nodes": [` +
		`{"address": {"host": "3230352e3230392e3131332e323534","port": 18888}},` +
		`{"address": {"host": "3130342e3234332e3133362e3631","port": 18888}},` +
		`{"address": {"host": "33352e3233342e39352e323533","port": 18935}}]}`

	tr, _ := newStubTransportAtPath(t, "/wallet/listnodes", http.StatusOK, body)

	res, err := tr.ListNodes(t.Context())
	require.NoError(t, err)

	type peer struct {
		host string
		port int32
	}
	got := make([]peer, 0, len(res.GetNodes()))
	for _, n := range res.GetNodes() {
		got = append(got, peer{string(n.GetAddress().GetHost()), n.GetAddress().GetPort()})
	}
	require.Equal(t, []peer{
		{"205.209.113.254", 18888},
		{"104.243.136.61", 18888},
		{"35.234.95.253", 18935},
	}, got)
}

// A node with listnodes disabled answers "{}"; that is an empty list, not an error.
func TestHTTPListNodesEmpty(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{}`)

	res, err := tr.ListNodes(t.Context())
	require.NoError(t, err)
	require.Empty(t, res.GetNodes())
}

// gRPC leaves Address nil for such an entry, so HTTP must not invent an empty one.
func TestHTTPListNodesKeepsMissingAddress(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{"nodes":[{}]}`)

	res, err := tr.ListNodes(t.Context())
	require.NoError(t, err)
	require.Len(t, res.GetNodes(), 1)
	require.Nil(t, res.GetNodes()[0].GetAddress())
}

func TestHTTPListNodesRejectsNonHexHost(t *testing.T) {
	tr, _ := newStubTransport(t, http.StatusOK, `{"nodes":[{"address":{"host":"1.2.3.4","port":18888}}]}`)

	_, err := tr.ListNodes(t.Context())

	var hexErr hex.InvalidByteError
	require.ErrorAs(t, err, &hexErr)
}
