package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// decodeTronJSON decodes a java-tron HTTP answer into msg.
//
// java-tron renders protobuf as JSON with its own printer, not protojson, and
// the two disagree in ways that protojson cannot be configured around:
//
//   - bytes are hex, not base64. An even-length hex string is valid base64, so
//     protojson decodes it without complaint into different bytes: a 21-byte
//     address came back as 31 bytes of noise.
//   - a map is an array of {"key": …, "value": …} objects, which protojson
//     refuses outright.
//   - an Any is {"type_url": …, "value": {…}}, not {"@type": …, …}.
//   - int64 is a JSON number, exact only if it is never routed through float64.
//
// The previous approach - rewrite the JSON by field name, converting hex to
// base64 for names on an allow-list, then hand it to protojson - got each of
// these wrong somewhere, because a field name does not say what a field is:
// "extra" is bytes in one message and a string in another, "ref_block_hash"
// was simply never listed, and a receipt carrying a map failed to decode at
// all. Here the message's own descriptor says what each field is, so every
// bytes field is read as hex and nothing else is.
//
// The answer must have been requested without "visible": with it the node
// renders addresses in base58 and some names as UTF-8 text, and those cannot
// be told from hex by looking at them.
//
// Keys the message does not have are skipped: the node adds its own
// (raw_data_hex, visible, a deployed contract's address) next to the proto
// fields.
func decodeTronJSON(body []byte, msg proto.Message) error {
	v, err := parseTronJSON(body)
	if err != nil {
		return err
	}

	return decodeTronMessage(v, msg.ProtoReflect())
}

// parseTronJSON parses a body into generic JSON values, keeping numbers as
// json.Number: a balance above 2^53 SUN does not survive float64.
func parseTronJSON(body []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	return v, nil
}

// decodeTronBlockExtention builds the BlockExtention gRPC returns from a block
// as the HTTP endpoints render it: core.Block plus "blockID", and each
// transaction plus its "txID". java-tron's own conversion marks every
// transaction of a block with Result{result: true}, and so does this one.
func decodeTronBlockExtention(v any) (*api.BlockExtention, error) {
	block := &core.Block{}
	if err := decodeTronMessage(v, block.ProtoReflect()); err != nil {
		return nil, err
	}

	obj := v.(map[string]any) // decodeTronMessage refused anything else
	result := &api.BlockExtention{BlockHeader: block.GetBlockHeader()}

	if id, _ := obj["blockID"].(string); id != "" {
		b, err := hex.DecodeString(id)
		if err != nil {
			return nil, fmt.Errorf("blockID: %w", err)
		}
		result.Blockid = b
	}

	txs, _ := obj["transactions"].([]any)
	for i, tx := range block.GetTransactions() {
		txObj, _ := txs[i].(map[string]any)
		txid, _ := txObj["txID"].(string)
		id, err := hex.DecodeString(txid)
		if err != nil {
			return nil, fmt.Errorf("transactions[%d].txID: %w", i, err)
		}
		result.Transactions = append(result.Transactions, &api.TransactionExtention{
			Transaction: tx,
			Txid:        id,
			Result:      &api.Return{Result: true},
		})
	}

	return result, nil
}

var (
	anyFullName                  = protoreflect.FullName("google.protobuf.Any")
	transactionFullName          = (&core.Transaction{}).ProtoReflect().Descriptor().FullName()
	transactionExtentionFullName = (&api.TransactionExtention{}).ProtoReflect().Descriptor().FullName()
	returnMessageFullName        = (&api.Return{}).ProtoReflect().Descriptor().Fields().ByName("message").FullName()

	// accountAssetIssuedIDFullName is the one bytes field java-tron rewrites
	// after printing: /wallet/getaccount turns asset_issued_ID back into its
	// UTF-8 text (Util.convertOutput), so it is "1000001", not hex.
	accountAssetIssuedIDFullName = (&core.Account{}).ProtoReflect().Descriptor().Fields().ByName("asset_issued_ID").FullName()
)

func tronField(fields protoreflect.FieldDescriptors, key string) protoreflect.FieldDescriptor {
	if fd := fields.ByName(protoreflect.Name(key)); fd != nil {
		return fd
	}
	return fields.ByJSONName(key)
}

