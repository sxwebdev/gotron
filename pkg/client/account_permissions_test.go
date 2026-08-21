package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

func delegatorOperations(t *testing.T) []byte {
	t.Helper()
	operations, err := ContractOperations(
		core.Transaction_Contract_TriggerSmartContract,
		core.Transaction_Contract_DelegateResourceContract,
		core.Transaction_Contract_UnDelegateResourceContract,
	)
	require.NoError(t, err)
	return operations
}

func TestContractOperationsEncodesProtocolBitmap(t *testing.T) {
	operations := delegatorOperations(t)
	require.Len(t, operations, 32)
	require.Equal(
		t,
		"0000008000000006000000000000000000000000000000000000000000000000",
		hex.EncodeToString(operations),
	)

	require.True(t, PermissionAllows(&core.Permission{Type: core.Permission_Active, Operations: operations}, core.Transaction_Contract_TriggerSmartContract))
	require.False(t, PermissionAllows(&core.Permission{Type: core.Permission_Active, Operations: operations}, core.Transaction_Contract_TransferContract))
	require.True(t, PermissionAllows(&core.Permission{Type: core.Permission_Owner}, core.Transaction_Contract_TransferContract))
	require.False(t, PermissionAllows(&core.Permission{Type: core.Permission_Witness}, core.Transaction_Contract_TransferContract))
}

func TestContractOperationsRejectsUnknownContractType(t *testing.T) {
	_, err := ContractOperations(core.Transaction_Contract_ContractType(255))
	require.ErrorIs(t, err, ErrInvalidPermission)

	_, err = ContractOperations(core.Transaction_Contract_ContractType(60))
	require.ErrorIs(t, err, ErrInvalidPermission)
	require.False(t, PermissionAllows(
		&core.Permission{Type: core.Permission_Active, Operations: make([]byte, permissionOperationsLen)},
		core.Transaction_Contract_ContractType(60),
	))
}

func TestPermissionConstructorsValidateThresholdAndKeys(t *testing.T) {
	owner, err := NewOwnerPermission("owner", 1, PermissionKey{Address: testAddr, Weight: 1})
	require.NoError(t, err)
	require.Equal(t, core.Permission_Owner, owner.GetType())
	require.Empty(t, owner.GetOperations())

	active, err := NewActivePermission("delegator", 1, delegatorOperations(t), PermissionKey{Address: testAddr2, Weight: 1})
	require.NoError(t, err)
	require.Equal(t, core.Permission_Active, active.GetType())
	require.Equal(t, delegatorOperations(t), active.GetOperations())

	witness, err := NewWitnessPermission("witness", 1, PermissionKey{Address: testAddr2, Weight: 1})
	require.NoError(t, err)
	require.Equal(t, core.Permission_Witness, witness.GetType())
	require.Equal(t, WitnessPermissionID, witness.GetId())
	require.Len(t, witness.GetKeys(), 1)

	_, err = NewActivePermission("delegator", 2, delegatorOperations(t), PermissionKey{Address: testAddr2, Weight: 1})
	require.ErrorIs(t, err, ErrInvalidPermission)

	_, err = NewOwnerPermission("owner", 1, PermissionKey{Address: "bad", Weight: 1})
	require.ErrorIs(t, err, ErrInvalidAddress)

	_, err = NewOwnerPermission(
		"owner", 2,
		PermissionKey{Address: testAddr, Weight: 1},
		PermissionKey{Address: testAddr, Weight: 1},
	)
	require.ErrorIs(t, err, ErrInvalidPermission)
}

func TestPermissionNameUsesJavaUTF16Length(t *testing.T) {
	_, err := NewOwnerPermission(strings.Repeat("я", 32), 1, PermissionKey{Address: testAddr, Weight: 1})
	require.NoError(t, err, "32 Cyrillic characters are 32 Java String code units despite occupying 64 UTF-8 bytes")

	_, err = NewOwnerPermission(strings.Repeat("я", 33), 1, PermissionKey{Address: testAddr, Weight: 1})
	require.ErrorIs(t, err, ErrInvalidPermission)

	_, err = NewOwnerPermission(strings.Repeat("😀", 16), 1, PermissionKey{Address: testAddr, Weight: 1})
	require.NoError(t, err, "16 supplementary runes occupy 32 UTF-16 code units")

	_, err = NewOwnerPermission(strings.Repeat("😀", 17), 1, PermissionKey{Address: testAddr, Weight: 1})
	require.ErrorIs(t, err, ErrInvalidPermission)
}

