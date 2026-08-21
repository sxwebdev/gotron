package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"unicode/utf16"

	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
)

const (
	OwnerPermissionID       int32 = 0
	WitnessPermissionID     int32 = 1
	FirstActivePermissionID int32 = 2
	permissionOperationsLen       = 32
	maxActivePermissions          = 8
	// java-tron rejects more than maxActivePermissions actives and assigns them
	// ids from FirstActivePermissionID upwards, which is what makes the last id
	// what it is. Deriving it keeps one source of truth for the two validators.
	LastActivePermissionID    = FirstActivePermissionID + maxActivePermissions - 1
	maxPermissionNameUTF16Len = 32
)

var (
	ErrInvalidPermissionID = errors.New("invalid permission id")
	ErrInvalidPermission   = errors.New("invalid permission")
	ErrPermissionNotFound  = errors.New("permission not found")
	ErrPermissionDenied    = errors.New("permission denied")
)

// PermissionKey is one address and its signing weight in a permission.
type PermissionKey struct {
	Address string
	Weight  int64
}

// AccountPermissionUpdateRequest replaces the complete permission set of Account.
// GetAccount can be used to copy permissions that must be preserved.
type AccountPermissionUpdateRequest struct {
	Account string
	Owner   *core.Permission
	Witness *core.Permission
	Actives []*core.Permission
}

// ContractOperations builds the 32-byte little-endian bitmap used by an active
// permission. Each bit corresponds to one ContractType known to the vendored
// protobuf.
//
// That is not the chain's own list: java-tron checks every bit against the
// dynamic getAvailableContractType parameter, so a type this SDK knows may
// still be refused (ShieldedTransferContract is, on mainnet today), and a type
// enabled on chain after the last `make genproto` cannot be granted here at
// all. Either way the node has the final say and reports it as a
// ContractValidateError naming the bit.
func ContractOperations(types ...core.Transaction_Contract_ContractType) ([]byte, error) {
	operations := make([]byte, permissionOperationsLen)
	for _, typ := range types {
		index, mask, registered := contractOperationBit(typ)
		if !registered {
			return nil, fmt.Errorf("%w: unregistered contract type id %d", ErrInvalidPermission, int32(typ))
		}
		operations[index] |= mask
	}
	return operations, nil
}

func contractOperationBit(typ core.Transaction_Contract_ContractType) (index int, mask byte, registered bool) {
	id := int32(typ)
	if id < 0 || id >= permissionOperationsLen*8 {
		return 0, 0, false
	}
	if _, registered = core.Transaction_Contract_ContractType_name[id]; !registered {
		return 0, 0, false
	}
	return int(id / 8), byte(1 << (id % 8)), true
}

// PermissionAllows reports whether permission authorizes a contract type.
// Owner permission allows every registered type; active permissions consult
// their operations bitmap. Witness permission cannot authorize transactions.
//
// Owner is the protobuf zero value, so a *core.Permission whose Type was never
// assigned is an owner permission as far as protobuf is concerned, and this
// reports that it allows everything. No reader of the message can tell the two
// apart, and neither transport can either - the HTTP parser rejects only a type
// name it cannot resolve, and gRPC decodes whatever the node sent. So this
// predicate answers for the permission it is given and nothing more; the check
// that closes the gap lives in ValidatePermissionSigner, which refuses a
// non-zero permission id that did not arrive typed Active.
func PermissionAllows(permission *core.Permission, typ core.Transaction_Contract_ContractType) bool {
	if permission == nil {
		return false
	}
	switch permission.GetType() {
	case core.Permission_Owner:
		_, _, registered := contractOperationBit(typ)
		return registered
	case core.Permission_Active:
		index, mask, registered := contractOperationBit(typ)
		operations := permission.GetOperations()
		return registered && index < len(operations) && operations[index]&mask != 0
	default:
		return false
	}
}

// NewOwnerPermission builds and validates an owner permission.
func NewOwnerPermission(name string, threshold int64, keys ...PermissionKey) (*core.Permission, error) {
	return newPermission(core.Permission_Owner, name, threshold, nil, keys)
}