func decodeTronMessage(v any, m protoreflect.Message) error {
	obj, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: expected a JSON object, got %T", m.Descriptor().FullName(), v)
	}

	desc := m.Descriptor()
	if desc.FullName() == anyFullName {
		return decodeTronAny(obj, m)
	}

	// A transaction carries its raw_data twice: as JSON and as raw_data_hex,
	// the exact bytes that were hashed and signed. The hex wins whenever it is
	// there - re-encoding the JSON would have to reproduce every byte for the
	// txid to come out the same, and it reaches into contract types this
	// package has never heard of.
	var rawHex string
	if desc.FullName() == transactionFullName {
		rawHex, _ = obj["raw_data_hex"].(string)
	}

	fields := desc.Fields()
	for key, val := range obj {
		fd := tronField(fields, key)
		if fd == nil || val == nil {
			continue
		}
		if rawHex != "" && fd.Name() == "raw_data" {
			continue
		}
		if err := decodeTronField(val, m, fd); err != nil {
			return fmt.Errorf("%s.%s: %w", desc.Name(), key, err)
		}
	}

	// The servlets that answer with a TransactionExtention drop its txid and
	// leave only the transaction's own "txID".
	if desc.FullName() == transactionExtentionFullName && len(m.Get(fields.ByName("txid")).Bytes()) == 0 {
		if tx, ok := obj["transaction"].(map[string]any); ok {
			if txid, _ := tx["txID"].(string); txid != "" {
				b, err := hex.DecodeString(txid)
				if err != nil {
					return fmt.Errorf("transaction.txID: %w", err)
				}
				m.Set(fields.ByName("txid"), protoreflect.ValueOfBytes(b))
			}
		}
	}

	if rawHex != "" {
		raw, err := hex.DecodeString(rawHex)
		if err != nil {
			return fmt.Errorf("decode raw_data_hex: %w", err)
		}
		fd := fields.ByName("raw_data")
		if err := proto.Unmarshal(raw, m.Mutable(fd).Message().Interface()); err != nil {
			return fmt.Errorf("unmarshal raw_data_hex: %w", err)
		}
	}

	return nil
}

func decodeTronField(v any, m protoreflect.Message, fd protoreflect.FieldDescriptor) error {
	switch {
	case fd.IsMap():
		return decodeTronMap(v, m.Mutable(fd).Map(), fd)

	case fd.IsList():
		items, ok := v.([]any)
		if !ok {
			return fmt.Errorf("expected a JSON array, got %T", v)
		}
		list := m.Mutable(fd).List()
		for i, item := range items {
			if fd.Message() != nil {
				elem := list.NewElement()
				if err := decodeTronMessage(item, elem.Message()); err != nil {
					return fmt.Errorf("[%d]: %w", i, err)
				}
				list.Append(elem)
				continue
			}
			value, err := decodeTronScalar(item, fd)
			if err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
			list.Append(value)
		}
		return nil

	case fd.Message() != nil:
		return decodeTronMessage(v, m.Mutable(fd).Message())

	default:
		value, err := decodeTronScalar(v, fd)
		if err != nil {
			return err
		}
		m.Set(fd, value)
		return nil
	}
}

