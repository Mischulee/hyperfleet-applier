package speccompare

import (
	"encoding/json"
	"testing"

	"github.com/openshift-hyperfleet/hyperfleet-applier/pkg/desire"
)

func TestEqual(t *testing.T) {
	const jsonV1 = `{"v":1}`
	for _, tc := range []struct {
		name  string
		left  string
		right string
		want  bool
	}{
		{"identical", jsonV1, jsonV1, true},
		{"whitespace and object key order", `{"v":1,"nested":{"a":true,"b":[1,2]}}`,
			` { "nested": { "b": [1, 2], "a": true }, "v": 1 } `, true},
		{"equivalent string escapes", `{"text":"a"}`, `{"text":"\u0061"}`, true},
		{"changed value", jsonV1, `{"v":2}`, false},
		{"changed metadata", `{"metadata":{"labels":{"a":"1"}}}`,
			`{"metadata":{"labels":{"a":"2"}}}`, false},
		{"array order matters", `[1,2]`, `[2,1]`, false},
		{"large integers retain precision", `{"v":9007199254740992}`,
			`{"v":9007199254740993}`, false},
		{"numeric spelling conservatively differs", jsonV1, `{"v":1.0}`, false},
		{"invalid stored content", `{`, jsonV1, false},
		{"identical invalid content", `{`, `{`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := desire.ApplySpec{KubeContent: json.RawMessage(tc.left)}
			right := desire.ApplySpec{KubeContent: json.RawMessage(tc.right)}
			if got := Equal(left, right); got != tc.want {
				t.Errorf("Equal(%q, %q) = %t, want %t", tc.left, tc.right, got, tc.want)
			}
			if got := Equal(right, left); got != tc.want {
				t.Errorf("Equal(%q, %q) = %t, want %t", tc.right, tc.left, got, tc.want)
			}
		})
	}
}
