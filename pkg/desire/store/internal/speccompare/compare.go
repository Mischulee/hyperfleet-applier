// Package speccompare compares Apply specs consistently across store backends.
package speccompare

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/openshift-hyperfleet/hyperfleet-applier/pkg/desire"
)

// Equal reports whether two Apply specs contain the same JSON value. Object
// key order and whitespace do not change the desired resource. UseNumber
// preserves numeric literals so large integers cannot compare equal through
// float64 rounding; distinct numeric spellings are conservatively treated as
// changes. Invalid stored JSON is never considered equal to valid input.
func Equal(a, b desire.ApplySpec) bool {
	if bytes.Equal(a.KubeContent, b.KubeContent) {
		return json.Valid(a.KubeContent)
	}
	left, ok := decode(a.KubeContent)
	if !ok {
		return false
	}
	right, ok := decode(b.KubeContent)
	return ok && reflect.DeepEqual(left, right)
}

func decode(content json.RawMessage) (any, bool) {
	if !json.Valid(content) {
		return nil, false
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	return value, true
}