// decodeTronMap reads a map, which java-tron renders as an array of
// {"key", "value"} objects. A JSON object is accepted as well, in case a
// node renders it the standard way.
func decodeTronMap(v any, mp protoreflect.Map, fd protoreflect.FieldDescriptor) error {
	// No map in Tron's schema has message values, so only scalar ones are read.
	set := func(k, val any) error {
		key, err := decodeTronScalar(k, fd.MapKey())
		if err != nil {
			return fmt.Errorf("key: %w", err)
		}
		value, err := decodeTronScalar(val, fd.MapValue())
		if err != nil {
			return fmt.Errorf("[%v]: %w", k, err)
		}
		mp.Set(key.MapKey(), value)
		return nil
	}

	switch entries := v.(type) {
	case []any:
		for i, e := range entries {
			entry, ok := e.(map[string]any)
			if !ok {
				return fmt.Errorf("[%d]: expected a key/value object, got %T", i, e)
			}
			// A zero value is left out like any other default.
			val, ok := entry["value"]
			if !ok {
				val = zeroTronJSON(fd.MapValue())
			}
			if err := set(entry["key"], val); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		for k, val := range entries {
			if err := set(k, val); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("expected a JSON array of key/value objects, got %T", v)
	}
}

func zeroTronJSON(fd protoreflect.FieldDescriptor) any {
	switch fd.Kind() {
	case protoreflect.StringKind, protoreflect.BytesKind:
		return ""
	case protoreflect.BoolKind:
		return false
	default:
		return json.Number("0")
	}
}

// decodeTronAny reads {"type_url": …, "value": …}, where value is either the
// message as JSON or, from the generic printer, its bytes as hex.
func decodeTronAny(obj map[string]any, m protoreflect.Message) error {
	typeURL, _ := obj["type_url"].(string)
	fields := m.Descriptor().Fields()
	m.Set(fields.ByName("type_url"), protoreflect.ValueOfString(typeURL))

	var value []byte
	switch v := obj["value"].(type) {
	case nil:
	case string:
		b, err := hex.DecodeString(v)
		if err != nil {
			return fmt.Errorf("any value: %w", err)
		}
		value = b
	default:
		mt, err := protoregistry.GlobalTypes.FindMessageByURL(typeURL)
		if err != nil {
			return fmt.Errorf("any %q: %w", typeURL, err)
		}
		inner := mt.New()
		if err := decodeTronMessage(v, inner); err != nil {
			return err
		}
		b, err := proto.MarshalOptions{Deterministic: true}.Marshal(inner.Interface())
		if err != nil {
			return fmt.Errorf("any %q: %w", typeURL, err)
		}
		value = b
	}

	m.Set(fields.ByName("value"), protoreflect.ValueOfBytes(value))
	return nil
}

func decodeTronScalar(v any, fd protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		switch b := v.(type) {
		case bool:
			return protoreflect.ValueOfBool(b), nil
		case string:
			parsed, err := strconv.ParseBool(b)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfBool(parsed), nil
		}

	case protoreflect.StringKind:
		if s, ok := v.(string); ok {
			return protoreflect.ValueOfString(s), nil
		}

	case protoreflect.BytesKind:
		if s, ok := v.(string); ok {
			if fd.FullName() == accountAssetIssuedIDFullName {
				return protoreflect.ValueOfBytes([]byte(s)), nil
			}
			b, err := hex.DecodeString(s)
			if err != nil {
				// Return.message is the one bytes field that also arrives as
				// plain text: some error paths print it with toStringUtf8.
				if fd.FullName() == returnMessageFullName {
					return protoreflect.ValueOfBytes([]byte(s)), nil
				}
				return protoreflect.Value{}, fmt.Errorf("bytes field %q is not hex: %w", s, err)
			}
			return protoreflect.ValueOfBytes(b), nil
		}

	case protoreflect.EnumKind:
		switch e := v.(type) {
		case string:
			if ev := fd.Enum().Values().ByName(protoreflect.Name(e)); ev != nil {
				return protoreflect.ValueOfEnum(ev.Number()), nil
			}
			// An enum value newer than this package's schema is printed by
			// number, which is still usable.
			n, err := strconv.ParseInt(e, 10, 32)
			if err != nil {
				return protoreflect.Value{}, fmt.Errorf("unknown %s value %q", fd.Enum().FullName(), e)
			}
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(n)), nil
		case json.Number:
			n, err := e.Int64()
			if err != nil || n < math.MinInt32 || n > math.MaxInt32 {
				return protoreflect.Value{}, fmt.Errorf("enum %s out of range", e)
			}
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(n)), nil
		}

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		n, err := tronInt(v, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(n)), nil

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		n, err := tronInt(v, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(n), nil

	case protoreflect.DoubleKind:
		if n, ok := v.(json.Number); ok {
			f, err := n.Float64()
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfFloat64(f), nil
		}
	}

	// Unsigned and float32 fields do not occur in Tron's schema; should a
	// regenerated one add them, this names the field instead of guessing.
	return protoreflect.Value{}, fmt.Errorf("cannot read %T as %s", v, fd.Kind())
}

// tronInt reads an integer that is a JSON number or, for map keys, a string.
func tronInt(v any, bits int) (int64, error) {
	var s string
	switch n := v.(type) {
	case json.Number:
		s = n.String()
	case string:
		s = n
	default:
		return 0, fmt.Errorf("expected an integer, got %T", v)
	}
	return strconv.ParseInt(s, 10, bits)
}
