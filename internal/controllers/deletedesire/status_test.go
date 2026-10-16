package deletedesire

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

func TestDeleted_SetsDeletedReasonAndObservedGeneration(t *testing.T) {
	out := deleted(desire.Status{}, 1)

	c := findCondition(out, desire.TypeSuccessful)
	if c == nil || c.Status != metav1.ConditionTrue || c.Reason != desire.ReasonDeleted {
		t.Errorf("deleted() did not set the Deleted condition, got %+v", c)
	}
	if c != nil && c.ObservedGeneration != 1 {
		t.Errorf("deleted() observed generation = %d, want 1", c.ObservedGeneration)
	}
}

func TestWaitingForDeletion_SetsWaitingForDeletionReasonAndObservedGeneration(t *testing.T) {
	out := waitingForDeletion(desire.Status{}, 2, "deletion timestamp: ...")

	c := findCondition(out, desire.TypeSuccessful)
	if c == nil || c.Status != metav1.ConditionFalse || c.Reason != desire.ReasonWaitingForDeletion {
		t.Errorf("waitingForDeletion() did not set the WaitingForDeletion condition, got %+v", c)
	}
	if c != nil && c.Message != "deletion timestamp: ..." {
		t.Errorf("waitingForDeletion() message = %q, want %q", c.Message, "deletion timestamp: ...")
	}
	if c != nil && c.ObservedGeneration != 2 {
		t.Errorf("waitingForDeletion() observed generation = %d, want 2", c.ObservedGeneration)
	}
}

func TestDeleteFailed_SetsKubeAPIErrorReasonAndObservedGeneration(t *testing.T) {
	kubeErr := errors.New("apiserver unavailable")
	out := deleteFailed(desire.Status{}, 3, kubeErr)

	c := findCondition(out, desire.TypeSuccessful)
	if c == nil || c.Status != metav1.ConditionFalse || c.Reason != desire.ReasonKubeAPIError {
		t.Errorf("deleteFailed() did not set the KubeAPIError condition, got %+v", c)
	}
	if c != nil && c.Message != kubeErr.Error() {
		t.Errorf("deleteFailed() message = %q, want %q", c.Message, kubeErr.Error())
	}
	if c != nil && c.ObservedGeneration != 3 {
		t.Errorf("deleteFailed() observed generation = %d, want 3", c.ObservedGeneration)
	}
}

func TestPreCheckFailed_SetsPreCheckFailedReasonAndObservedGeneration(t *testing.T) {
	out := preCheckFailed(desire.Status{}, 4, "bad mapping")

	c := findCondition(out, desire.TypeSuccessful)
	if c == nil || c.Status != metav1.ConditionFalse || c.Reason != desire.ReasonPreCheckFailed {
		t.Errorf("preCheckFailed() did not set the PreCheckFailed condition, got %+v", c)
	}
	if c != nil && c.Message != "bad mapping" {
		t.Errorf("preCheckFailed() message = %q, want %q", c.Message, "bad mapping")
	}
	if c != nil && c.ObservedGeneration != 4 {
		t.Errorf("preCheckFailed() observed generation = %d, want 4", c.ObservedGeneration)
	}
}
