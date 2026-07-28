package abi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sxwebdev/gotron/schema/pb/core"
)

// LoadContractABI parses a contract's ABI into the protobuf form a deployment
// needs.
//
// It accepts both shapes that occur in practice: the top-level array solc emits,
// and Tron's {"entrys": [...]} envelope, which is what /wallet/getcontract
// returns. Neither survives protojson - an array has no message to unmarshal
// into, and the enums arrive as solc's lowercase "function" / "nonpayable"
// against proto names that are capitalised - so the conversion is done by hand.
func LoadContractABI(jString string) (*core.SmartContract_ABI, error) {
	trimmed := strings.TrimSpace(jString)
	if trimmed == "" {
		return nil, fmt.Errorf("empty contract ABI")
	}

	var entries []abiEntryJSON

	// The shape is dispatched on the first character rather than by trying each
	// in turn, so that anything else - null, a bare string, a single entry
	// object - is rejected outright instead of unmarshalling into nothing and
	// returning an ABI with no entries.
	switch trimmed[0] {
	case '[':
		if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
			return nil, fmt.Errorf("unmarshal contract ABI: %w", err)
		}
	case '{':
		var envelope struct {
			// A pointer so that a missing "entrys" is told apart from an empty
			// one: without it, an object that is a single ABI entry rather than
			// Tron's envelope would parse into an empty ABI and no error.
			Entrys *[]abiEntryJSON `json:"entrys"`
		}
		if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
			return nil, fmt.Errorf("unmarshal contract ABI: %w", err)
		}
		if envelope.Entrys == nil {
			return nil, fmt.Errorf(`unmarshal contract ABI: object has no "entrys" field; ` +
				`pass solc's ABI array or Tron's {"entrys":[...]} envelope`)
		}
		entries = *envelope.Entrys
	default:
		return nil, fmt.Errorf("unmarshal contract ABI: expected a JSON array or object, got %.10q", trimmed)
	}

	result := &core.SmartContract_ABI{}
	for i, e := range entries {
		entry, err := e.toProto()
		if err != nil {
			return nil, fmt.Errorf("contract ABI entry %d: %w", i, err)
		}
		result.Entrys = append(result.Entrys, entry)
	}

	return result, nil
}

// abiEntryJSON is one entry of a Solidity ABI. Constant and Payable are
// pointers because solc stopped emitting them once stateMutability arrived:
// absent and false have to be told apart to know whether to believe them.
type abiEntryJSON struct {
	Type            string         `json:"type"`
	Name            string         `json:"name"`
	Inputs          []abiParamJSON `json:"inputs"`
	Outputs         []abiParamJSON `json:"outputs"`
	StateMutability string         `json:"stateMutability"`
	Anonymous       bool           `json:"anonymous"`
	Constant        *bool          `json:"constant"`
	Payable         *bool          `json:"payable"`
}

type abiParamJSON struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed"`
}

var (
	abiEntryTypes = map[string]core.SmartContract_ABI_Entry_EntryType{
		"function":    core.SmartContract_ABI_Entry_Function,
		"constructor": core.SmartContract_ABI_Entry_Constructor,
		"event":       core.SmartContract_ABI_Entry_Event,
		"fallback":    core.SmartContract_ABI_Entry_Fallback,
		"receive":     core.SmartContract_ABI_Entry_Receive,
		"error":       core.SmartContract_ABI_Entry_Error,
	}

	abiMutabilities = map[string]core.SmartContract_ABI_Entry_StateMutabilityType{
		"pure":       core.SmartContract_ABI_Entry_Pure,
		"view":       core.SmartContract_ABI_Entry_View,
		"nonpayable": core.SmartContract_ABI_Entry_Nonpayable,
		"payable":    core.SmartContract_ABI_Entry_Payable,
	}
)

func (e abiEntryJSON) toProto() (*core.SmartContract_ABI_Entry, error) {
	// The ABI spec makes "function" the default, and pre-0.6 solc relied on it.
	rawType := strings.ToLower(strings.TrimSpace(e.Type))
	if rawType == "" {
		rawType = "function"
	}
	entryType, ok := abiEntryTypes[rawType]
	if !ok {
		return nil, fmt.Errorf("unknown entry type %q", e.Type)
	}

	rawMutability := strings.ToLower(strings.TrimSpace(e.StateMutability))
	mutability := core.SmartContract_ABI_Entry_UnknownMutabilityType
	if rawMutability != "" {
		mutability, ok = abiMutabilities[rawMutability]
		if !ok {
			return nil, fmt.Errorf("unknown state mutability %q", e.StateMutability)
		}
	}

	// Old and new solc describe the same thing in different fields, and the
	// chain stores all three. Fill each from whichever the compiler supplied so
	// that the same contract yields the same entry either way.
	constant := mutability == core.SmartContract_ABI_Entry_View ||
		mutability == core.SmartContract_ABI_Entry_Pure
	if e.Constant != nil {
		constant = *e.Constant
	}

	payable := mutability == core.SmartContract_ABI_Entry_Payable
	if e.Payable != nil {
		payable = *e.Payable
	}

	if mutability == core.SmartContract_ABI_Entry_UnknownMutabilityType {
		switch {
		case payable:
			mutability = core.SmartContract_ABI_Entry_Payable
		case constant:
			mutability = core.SmartContract_ABI_Entry_View
		}
	}

	return &core.SmartContract_ABI_Entry{
		Anonymous:       e.Anonymous,
		Constant:        constant,
		Name:            e.Name,
		Inputs:          abiParams(e.Inputs),
		Outputs:         abiParams(e.Outputs),
		Type:            entryType,
		Payable:         payable,
		StateMutability: mutability,
	}, nil
}

func abiParams(params []abiParamJSON) []*core.SmartContract_ABI_Entry_Param {
	if len(params) == 0 {
		return nil
	}
	out := make([]*core.SmartContract_ABI_Entry_Param, 0, len(params))
	for _, p := range params {
		out = append(out, &core.SmartContract_ABI_Entry_Param{
			Indexed: p.Indexed,
			Name:    p.Name,
			Type:    p.Type,
		})
	}
	return out
}
