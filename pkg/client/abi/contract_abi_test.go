package abi_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sxwebdev/gotron/pkg/client/abi"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// solcOutput is what `solc --abi` emits for a contract with a constructor, a
// mutating function, a view function, a payable function, an event, a custom
// error and a receive fallback. Copied in the shape solc produces it: a
// top-level array with lowercase types and stateMutability.
const solcOutput = `[
  {"inputs":[{"internalType":"uint256","name":"supply","type":"uint256"},
             {"internalType":"address","name":"owner","type":"address"}],
   "stateMutability":"nonpayable","type":"constructor"},
  {"inputs":[{"internalType":"address","name":"to","type":"address"},
             {"internalType":"uint256","name":"value","type":"uint256"}],
   "name":"transfer",
   "outputs":[{"internalType":"bool","name":"","type":"bool"}],
   "stateMutability":"nonpayable","type":"function"},
  {"inputs":[{"internalType":"address","name":"who","type":"address"}],
   "name":"balanceOf",
   "outputs":[{"internalType":"uint256","name":"","type":"uint256"}],
   "stateMutability":"view","type":"function"},
  {"inputs":[],"name":"topUp","outputs":[],"stateMutability":"payable","type":"function"},
  {"anonymous":false,
   "inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},
             {"indexed":true,"internalType":"address","name":"to","type":"address"},
             {"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],
   "name":"Transfer","type":"event"},
  {"inputs":[{"internalType":"uint256","name":"needed","type":"uint256"}],
   "name":"InsufficientBalance","type":"error"},
  {"stateMutability":"payable","type":"receive"}
]`

func entryByName(t *testing.T, parsed *core.SmartContract_ABI, name string) *core.SmartContract_ABI_Entry {
	t.Helper()
	for _, e := range parsed.GetEntrys() {
		if e.GetName() == name {
			return e
		}
	}
	t.Fatalf("no entry named %q", name)
	return nil
}

func TestLoadContractABIParsesSolcOutput(t *testing.T) {
	parsed, err := abi.LoadContractABI(solcOutput)
	require.NoError(t, err)
	require.Len(t, parsed.GetEntrys(), 7)

	t.Run("constructor keeps its argument order", func(t *testing.T) {
		var ctor *core.SmartContract_ABI_Entry
		for _, e := range parsed.GetEntrys() {
			if e.GetType() == core.SmartContract_ABI_Entry_Constructor {
				ctor = e
			}
		}
		require.NotNil(t, ctor)
		require.Len(t, ctor.GetInputs(), 2)
		// Order decides what the encoded constructor arguments mean, so a
		// reordering here would silently deploy a contract with swapped values.
		require.Equal(t, "uint256", ctor.GetInputs()[0].GetType())
		require.Equal(t, "supply", ctor.GetInputs()[0].GetName())
		require.Equal(t, "address", ctor.GetInputs()[1].GetType())
		require.Equal(t, "owner", ctor.GetInputs()[1].GetName())
	})

	t.Run("mutating function", func(t *testing.T) {
		e := entryByName(t, parsed, "transfer")
		require.Equal(t, core.SmartContract_ABI_Entry_Function, e.GetType())
		require.Equal(t, core.SmartContract_ABI_Entry_Nonpayable, e.GetStateMutability())
		require.False(t, e.GetConstant())
		require.False(t, e.GetPayable())
		require.Len(t, e.GetInputs(), 2)
		require.Len(t, e.GetOutputs(), 1)
		require.Equal(t, "bool", e.GetOutputs()[0].GetType())
	})

	t.Run("view function is also marked constant", func(t *testing.T) {
		e := entryByName(t, parsed, "balanceOf")
		require.Equal(t, core.SmartContract_ABI_Entry_View, e.GetStateMutability())
		require.True(t, e.GetConstant())
		require.False(t, e.GetPayable())
	})

	t.Run("payable function is also marked payable", func(t *testing.T) {
		e := entryByName(t, parsed, "topUp")
		require.Equal(t, core.SmartContract_ABI_Entry_Payable, e.GetStateMutability())
		require.True(t, e.GetPayable())
		require.False(t, e.GetConstant())
	})

	t.Run("event keeps the indexed flags", func(t *testing.T) {
		e := entryByName(t, parsed, "Transfer")
		require.Equal(t, core.SmartContract_ABI_Entry_Event, e.GetType())
		require.False(t, e.GetAnonymous())
		require.Len(t, e.GetInputs(), 3)
		// Indexed decides topic-vs-data, so losing it breaks every log filter
		// built from this ABI.
		require.True(t, e.GetInputs()[0].GetIndexed())
		require.True(t, e.GetInputs()[1].GetIndexed())
		require.False(t, e.GetInputs()[2].GetIndexed())
	})

	t.Run("custom error", func(t *testing.T) {
		e := entryByName(t, parsed, "InsufficientBalance")
		require.Equal(t, core.SmartContract_ABI_Entry_Error, e.GetType())
	})

	t.Run("receive", func(t *testing.T) {
		var got *core.SmartContract_ABI_Entry
		for _, e := range parsed.GetEntrys() {
			if e.GetType() == core.SmartContract_ABI_Entry_Receive {
				got = e
			}
		}
		require.NotNil(t, got)
		require.True(t, got.GetPayable())
	})
}

