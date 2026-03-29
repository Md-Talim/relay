package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"sort"
)

func hashPayload(payload []byte) ([]byte, error) {
	canonical, err := canonicalizeJSON(payload)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(canonical)
	return hash[:], nil
}

func canonicalizeJSON(input []byte) ([]byte, error) {
	var obj any

	if err := json.Unmarshal(input, &obj); err != nil {
		return nil, err
	}

	return canonicalize(obj)
}

func canonicalize(v any) ([]byte, error) {
	switch val := v.(type) {
	case map[string]any:
		// sort keys
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		buf := bytes.NewBufferString("{")
		for i, k := range keys {
			if i > 0 {
				buf.WriteString(",")
			}
			keyBytes, _ := json.Marshal(k)
			buf.Write(keyBytes)
			buf.WriteString(":")

			child, _ := canonicalize(val[k])
			buf.Write(child)
		}
		buf.WriteString("}")
		return buf.Bytes(), nil

	case []any:
		buf := bytes.NewBufferString("[")
		for i, item := range val {
			if i > 0 {
				buf.WriteString(",")
			}
			child, _ := canonicalize(item)
			buf.Write(child)
		}
		buf.WriteString("]")
		return buf.Bytes(), nil

	default:
		return json.Marshal(val)
	}
}
