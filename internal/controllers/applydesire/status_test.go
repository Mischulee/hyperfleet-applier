package applydesire

import (
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-hyperfleet/hyperfleet-applier/pkg/desire"
)

func findCondition(status desire.Status, condType string) *metav1.Condition {
	for i := range status.Conditions {
		if status.Conditions[i].Type == condType {
			return &status.Conditions[i]
		}
	}
	return nil
}

func TestApplied_ReplacesPriorFailedCondition(t *testing.T) {
	failed := preCheckFailed(desire.Status{}, 1, "previously broken")

	out := applied(failed, 2)

	if len(out.Conditions) != 1 {
		t.Fatalf(
			"applied() over a failed status produced %d conditions, want exactly 1: %+v",
			len(out.Conditions), out.Conditions,
		)
	}
	c := findCondition(out, desire.TypeSuccessful)
	if c == nil || c.Status != metav1.ConditionTrue || c.Reason != desire.ReasonApplied || c.ObservedGeneration != 2 {
		t.Errorf("applied() did not overwrite the prior condition, got %+v", c)
	}
}

func TestApplyFailed_SetsKubeAPIErrorReason(t *testing.T) {
	kubeErr := errors.New("apiserver unavailable")
	out := applyFailed(desire.Status{}, 3, kubeErr)

	c := findCondition(out, desire.TypeSuccessful)
	if c == nil {
		t.Fatalf("applyFailed() did not set the %q condition, got %+v", desire.TypeSuccessful, out.Conditions)
	}
	if c.Status != metav1.ConditionFalse || c.Reason != desire.ReasonKubeAPIError {
		t.Errorf("applyFailed() did not set the KubeAPIError condition, got %+v", c)
	}
	if c.Message != kubeErr.Error() {
		t.Errorf("applyFailed() message = %q, want %q", c.Message, kubeErr.Error())
	}
	if c.ObservedGeneration != 3 {
		t.Errorf("applyFailed() observed generation = %d, want 3", c.ObservedGeneration)
	}
}

func TestPreCheckFailed_SetsPreCheckFailedReason(t *testing.T) {
	out := preCheckFailed(desire.Status{}, 4, "bad manifest")

	c := findCondition(out, desire.TypeSuccessful)
	if c == nil {
		t.Fatalf("preCheckFailed() did not set the %q condition, got %+v", desire.TypeSuccessful, out.Conditions)
	}
	if c.Status != metav1.ConditionFalse || c.Reason != desire.ReasonPreCheckFailed {
		t.Errorf("preCheckFailed() did not set the PreCheckFailed condition, got %+v", c)
	}
	if c.Message != "bad manifest" {
		t.Errorf("preCheckFailed() message = %q, want %q", c.Message, "bad manifest")
	}
	if c.ObservedGeneration != 4 {
		t.Errorf("preCheckFailed() observed generation = %d, want 4", c.ObservedGeneration)
	}
}