func TestPermissionConstructorDoesNotHardcodeDynamicKeyLimit(t *testing.T) {
	keys := make([]PermissionKey, 6)
	for i := range keys {
		address := make([]byte, tronutils.AddressLength)
		address[0] = tronutils.TronBytePrefix
		address[len(address)-1] = byte(i + 1)
		keys[i] = PermissionKey{Address: tronutils.EncodeCheck(address), Weight: 1}
	}

	permission, err := NewOwnerPermission("owner", 6, keys...)
	require.NoError(t, err)
	require.Len(t, permission.GetKeys(), 6)
}

func TestValidatePermissionSigner(t *testing.T) {
	accountAddress, err := tronutils.DecodeCheck(testAddr)
	require.NoError(t, err)
	signerAddress, err := tronutils.DecodeCheck(testAddr2)
	require.NoError(t, err)

	permission := &core.Permission{
		Type:       core.Permission_Active,
		Id:         2,
		Threshold:  2,
		Operations: delegatorOperations(t),
		Keys:       []*core.Key{{Address: signerAddress, Weight: 2}},
	}
	newClient := func(p *core.Permission) *Client {
		t.Helper()
		return newTestClient(&fakeTransport{getAccount: func(context.Context, *core.Account) (*core.Account, error) {
			return &core.Account{Address: accountAddress, ActivePermission: []*core.Permission{p}}, nil
		}})
	}

	t.Run("authorized", func(t *testing.T) {
		err := newClient(permission).ValidatePermissionSigner(
			t.Context(), testAddr, testAddr2, 2,
			core.Transaction_Contract_TriggerSmartContract,
			core.Transaction_Contract_DelegateResourceContract,
			core.Transaction_Contract_UnDelegateResourceContract,
		)
		require.NoError(t, err)
	})

	t.Run("key below threshold", func(t *testing.T) {
		p := proto.Clone(permission).(*core.Permission)
		p.Keys[0].Weight = 1
		err := newClient(p).ValidatePermissionSigner(t.Context(), testAddr, testAddr2, 2)
		require.ErrorIs(t, err, ErrPermissionDenied)
		require.Contains(t, err.Error(), "weight 1 is below permission 2 threshold 2")
	})

	// An address that holds no key at all reported "weight 0 is below
	// threshold", which reads as a registered but under-weighted key and sends
	// the operator after a threshold change instead of after the missing key.
	t.Run("signer holds no key", func(t *testing.T) {
		err := newClient(permission).ValidatePermissionSigner(t.Context(), testAddr, testAddr, 2)
		require.ErrorIs(t, err, ErrPermissionDenied)
		require.Contains(t, err.Error(), "is not a key of permission 2")
		require.NotContains(t, err.Error(), "weight 0")
	})

	t.Run("operation denied", func(t *testing.T) {
		err := newClient(permission).ValidatePermissionSigner(
			t.Context(), testAddr, testAddr2, 2,
			core.Transaction_Contract_TransferContract,
		)
		require.ErrorIs(t, err, ErrPermissionDenied)
	})

	t.Run("permission missing", func(t *testing.T) {
		err := newClient(permission).ValidatePermissionSigner(t.Context(), testAddr, testAddr2, 3)
		require.ErrorIs(t, err, ErrPermissionNotFound)
	})

	t.Run("foreign signer with zero threshold is not accepted", func(t *testing.T) {
		p := proto.Clone(permission).(*core.Permission)
		p.Threshold = 0
		err := newClient(p).ValidatePermissionSigner(t.Context(), testAddr, testAddr, 2)
		require.ErrorIs(t, err, ErrInvalidPermission)
	})

	// java-tron refuses a non-zero permission id whose permission is not Active
	// before it consults the bitmap. Owner is the protobuf zero value, so an
	// active slot holding an Owner-typed permission - or one whose type never
	// arrived - otherwise reads as authorizing every contract type.
	t.Run("active slot not typed active", func(t *testing.T) {
		for _, typ := range []core.Permission_PermissionType{core.Permission_Owner, core.Permission_Witness} {
			p := proto.Clone(permission).(*core.Permission)
			p.Type = typ
			err := newClient(p).ValidatePermissionSigner(
				t.Context(), testAddr, testAddr2, 2,
				core.Transaction_Contract_TransferContract,
			)
			require.ErrorIs(t, err, ErrInvalidPermission, typ)
		}
	})

	t.Run("invalid signer address is typed", func(t *testing.T) {
		err := newClient(permission).ValidatePermissionSigner(t.Context(), testAddr, "bad", 2)
		require.ErrorIs(t, err, ErrInvalidAddress)
	})

	t.Run("witness permission cannot authorize a transaction", func(t *testing.T) {
		calls := 0
		c := newTestClient(&fakeTransport{getAccount: func(context.Context, *core.Account) (*core.Account, error) {
			calls++
			return &core.Account{}, nil
		}})

		err := c.ValidatePermissionSigner(t.Context(), testAddr, testAddr2, WitnessPermissionID)
		require.ErrorIs(t, err, ErrInvalidPermissionID)
		require.Zero(t, calls, "an invalid transaction permission must be rejected before an RPC")
	})
}

