package readdesire

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-hyperfleet/hyperfleet-applier/internal/controllers/util"
	"github.com/openshift-hyperfleet/hyperfleet-applier/pkg/desire"
)

const (
	metadataField        = "metadata"
	managedFieldsField   = "managedFields"
	resourceVersionField = "resourceVersion"
	generationField      = "generation"
)

// synced returns a copy of status with the Successful condition set to True,
// reason Synced, and KubeContent set to the mirrored object's content.
func synced(status desire.ReadStatus, kubeContent []byte) desire.ReadStatus {
	return withReadCondition(status, metav1.Condition{
		Type:   desire.TypeSuccessful,
		Status: metav1.ConditionTrue,
		Reason: desire.ReasonSynced,
	}, kubeContent)
}

// notFound returns a copy of status with the Successful condition set to
// False, reason NotFound, and KubeContent cleared: the target does not
// currently exist, so there is nothing to mirror - the next observed
// creation resolves it.
func notFound(status desire.ReadStatus) desire.ReadStatus {
	return withReadCondition(status, metav1.Condition{
		Type:   desire.TypeSuccessful,
		Status: metav1.ConditionFalse,
		Reason: desire.ReasonNotFound,
	}, nil)
}

// kubeAPIError returns a copy of status with the Successful condition set to
// False, reason KubeAPIError. KubeContent is left as it was on status: a
// transient read failure doesn't invalidate the last known mirrored content.
func kubeAPIError(status desire.ReadStatus, err error) desire.ReadStatus {
	return withReadCondition(status, metav1.Condition{
		Type:    desire.TypeSuccessful,
		Status:  metav1.ConditionFalse,
		Reason:  desire.ReasonKubeAPIError,
		Message: err.Error(),
	}, status.KubeContent)
}

// preCheckFailed returns a copy of status with the Successful condition set
// to False, reason PreCheckFailed: the desire's target could not even be
// resolved (e.g. no REST mapping for its Group/Resource), so no informer
// could be started for it at all. Mirrors applydesire's preCheckFailed -
// same reason, same "the call could not be executed at all" meaning.
// KubeContent is left as it was: an unresolvable target doesn't invalidate
// previously-mirrored content from before it became unresolvable.
func preCheckFailed(status desire.ReadStatus, msg string) desire.ReadStatus {
	return withReadCondition(status, metav1.Condition{
		Type:    desire.TypeSuccessful,
		Status:  metav1.ConditionFalse,
		Reason:  desire.ReasonPreCheckFailed,
		Message: msg,
	}, status.KubeContent)
}

// withReadCondition returns a copy of status with cond set on its embedded
// Status and kubeContent as its KubeContent - util.WithCondition alone
// only covers the Status half of desire.ReadStatus.
func withReadCondition(status desire.ReadStatus, cond metav1.Condition, kubeContent []byte) desire.ReadStatus {
	return desire.ReadStatus{
		Status:      util.WithCondition(status.Status, cond),
		KubeContent: kubeContent,
	}
}

// readStatusEqual reports whether a and b are equal for the purpose of
// suppressing a redundant status write: same conditions (via util.Equal)
// and semantically identical KubeContent. Kubernetes-managed metadata is
// ignored when comparing content, but the raw observed content remains in
// the status that is written.
func readStatusEqual(a, b desire.ReadStatus) bool {
	return util.Equal(a.Status, b.Status) && semanticContentEqual(a.KubeContent, b.KubeContent)
}

func semanticContentEqual(a, b []byte) bool {
	if bytes.Equal(a, b) {
		return true
	}

	canonicalA, okA := canonicalContent(a)
	canonicalB, okB := canonicalContent(b)
	return okA && okB && bytes.Equal(canonicalA, canonicalB)
}

// canonicalContent returns canonical JSON with Kubernetes-managed metadata
// removed. An invalid persisted value is deliberately treated as different by
// the caller, allowing the next valid observation to replace it.
func canonicalContent(content []byte) ([]byte, bool) {
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, false
	}
	if metadata, ok := object[metadataField].(map[string]any); ok {
		delete(metadata, managedFieldsField)
		delete(metadata, resourceVersionField)
		delete(metadata, generationField)
	}
	normalized, err := json.Marshal(object)
	if err != nil {
		return nil, false
	}
	return normalized, true
}
