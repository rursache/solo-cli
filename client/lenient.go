package client

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// Decoding helpers for endpoints whose scalars may arrive as numbers,
// numeric strings or null. A value of the wrong kind decodes as missing
// instead of failing the whole response

// lenientNumber is nil for null, missing or non-numeric values
type lenientNumber struct{ v *float64 }

func (n *lenientNumber) UnmarshalJSON(b []byte) error {
	n.v = nil
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	var f float64
	if len(b) > 0 && b[0] == '"' {
		var s string
		if json.Unmarshal(b, &s) != nil {
			return nil
		}
		s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
		parsed, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		f = parsed
	} else if json.Unmarshal(b, &f) != nil {
		return nil
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	n.v = &f
	return nil
}

func (n lenientNumber) int() int {
	if n.v == nil {
		return 0
	}
	return int(math.Round(*n.v))
}

type lenientBool struct{ v bool }

func (l *lenientBool) UnmarshalJSON(b []byte) error {
	var v any
	if json.Unmarshal(b, &v) != nil {
		l.v = false
		return nil
	}
	switch x := v.(type) {
	case bool:
		l.v = x
	case string:
		l.v = strings.EqualFold(strings.TrimSpace(x), "true") || strings.TrimSpace(x) == "1"
	case float64:
		l.v = x != 0
	default:
		l.v = false
	}
	return nil
}

// lenientString keeps strings, formats numbers and bools, and is empty for
// null or objects
type lenientString struct{ v string }

func (l *lenientString) UnmarshalJSON(b []byte) error {
	var v any
	if json.Unmarshal(b, &v) != nil {
		l.v = ""
		return nil
	}
	switch x := v.(type) {
	case string:
		l.v = strings.TrimSpace(x)
	case float64:
		l.v = strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		l.v = strconv.FormatBool(x)
	default:
		l.v = ""
	}
	return nil
}

// valueOr0 dereferences an optional amount, nil counting as zero
func valueOr0(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