func TestGetAccountPermissionSynthesizesImplicitOwner(t *testing.T) {
	accountAddress := mustDecode(t, testAddr)
	calls := 0
	c := newTestClient(&fakeTransport{getAccount: func(context.Context, *core.Account) (*core.Account, error) {
		calls++
		return &core.Account{Address: accountAddress}, nil
	}})

	permission, err := c.GetAccountPermission(t.Context(), testAddr, OwnerPermissionID)
	require.NoError(t, err)
	require.Equal(t, core.Permission_Owner, permission.GetType())
	require.Equal(t, OwnerPermissionID, permission.GetId())
	require.Equal(t, "owner", permission.GetPermissionName())
	require.EqualValues(t, 1, permission.GetThreshold())
	require.Equal(t, accountAddress, permission.GetKeys()[0].GetAddress())

	err = c.ValidatePermissionSigner(t.Context(), testAddr, testAddr, OwnerPermissionID)
	require.NoError(t, err)
	require.Equal(t, 2, calls, "each public validation/read performs exactly one account RPC")
}

func TestValidatePermissionRejectsNonTronKeyPrefix(t *testing.T) {
	address := make([]byte, tronutils.AddressLength)
	address[0] = tronutils.TronBytePrefix + 1
	permission := &core.Permission{
		Type:      core.Permission_Owner,
		Threshold: 1,
		Keys:      []*core.Key{{Address: address, Weight: 1}},
	}

	require.ErrorIs(t, validatePermission(permission, core.Permission_Owner), ErrInvalidPermission)
}

func TestUpdateAccountPermissionsBuildsDetachedContract(t *testing.T) {
	owner, err := NewOwnerPermission("owner", 1, PermissionKey{Address: testAddr, Weight: 1})
	require.NoError(t, err)
	active, err := NewActivePermission("delegator", 1, delegatorOperations(t), PermissionKey{Address: testAddr2, Weight: 1})
	require.NoError(t, err)

	var got *core.AccountPermissionUpdateContract
	c := newTestClient(&fakeTransport{accountPermissionUpdate: func(_ context.Context, contract *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
		got = contract
		return okTx(), nil
	}})
	tx, err := c.UpdateAccountPermissions(t.Context(), AccountPermissionUpdateRequest{
		Account: testAddr,
		Owner:   owner,
		Actives: []*core.Permission{active},
	})
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, got)
	require.Equal(t, mustDecode(t, testAddr), got.GetOwnerAddress())
	require.Len(t, got.GetActives(), 1)

	// The caller can reuse or wipe its request after the build; the transport
	// receives a detached protobuf tree.
	owner.Keys[0].Weight = 99
	active.Operations[3] = 0
	require.EqualValues(t, 1, got.GetOwner().GetKeys()[0].GetWeight())
	require.True(t, PermissionAllows(got.GetActives()[0], core.Transaction_Contract_TriggerSmartContract))
}

