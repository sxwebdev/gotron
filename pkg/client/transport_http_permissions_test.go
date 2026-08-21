package client

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

func TestHTTPAccountPermissionUpdateUsesTronJSONAndParsesTransaction(t *testing.T) {
	raw := &core.TransactionRaw{Contract: []*core.Transaction_Contract{{Type: core.Transaction_Contract_AccountPermissionUpdateContract}}}
	rawBytes, err := proto.Marshal(raw)
	require.NoError(t, err)
	digest := sha256.Sum256(rawBytes)
	response := fmt.Sprintf(`{"raw_data_hex":%q,"txID":%q}`, hex.EncodeToString(rawBytes), hex.EncodeToString(digest[:]))
	transport, request := newStubTransportAtPath(t, "/wallet/accountpermissionupdate", http.StatusOK, response)

	owner, err := NewOwnerPermission("owner", 1, PermissionKey{Address: testAddr, Weight: 1})
	require.NoError(t, err)
	witness, err := NewWitnessPermission("witness", 1, PermissionKey{Address: testAddr, Weight: 1})
	require.NoError(t, err)
	active, err := NewActivePermission("delegator", 1, delegatorOperations(t), PermissionKey{Address: testAddr2, Weight: 1})
	require.NoError(t, err)
	tx, err := transport.AccountPermissionUpdate(t.Context(), &core.AccountPermissionUpdateContract{
		OwnerAddress: mustDecode(t, testAddr),
		Owner:        owner,
		Witness:      witness,
		Actives:      []*core.Permission{active},
	})
	require.NoError(t, err)
	require.Equal(t, core.Transaction_Contract_AccountPermissionUpdateContract, tx.GetTransaction().GetRawData().GetContract()[0].GetType())

	require.Equal(t, testAddr, (*request)["owner_address"])
	require.Equal(t, true, (*request)["visible"])
	ownerJSON := (*request)["owner"].(map[string]any)
	require.EqualValues(t, 0, ownerJSON["type"])
	require.EqualValues(t, 1, ownerJSON["threshold"])
	require.NotContains(t, ownerJSON, "operations")
	witnessJSON := (*request)["witness"].(map[string]any)
	require.EqualValues(t, WitnessPermissionID, witnessJSON["id"])
	require.EqualValues(t, core.Permission_Witness, witnessJSON["type"])
	require.Equal(t, testAddr, witnessJSON["keys"].([]any)[0].(map[string]any)["address"])
	activesJSON := (*request)["actives"].([]any)
	require.Len(t, activesJSON, 1)
	activeJSON := activesJSON[0].(map[string]any)
	require.EqualValues(t, 2, activeJSON["type"])
	require.Equal(t, "0000008000000006000000000000000000000000000000000000000000000000", activeJSON["operations"])
	keysJSON := activeJSON["keys"].([]any)
	require.Equal(t, testAddr2, keysJSON[0].(map[string]any)["address"])
}