func TestLoadContractABIAcceptsTronEnvelope(t *testing.T) {
	// What /wallet/getcontract returns, so an ABI read off the chain can be
	// fed straight back into a redeployment.
	envelope := `{"entrys":[
      {"inputs":[{"name":"to","type":"address"}],"name":"kill",
       "stateMutability":"Nonpayable","type":"Function"}]}`

	parsed, err := abi.LoadContractABI(envelope)
	require.NoError(t, err)
	require.Len(t, parsed.GetEntrys(), 1)

	e := parsed.GetEntrys()[0]
	// Tron writes the enums capitalised; solc writes them lowercase. Both have
	// to land on the same value.
	require.Equal(t, core.SmartContract_ABI_Entry_Function, e.GetType())
	require.Equal(t, core.SmartContract_ABI_Entry_Nonpayable, e.GetStateMutability())
	require.Equal(t, "kill", e.GetName())
}

func TestLoadContractABILegacySolc(t *testing.T) {
	// solc before 0.5 had no stateMutability, only constant/payable. Both
	// vintages must describe the same contract the same way, or a redeployment
	// from an old artifact records a different ABI than the original.
	cases := []struct {
		name           string
		json           string
		wantMutability core.SmartContract_ABI_Entry_StateMutabilityType
		wantConstant   bool
		wantPayable    bool
	}{
		{
			name:           "constant true",
			json:           `[{"constant":true,"inputs":[],"name":"total","outputs":[],"payable":false,"type":"function"}]`,
			wantMutability: core.SmartContract_ABI_Entry_View,
			wantConstant:   true,
		},
		{
			name:           "payable true",
			json:           `[{"constant":false,"inputs":[],"name":"fund","outputs":[],"payable":true,"type":"function"}]`,
			wantMutability: core.SmartContract_ABI_Entry_Payable,
			wantPayable:    true,
		},
		{
			name:           "neither",
			json:           `[{"constant":false,"inputs":[],"name":"poke","outputs":[],"payable":false,"type":"function"}]`,
			wantMutability: core.SmartContract_ABI_Entry_UnknownMutabilityType,
		},
		{
			name: "explicit flags win over stateMutability",
			// Some toolchains emit both and disagree; the explicit flag is the
			// one the chain stores, so it must not be overwritten.
			json:           `[{"constant":true,"inputs":[],"name":"odd","outputs":[],"stateMutability":"nonpayable","type":"function"}]`,
			wantMutability: core.SmartContract_ABI_Entry_Nonpayable,
			wantConstant:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := abi.LoadContractABI(tc.json)
			require.NoError(t, err)
			require.Len(t, parsed.GetEntrys(), 1)

			e := parsed.GetEntrys()[0]
			require.Equal(t, tc.wantMutability, e.GetStateMutability())
			require.Equal(t, tc.wantConstant, e.GetConstant())
			require.Equal(t, tc.wantPayable, e.GetPayable())
		})
	}
}

func TestLoadContractABIDefaultsTypeToFunction(t *testing.T) {
	// The ABI spec makes "function" the default when type is absent, and old
	// artifacts rely on it. Defaulting to the zero enum instead would store
	// UnknownEntryType and make the method invisible to any decoder.
	parsed, err := abi.LoadContractABI(`[{"name":"ping","inputs":[],"outputs":[]}]`)
	require.NoError(t, err)
	require.Len(t, parsed.GetEntrys(), 1)
	require.Equal(t, core.SmartContract_ABI_Entry_Function, parsed.GetEntrys()[0].GetType())
}

func TestLoadContractABIEmptyArray(t *testing.T) {
	// A contract with no external interface is legal, and an empty ABI is not
	// the same failure as a missing one.
	parsed, err := abi.LoadContractABI(`[]`)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Empty(t, parsed.GetEntrys())
}

func TestLoadContractABIRejects(t *testing.T) {
	cases := []struct {
		name, json, wantErr string
	}{
		{"empty string", "", "empty contract ABI"},
		{"only whitespace", "   \n\t ", "empty contract ABI"},
		{"malformed json", `[{"type":"function"`, "unmarshal"},
		{"malformed envelope", `{"entrys":`, "unmarshal"},
		{"a bare string", `"function"`, "expected a JSON array or object"},
		// null and a single entry object both parse cleanly into nothing, so
		// without an explicit shape check they would come back as a valid ABI
		// with no entries - and deploy a contract nobody can call.
		{"null", `null`, "expected a JSON array or object"},
		{"a number", `42`, "expected a JSON array or object"},
		{"one entry instead of the envelope", `{"type":"function","name":"ping"}`, `no "entrys" field`},
		{"unknown entry type", `[{"type":"modifier","name":"x"}]`, `unknown entry type "modifier"`},
		{"unknown mutability", `[{"type":"function","name":"x","stateMutability":"cheap"}]`, `unknown state mutability "cheap"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := abi.LoadContractABI(tc.json)
			require.Error(t, err)
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, parsed)
		})
	}
}

func TestLoadContractABIReportsTheFailingEntry(t *testing.T) {
	// A real ABI has dozens of entries; without the index the error says only
	// that one of them is wrong.
	_, err := abi.LoadContractABI(
		`[{"type":"function","name":"a"},{"type":"function","name":"b"},{"type":"nonsense","name":"c"}]`,
	)
	require.ErrorContains(t, err, "entry 2")
}
