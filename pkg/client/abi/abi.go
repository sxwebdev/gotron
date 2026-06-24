package abi

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sxwebdev/gotron/pkg/tronutils"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"golang.org/x/crypto/sha3"
)

// Param list
type Param map[string]any

// LoadFromJSON string into ABI data
func LoadFromJSON(jString string) ([]Param, error) {
	if len(jString) == 0 {
		return nil, nil
	}
	data := []Param{}
	err := json.Unmarshal([]byte(jString), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Signature of a method
func Signature(method string) []byte {
	// hash method
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(method))
	b := hasher.Sum(nil)
	return b[:4]
}

func convetToAddress(v any) (common.Address, error) {
	switch v := v.(type) {
	case string:
		addr, err := tronutils.DecodeCheck(v)
		if err != nil {
			return common.Address{}, fmt.Errorf("invalid address %s: %+v", v, err)
		}
		return common.BytesToAddress(addr[len(addr)-20:]), nil
	}
	return common.Address{}, fmt.Errorf("invalid address %v", v)
}

func convertToInt(ty eABI.Type, v any) (any, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("invalid integer %v", v)
	}
	if ty.T == eABI.IntTy && ty.Size <= 64 {
		tmp, err := strconv.ParseInt(s, 10, ty.Size)
		if err != nil {
			return nil, fmt.Errorf("invalid int%d %q: %w", ty.Size, s, err)
		}
		switch ty.Size {
		case 8:
			v = int8(tmp)
		case 16:
			v = int16(tmp)
		case 32:
			v = int32(tmp)
		case 64:
			v = int64(tmp)
		}
	} else if ty.T == eABI.UintTy && ty.Size <= 64 {
		tmp, err := strconv.ParseUint(s, 10, ty.Size)
		if err != nil {
			return nil, fmt.Errorf("invalid uint%d %q: %w", ty.Size, s, err)
		}
		switch ty.Size {
		case 8:
			v = uint8(tmp)
		case 16:
			v = uint16(tmp)
		case 32:
			v = uint32(tmp)
		case 64:
			v = uint64(tmp)
		}
	} else {
		var set bool
		// check for hex char
		if strings.HasPrefix(s, "0x") {
			v, set = new(big.Int).SetString(s[2:], 16)
		} else {
			v, set = new(big.Int).SetString(s, 10)
		}
		if !set {
			return nil, fmt.Errorf("invalid integer %q", s)
		}
	}
	return v, nil
}

// GetPaddedParam from struct
func GetPaddedParam(param []Param) ([]byte, error) {
	values := make([]any, 0)
	arguments := eABI.Arguments{}

	for _, p := range param {
		if len(p) != 1 {
			return nil, fmt.Errorf("invalid param %+v", p)
		}
		for k, v := range p {
			ty, err := eABI.NewType(k, "", nil)
			if err != nil {
				return nil, fmt.Errorf("invalid param %+v: %+v", p, err)
			}
			arguments = append(
				arguments,
				eABI.Argument{
					Name:    "",
					Type:    ty,
					Indexed: false,
				},
			)

			if ty.T == eABI.SliceTy || ty.T == eABI.ArrayTy {
				if ty.Elem.T == eABI.AddressTy {
					tmp, ok := v.([]any)
					if !ok {
						return nil, fmt.Errorf("unable to convert array of addresses %+v", p)
					}
					v = make([]common.Address, 0)
					for i := range tmp {
						addr, err := convetToAddress(tmp[i])
						if err != nil {
							return nil, err
						}
						v = append(v.([]common.Address), addr)
					}
				}

				if (ty.Elem.T == eABI.IntTy || ty.Elem.T == eABI.UintTy) &&
					ty.Elem.Size > 64 {
					// Accept both []string and the []interface{} that JSON
					// decoding (LoadFromJSON) produces.
					var tmpSlice []string
					switch arr := v.(type) {
					case []string:
						tmpSlice = arr
					case []any:
						for _, e := range arr {
							s, ok := e.(string)
							if !ok {
								return nil, fmt.Errorf("unable to convert array of uints %+v", p)
							}
							tmpSlice = append(tmpSlice, s)
						}
					default:
						return nil, fmt.Errorf("unable to convert array of uints %+v", p)
					}
					tmp := make([]*big.Int, 0, len(tmpSlice))
					for i := range tmpSlice {
						var value *big.Int
						var ok bool
						// check for hex char
						if strings.HasPrefix(tmpSlice[i], "0x") {
							value, ok = new(big.Int).SetString(tmpSlice[i][2:], 16)
						} else {
							value, ok = new(big.Int).SetString(tmpSlice[i], 10)
						}
						if !ok {
							return nil, fmt.Errorf("invalid integer %q", tmpSlice[i])
						}
						tmp = append(tmp, value)
					}
					v = tmp
				}
			}
			if ty.T == eABI.AddressTy {
				if v, err = convetToAddress(v); err != nil {
					return nil, err
				}
			}
			if (ty.T == eABI.IntTy || ty.T == eABI.UintTy) && reflect.TypeOf(v).Kind() == reflect.String {
				if v, err = convertToInt(ty, v); err != nil {
					return nil, err
				}
			}

			if ty.T == eABI.BytesTy || ty.T == eABI.FixedBytesTy {
				var err error
				if v, err = convertToBytes(ty, v); err != nil {
					return nil, err
				}
			}

			values = append(values, v)
		}
	}
	// convert params to bytes
	return arguments.PackValues(values)
}