func TestUpdateAccountPermissionsRejectsInvalidCompleteSet(t *testing.T) {
	owner, err := NewOwnerPermission("owner", 1, PermissionKey{Address: testAddr, Weight: 1})
	require.NoError(t, err)
	c := newTestClient(&fakeTransport{})

	tests := []struct {
		name    string
		req     AccountPermissionUpdateRequest
		wantErr error
	}{
		{name: "invalid account", req: AccountPermissionUpdateRequest{Account: "bad", Owner: owner, Actives: []*core.Permission{owner}}, wantErr: ErrInvalidAddress},
		{name: "missing owner", req: AccountPermissionUpdateRequest{Account: testAddr, Actives: []*core.Permission{owner}}, wantErr: ErrInvalidPermission},
		{name: "missing active", req: AccountPermissionUpdateRequest{Account: testAddr, Owner: owner}, wantErr: ErrInvalidPermission},
		{name: "owner used as active", req: AccountPermissionUpdateRequest{Account: testAddr, Owner: owner, Actives: []*core.Permission{owner}}, wantErr: ErrInvalidPermission},
		{name: "active has witness id", req: AccountPermissionUpdateRequest{Account: testAddr, Owner: owner, Actives: []*core.Permission{{Type: core.Permission_Active, Id: WitnessPermissionID, Threshold: 1, Operations: delegatorOperations(t), Keys: owner.GetKeys()}}}, wantErr: ErrInvalidPermission},
		{name: "nonzero parent", req: AccountPermissionUpdateRequest{Account: testAddr, Owner: &core.Permission{Type: core.Permission_Owner, ParentId: 1, Threshold: 1, Keys: owner.GetKeys()}, Actives: []*core.Permission{owner}}, wantErr: ErrInvalidPermission},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.UpdateAccountPermissions(t.Context(), tt.req)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSetPermissionID(t *testing.T) {
	tx := &api.TransactionExtention{
		Transaction: &core.Transaction{RawData: &core.TransactionRaw{Contract: []*core.Transaction_Contract{{Type: core.Transaction_Contract_DelegateResourceContract}}}},
		Txid:        []byte("stale transaction id"),
	}
	require.NoError(t, SetPermissionID(tx, 2))
	require.Equal(t, int32(2), tx.GetTransaction().GetRawData().GetContract()[0].GetPermissionId())
	raw, err := proto.Marshal(tx.GetTransaction().GetRawData())
	require.NoError(t, err)
	wantTxID := sha256.Sum256(raw)
	require.Equal(t, wantTxID[:], tx.GetTxid())

	for _, id := range []int32{-1, WitnessPermissionID, 10} {
		err := SetPermissionID(tx, id)
		require.ErrorIs(t, err, ErrInvalidPermissionID)
	}
	tx.GetTransaction().Signature = [][]byte{{1, 2, 3}}
	oldTxID := append([]byte(nil), tx.GetTxid()...)
	err = SetPermissionID(tx, 3)
	require.ErrorIs(t, err, ErrInvalidTransaction)
	require.Equal(t, int32(2), tx.GetTransaction().GetRawData().GetContract()[0].GetPermissionId())
	require.Equal(t, oldTxID, tx.GetTxid())
	require.ErrorIs(t, SetPermissionID(&api.TransactionExtention{}, 2), ErrInvalidTransaction)
	require.ErrorIs(t, SetPermissionID(&api.TransactionExtention{Transaction: &core.Transaction{RawData: &core.TransactionRaw{Contract: []*core.Transaction_Contract{nil}}}}, 2), ErrInvalidTransaction)
}

func TestUpdateAccountPermissionsSurfacesTransportAndResultErrors(t *testing.T) {
	owner, err := NewOwnerPermission("owner", 1, PermissionKey{Address: testAddr, Weight: 1})
	require.NoError(t, err)
	active, err := NewActivePermission("active", 1, delegatorOperations(t), PermissionKey{Address: testAddr2, Weight: 1})
	require.NoError(t, err)
	req := AccountPermissionUpdateRequest{Account: testAddr, Owner: owner, Actives: []*core.Permission{active}}

	want := errors.New("transport down")
	c := newTestClient(&fakeTransport{accountPermissionUpdate: func(context.Context, *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
		return nil, want
	}})
	_, err = c.UpdateAccountPermissions(t.Context(), req)
	require.ErrorIs(t, err, want)

	c = newTestClient(&fakeTransport{accountPermissionUpdate: func(context.Context, *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
		return &api.TransactionExtention{
			Result: &api.Return{Code: api.Return_CONTRACT_VALIDATE_ERROR, Message: []byte("bad permission")},
		}, nil
	}})
	_, err = c.UpdateAccountPermissions(t.Context(), req)
	var validateErr *ContractValidateError
	require.ErrorAs(t, err, &validateErr)
}
