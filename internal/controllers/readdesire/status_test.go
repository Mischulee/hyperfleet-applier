package readdesire

import (
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-hyperfleet/hyperfleet-applier/pkg/desire"
)

func TestSynced_SetsSyncedReasonAndKubeContent(t *testing.T) {
	content := []byte(`{"kind":"ConfigMap"}`)
	got := synced(desire.ReadStatus{}, 5, content)

	cond := findCondition(got.Status, desire.TypeSuccessful)
	if cond == nil || cond.Status != metav1.ConditionTrue || cond.Reason != desire.ReasonSynced {
		t.Errorf("condition = %+v, want Status=True Reason=%q", cond, desire.ReasonSynced)
	}
	if cond != nil && cond.ObservedGeneration != 5 {
		t.Errorf("condition.ObservedGeneration = %d, want 5", cond.ObservedGeneration)
	}
	if string(got.KubeContent) != string(content) {
		t.Errorf("KubeContent = %s, want %s", got.KubeContent, content)
	}
}

func TestNotFound_ClearsKubeContentAndSetsNotFoundReason(t *testing.T) {
	current := desire.ReadStatus{KubeContent: []byte(`{"stale":"content"}`)}
	got := notFound(current, 6)

	cond := findCondition(got.Status, desire.TypeSuccessful)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != desire.ReasonNotFound {
		t.Errorf("condition = %+v, want Status=False Reason=%q", cond, desire.ReasonNotFound)
	}
	if cond != nil && cond.ObservedGeneration != 6 {
		t.Errorf("condition.ObservedGeneration = %d, want 6", cond.ObservedGeneration)
	}
	if got.KubeContent != nil {
		t.Errorf("KubeContent = %s, want nil: a not-found result must not carry over stale content", got.KubeContent)
	}
}

func TestKubeAPIError_PreservesPriorKubeContent(t *testing.T) {
	priorContent := []byte(`{"last":"known-good"}`)
	current := desire.ReadStatus{KubeContent: priorContent}
	readErr := errors.New("transient list error")

	got := kubeAPIError(current, 7, readErr)

	cond := findCondition(got.Status, desire.TypeSuccessful)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != desire.ReasonKubeAPIError {
		t.Errorf("condition = %+v, want Status=False Reason=%q", cond, desire.ReasonKubeAPIError)
	}
	if cond != nil && cond.Message != readErr.Error() {
		t.Errorf("condition.Message = %q, want %q", cond.Message, readErr.Error())
	}
	if cond != nil && cond.ObservedGeneration != 7 {
		t.Errorf("condition.ObservedGeneration = %d, want 7", cond.ObservedGeneration)
	}
	if string(got.KubeContent) != string(priorContent) {
		t.Errorf("KubeContent = %s, want %s: a transient read failure must not erase the last known mirror",
			got.KubeContent, priorContent)
	}
}

func TestPreCheckFailed_SetsPreCheckFailedReasonAndObservedGeneration(t *testing.T) {
	priorContent := []byte(`{"last":"known-good"}`)
	current := desire.ReadStatus{KubeContent: priorContent}

	got := preCheckFailed(current, 8, "no resource mapping")

	cond := findCondition(got.Status, desire.TypeSuccessful)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != desire.ReasonPreCheckFailed {
		t.Errorf("condition = %+v, want Status=False Reason=%q", cond, desire.ReasonPreCheckFailed)
	}
	if cond != nil && cond.Message != "no resource mapping" {
		t.Errorf("condition.Message = %q, want %q", cond.Message, "no resource mapping")
	}
	if cond != nil && cond.ObservedGeneration != 8 {
		t.Errorf("condition.ObservedGeneration = %d, want 8", cond.ObservedGeneration)
	}
	if string(got.KubeContent) != string(priorContent) {
		t.Errorf("KubeContent = %s, want %s: an unresolvable target must not erase the last known mirror",
			got.KubeContent, priorContent)
	}
}

// TestReadStatusEqual_DetectsKubeContentOnlyDifference proves readStatusEqual
// catches a KubeContent-only change even when conditions are identical -
// conditions.Equal alone would incorrectly treat this as unchanged.
func TestReadStatusEqual_DetectsKubeContentOnlyDifference(t *testing.T) {
	cond := metav1.Condition{Type: desire.TypeSuccessful, Status: metav1.ConditionTrue, Reason: desire.ReasonSynced}
	a := desire.ReadStatus{Status: desire.Status{Conditions: []metav1.Condition{cond}}, KubeContent: []byte(`{"v":1}`)}
	b := desire.ReadStatus{Status: desire.Status{Conditions: []metav1.Condition{cond}}, KubeContent: []byte(`{"v":2}`)}

	if readStatusEqual(a, b) {
		t.Error("readStatusEqual(a, b) = true, want false: KubeContent differs even though conditions match")
	}
	if !readStatusEqual(a, a) {
		t.Error("readStatusEqual(a, a) = false, want true: identical values must compare equal")
	}
}

func TestReadStatusEqual_ContentComparison(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		stored    string
		wantEqual bool
	}{
		{
			name:      managedFieldsField,
			current:   `{"metadata":{"name":"cm","managedFields":[{"manager":"a"}]},"data":{"key":"value"}}`,
			stored:    `{"metadata":{"name":"cm","managedFields":[{"manager":"b"}]},"data":{"key":"value"}}`,
			wantEqual: true,
		},
		{
			name:      resourceVersionField,
			current:   `{"metadata":{"name":"cm","resourceVersion":"1"},"data":{"key":"value"}}`,
			stored:    `{"metadata":{"name":"cm","resourceVersion":"2"},"data":{"key":"value"}}`,
			wantEqual: true,
		},
		{
			name:      "generation metadata",
			current:   `{"metadata":{"name":"cm","generation":1},"data":{"key":"value"}}`,
			stored:    `{"metadata":{"name":"cm","generation":2},"data":{"key":"value"}}`,
			wantEqual: true,
		},
		{
			name:      "meaningful content",
			current:   `{"metadata":{"name":"cm"},"data":{"key":"one"}}`,
			stored:    `{"metadata":{"name":"cm"},"data":{"key":"two"}}`,
			wantEqual: false,
		},
		{
			name:      "distinct large numbers",
			current:   `{"metadata":{"name":"cm"},"data":{"value":9007199254740992}}`,
			stored:    `{"metadata":{"name":"cm"},"data":{"value":9007199254740993}}`,
			wantEqual: false,
		},
		{
			name:      "malformed persisted content",
			current:   `{"metadata":{"name":"cm"}}`,
			stored:    `{"metadata":`,
			wantEqual: false,
		},
		{
			name:      "trailing JSON content",
			current:   `{"metadata":{"name":"cm"}}{"x":1}`,
			stored:    `{"metadata":{"name":"cm"}}`,
			wantEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readStatusEqual(
				desire.ReadStatus{KubeContent: []byte(tt.current)},
				desire.ReadStatus{KubeContent: []byte(tt.stored)},
			)
			if got != tt.wantEqual {
				t.Errorf("readStatusEqual() = %t, want %t", got, tt.wantEqual)
			}
		})
	}
}