// NewWitnessPermission builds and validates a witness permission. Witness
// permissions contain exactly one key and cannot authorize transactions.
func NewWitnessPermission(name string, threshold int64, key PermissionKey) (*core.Permission, error) {
	return newPermission(core.Permission_Witness, name, threshold, nil, []PermissionKey{key})
}

// NewActivePermission builds and validates an active permission. The network
// assigns its final id from its position in AccountPermissionUpdateRequest.Actives.
func NewActivePermission(name string, threshold int64, operations []byte, keys ...PermissionKey) (*core.Permission, error) {
	return newPermission(core.Permission_Active, name, threshold, operations, keys)
}

func newPermission(permissionType core.Permission_PermissionType, name string, threshold int64, operations []byte, keys []PermissionKey) (*core.Permission, error) {
	protoKeys := make([]*core.Key, 0, len(keys))
	for _, key := range keys {
		address, err := tronutils.DecodeCheck(key.Address)
		if err != nil {
			return nil, fmt.Errorf("%w: permission key %q: %v", ErrInvalidAddress, key.Address, err)
		}
		protoKeys = append(protoKeys, &core.Key{Address: address, Weight: key.Weight})
	}
	permission := &core.Permission{
		Type:           permissionType,
		PermissionName: name,
		Threshold:      threshold,
		Operations:     slices.Clone(operations),
		Keys:           protoKeys,
	}
	if permissionType == core.Permission_Witness {
		permission.Id = WitnessPermissionID
	}
	if err := validatePermission(permission, permissionType); err != nil {
		return nil, err
	}
	return permission, nil
}

// GetAccountPermission returns the owner (0), witness (1), or active (2..9)
// permission currently stored on account.
func (c *Client) GetAccountPermission(ctx context.Context, account string, permissionID int32) (*core.Permission, error) {
	if err := ValidatePermissionID(permissionID); err != nil {
		return nil, err
	}
	acc, err := c.GetAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	var permission *core.Permission
	switch permissionID {
	case OwnerPermissionID:
		permission = acc.GetOwnerPermission()
		if permission == nil {
			permission = defaultOwnerPermission(acc.GetAddress())
		}
	case WitnessPermissionID:
		permission = acc.GetWitnessPermission()
	default:
		for _, candidate := range acc.GetActivePermission() {
			if candidate.GetId() == permissionID {
				permission = candidate
				break
			}
		}
	}
	if permission == nil {
		return nil, fmt.Errorf("%w: account %s has no permission %d", ErrPermissionNotFound, account, permissionID)
	}
	return proto.Clone(permission).(*core.Permission), nil
}

// defaultOwnerPermission mirrors AccountCapsule.getPermissionById(0): legacy
// accounts without stored permission fields are still authorized by their own
// address under owner permission 0.
func defaultOwnerPermission(address []byte) *core.Permission {
	return &core.Permission{
		Type:           core.Permission_Owner,
		Id:             OwnerPermissionID,
		PermissionName: "owner",
		Threshold:      1,
		ParentId:       OwnerPermissionID,
		Keys: []*core.Key{{
			Address: slices.Clone(address),
			Weight:  1,
		}},
	}
}