func convertToBytes(ty eABI.Type, v any) (any, error) {
	// if string
	if data, ok := v.(string); ok {
		// convert from hex string
		dataBytes, err := hex.DecodeString(data)
		if err != nil {
			// try with base64
			dataBytes, err = base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, err
			}
		}
		// if array and size == 0
		if ty.T == eABI.BytesTy || ty.Size == 0 {
			return dataBytes, nil
		}
		if len(dataBytes) != ty.Size {
			return nil, fmt.Errorf("invalid size: %d/%d", ty.Size, len(dataBytes))
		}
		// Build a [ty.Size]byte array for any fixed-bytes size (bytes1..bytes32),
		// which is the value type go-ethereum's PackValues expects.
		arr := reflect.New(reflect.ArrayOf(ty.Size, reflect.TypeFor[byte]())).Elem()
		reflect.Copy(arr, reflect.ValueOf(dataBytes))
		return arr.Interface(), nil
	}
	return v, nil
}

// Pack data into bytes
func Pack(method string, param []Param) ([]byte, error) {
	signature := Signature(method)
	pBytes, err := GetPaddedParam(param)
	if err != nil {
		return nil, err
	}
	signature = append(signature, pBytes...)
	return signature, nil
}

// GetParser return output method parser arguments from ABI
func GetParser(ABI *core.SmartContract_ABI, method string) (eABI.Arguments, error) {
	arguments := eABI.Arguments{}
	for _, entry := range ABI.Entrys {
		if entry.Name == method {
			for _, out := range entry.Outputs {
				ty, err := eABI.NewType(out.Type, "", nil)
				if err != nil {
					return nil, fmt.Errorf("invalid param %s: %+v", out.Type, err)
				}
				arguments = append(arguments, eABI.Argument{
					Name:    out.Name,
					Type:    ty,
					Indexed: out.Indexed,
				})
			}
			return arguments, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

// GetInputsParser returns input method parser arguments from ABI
func GetInputsParser(ABI *core.SmartContract_ABI, method string) (eABI.Arguments, error) {
	arguments := eABI.Arguments{}
	for _, entry := range ABI.Entrys {
		if entry.Name == method {
			for _, out := range entry.Inputs {
				ty, err := eABI.NewType(out.Type, "", nil)
				if err != nil {
					return nil, fmt.Errorf("invalid param %s: %+v", out.Type, err)
				}
				arguments = append(arguments, eABI.Argument{
					Name:    out.Name,
					Type:    ty,
					Indexed: out.Indexed,
				})
			}
			return arguments, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