// ValidatePermissionSigner verifies that signer alone reaches the permission
// threshold and that every required contract type is authorized.
func (c *Client) ValidatePermissionSigner(ctx context.Context, account, signer string, permissionID int32, required ...core.Transaction_Contract_ContractType) error {
	if err := validateTransactionPermissionID(permissionID); err != nil {
		return err
	}
	signerAddress, err := tronutils.DecodeCheck(signer)
	if err != nil {
		return fmt.Errorf("%w: signer %q: %v", ErrInvalidAddress, signer, err)
	}
	permission, err := c.GetAccountPermission(ctx, account, permissionID)
	if err != nil {
		return err
	}
	// java-tron's TransactionCapsule.checkPermission refuses any transaction
	// whose permission id is non-zero and whose permission is not Active,
	// before it ever consults the operations bitmap. Without the same check a
	// permission stored at an active slot but typed Owner - which is also the
	// protobuf zero value, so an absent type lands here too - reads as
	// authorizing every contract type, and this call would approve a signer the
	// chain refuses at broadcast.
	if permissionID != OwnerPermissionID && permission.GetType() != core.Permission_Active {
		return fmt.Errorf("%w: permission %d is %s, not an active permission", ErrInvalidPermission, permissionID, permission.GetType())
	}
	if permission.GetThreshold() <= 0 {
		return fmt.Errorf("%w: permission %d threshold must be positive", ErrInvalidPermission, permissionID)
	}
	var weight int64
	found := false
	for _, key := range permission.GetKeys() {
		if bytes.Equal(key.GetAddress(), signerAddress) {
			found = true
			weight = key.GetWeight()
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: signer %s is not a key of permission %d", ErrPermissionDenied, signer, permissionID)
	}
	if weight < permission.GetThreshold() {
		return fmt.Errorf("%w: signer %s weight %d is below permission %d threshold %d", ErrPermissionDenied, signer, weight, permissionID, permission.GetThreshold())
	}
	for _, typ := range required {
		if !PermissionAllows(permission, typ) {
			return fmt.Errorf("%w: permission %d does not allow %s", ErrPermissionDenied, permissionID, typ)
		}
	}
	return nil
}

// UpdateAccountPermissions builds an unsigned AccountPermissionUpdateContract.
// The request replaces owner, witness, and every active permission atomically.
func (c *Client) UpdateAccountPermissions(ctx context.Context, req AccountPermissionUpdateRequest) (*api.TransactionExtention, error) {
	ownerAddress, err := tronutils.DecodeCheck(req.Account)
	if err != nil {
		return nil, fmt.Errorf("%w: account %q: %v", ErrInvalidAddress, req.Account, err)
	}
	if err := validatePermissionUpdate(req); err != nil {
		return nil, err
	}
	contract := &core.AccountPermissionUpdateContract{
		OwnerAddress: ownerAddress,
		Owner:        proto.Clone(req.Owner).(*core.Permission),
		Actives:      clonePermissions(req.Actives),
	}
	if req.Witness != nil {
		contract.Witness = proto.Clone(req.Witness).(*core.Permission)
	}
	tx, err := c.transport.AccountPermissionUpdate(ctx, contract)
	if err != nil {
		return nil, err
	}
	if err := checkTransaction(tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func clonePermissions(in []*core.Permission) []*core.Permission {
	out := make([]*core.Permission, len(in))
	for i, permission := range in {
		out[i] = proto.Clone(permission).(*core.Permission)
	}
	return out
}

func validatePermissionUpdate(req AccountPermissionUpdateRequest) error {
	if err := validatePermission(req.Owner, core.Permission_Owner); err != nil {
		return err
	}
	if req.Witness != nil {
		if err := validatePermission(req.Witness, core.Permission_Witness); err != nil {
			return err
		}
	}
	if len(req.Actives) == 0 || len(req.Actives) > maxActivePermissions {
		return fmt.Errorf("%w: active permission count must be in [1,%d]", ErrInvalidPermission, maxActivePermissions)
	}
	for _, permission := range req.Actives {
		if err := validatePermission(permission, core.Permission_Active); err != nil {
			return err
		}
	}
	return nil
}

func validatePermission(permission *core.Permission, wantType core.Permission_PermissionType) error {
	if permission == nil {
		return fmt.Errorf("%w: missing %s permission", ErrInvalidPermission, wantType)
	}
	if permission.GetType() != wantType {
		return fmt.Errorf("%w: got type %s, want %s", ErrInvalidPermission, permission.GetType(), wantType)
	}
	if permission.GetParentId() != OwnerPermissionID {
		return fmt.Errorf("%w: parent id must be %d", ErrInvalidPermission, OwnerPermissionID)
	}
	switch wantType {
	case core.Permission_Owner:
		if permission.GetId() != OwnerPermissionID {
			return fmt.Errorf("%w: owner permission id must be %d", ErrInvalidPermission, OwnerPermissionID)
		}
	case core.Permission_Witness:
		if permission.GetId() != WitnessPermissionID {
			return fmt.Errorf("%w: witness permission id must be %d", ErrInvalidPermission, WitnessPermissionID)
		}
	case core.Permission_Active:
		if permission.GetId() != 0 && (permission.GetId() < FirstActivePermissionID || permission.GetId() > LastActivePermissionID) {
			return fmt.Errorf("%w: active permission id must be 0 or in [%d,%d]", ErrInvalidPermission, FirstActivePermissionID, LastActivePermissionID)
		}
	}
	if len(utf16.Encode([]rune(permission.GetPermissionName()))) > maxPermissionNameUTF16Len {
		return fmt.Errorf("%w: permission name exceeds %d UTF-16 code units", ErrInvalidPermission, maxPermissionNameUTF16Len)
	}
	if permission.GetThreshold() <= 0 {
		return fmt.Errorf("%w: threshold must be positive", ErrInvalidPermission)
	}
	if len(permission.GetKeys()) == 0 {
		return fmt.Errorf("%w: permission must contain at least one key", ErrInvalidPermission)
	}
	if wantType == core.Permission_Witness && len(permission.GetKeys()) != 1 {
		return fmt.Errorf("%w: witness permission must contain exactly one key", ErrInvalidPermission)
	}
	var total int64
	addresses := make(map[string]struct{}, len(permission.GetKeys()))
	for _, key := range permission.GetKeys() {
		if len(key.GetAddress()) != tronutils.AddressLength || key.GetAddress()[0] != tronutils.TronBytePrefix || key.GetWeight() <= 0 {
			return fmt.Errorf("%w: invalid permission key", ErrInvalidPermission)
		}
		if total > math.MaxInt64-key.GetWeight() {
			return fmt.Errorf("%w: key weight sum overflows int64", ErrInvalidPermission)
		}
		address := string(key.GetAddress())
		if _, exists := addresses[address]; exists {
			return fmt.Errorf("%w: duplicate permission key", ErrInvalidPermission)
		}
		addresses[address] = struct{}{}
		total += key.GetWeight()
	}
	if total < permission.GetThreshold() {
		return fmt.Errorf("%w: key weight %d is below threshold %d", ErrInvalidPermission, total, permission.GetThreshold())
	}
	if wantType == core.Permission_Active {
		if len(permission.GetOperations()) != permissionOperationsLen {
			return fmt.Errorf("%w: active operations must be %d bytes", ErrInvalidPermission, permissionOperationsLen)
		}
	} else if len(permission.GetOperations()) != 0 {
		return fmt.Errorf("%w: %s permission must not define operations", ErrInvalidPermission, wantType)
	}
	return nil
}

// ValidatePermissionID accepts owner (0), witness (1), and active (2..9) ids.
func ValidatePermissionID(permissionID int32) error {
	if permissionID < OwnerPermissionID || permissionID > LastActivePermissionID {
		return fmt.Errorf("%w: %d", ErrInvalidPermissionID, permissionID)
	}
	return nil
}

func validateTransactionPermissionID(permissionID int32) error {
	if err := ValidatePermissionID(permissionID); err != nil {
		return err
	}
	if permissionID == WitnessPermissionID {
		return fmt.Errorf("%w: witness permission cannot authorize transactions", ErrInvalidPermissionID)
	}
	return nil
}

// SetPermissionID selects which account permission authorizes a transaction
// and refreshes its txid. Call it before signing because Permission_id is part
// of raw_data.
func SetPermissionID(tx *api.TransactionExtention, permissionID int32) error {
	if err := validateTransactionPermissionID(permissionID); err != nil {
		return err
	}
	transaction := tx.GetTransaction()
	if transaction == nil || transaction.GetRawData() == nil {
		return ErrInvalidTransaction
	}
	if len(transaction.GetSignature()) != 0 {
		return fmt.Errorf("%w: permission id must be set before signing", ErrInvalidTransaction)
	}
	contracts := transaction.GetRawData().GetContract()
	if len(contracts) == 0 {
		return ErrInvalidTransaction
	}
	for _, contract := range contracts {
		if contract == nil {
			return ErrInvalidTransaction
		}
	}
	for _, contract := range contracts {
		contract.PermissionId = permissionID
	}
	if err := tx.UpdateHash(); err != nil {
		return fmt.Errorf("%w: update transaction hash: %v", ErrInvalidTransaction, err)
	}
	return nil
}
