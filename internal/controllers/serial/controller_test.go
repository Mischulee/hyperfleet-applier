package serial

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/openshift-hyperfleet/hyperfleet-applier/internal/controllers/applydesire"
	"github.com/openshift-hyperfleet/hyperfleet-applier/internal/controllers/deletedesire"
	"github.com/openshift-hyperfleet/hyperfleet-applier/pkg/desire"
	"github.com/openshift-hyperfleet/hyperfleet-applier/pkg/desire/store/memory"
)

const (
	testManagementCluster = "test-cluster"
	testOwner             = "test-owner"
)

type applyLister struct {
	list func(context.Context) ([]desire.ApplyDesire, error)
}

func (l applyLister) ListApplyDesires(ctx context.Context, _ string) ([]desire.ApplyDesire, error) {
	return l.list(ctx)
}

func (applyLister) GetApplyDesire(context.Context, desire.Identity) (desire.ApplyDesire, error) {
	return desire.ApplyDesire{}, desire.ErrNotFound
}

type deleteLister struct {
	list func(context.Context) ([]desire.DeleteDesire, error)
}

func (l deleteLister) ListDeleteDesires(ctx context.Context, _ string) ([]desire.DeleteDesire, error) {
	return l.list(ctx)
}

func (deleteLister) GetDeleteDesire(context.Context, desire.Identity) (desire.DeleteDesire, error) {
	return desire.DeleteDesire{}, desire.ErrNotFound
}

func TestStart_ReconcilesApplyThenDeleteAndStopsCleanly(t *testing.T) {
	var order []string
	applyEntered := make(chan struct{})
	deleteEntered := make(chan struct{})

	apply := applydesire.New(applyLister{list: func(context.Context) ([]desire.ApplyDesire, error) {
		order = append(order, "apply")
		close(applyEntered)
		return nil, nil
	}}, nil, nil, nil, testManagementCluster)
	del := deletedesire.New(deleteLister{list: func(context.Context) ([]desire.DeleteDesire, error) {
		order = append(order, "delete")
		close(deleteEntered)
		return nil, nil
	}}, nil, nil, nil, testManagementCluster)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		done <- New(time.Hour, apply, del).Start(ctx)
	}()

	select {
	case <-applyEntered:
	case <-time.After(time.Second):
		t.Fatal("Start did not begin the apply pass immediately")
	}
	select {
	case <-deleteEntered:
	case <-time.After(time.Second):
		t.Fatal("Start did not run delete after apply")
	}

	if len(order) != 2 || order[0] != "apply" || order[1] != "delete" {
		t.Fatalf("pass order = %v, want [apply delete]", order)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() error = %v, want nil after caller cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not stop after context cancellation")
	}
}

func TestStart_ApplyFailureStillRunsDelete(t *testing.T) {
	deleteEntered := make(chan struct{})
	apply := applydesire.New(applyLister{list: func(context.Context) ([]desire.ApplyDesire, error) {
		return nil, errors.New("list apply desires")
	}}, nil, nil, nil, testManagementCluster)
	del := deletedesire.New(deleteLister{list: func(context.Context) ([]desire.DeleteDesire, error) {
		close(deleteEntered)
		return nil, nil
	}}, nil, nil, nil, testManagementCluster)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		_ = New(time.Hour, apply, del).Start(ctx)
	}()

	select {
	case <-deleteEntered:
	case <-time.After(time.Second):
		t.Fatal("delete pass did not run after apply failed")
	}
}

func TestStart_CanceledApplySkipsDelete(t *testing.T) {
	deleteEntered := make(chan struct{})
	apply := applydesire.New(applyLister{list: func(ctx context.Context) ([]desire.ApplyDesire, error) {
		return nil, ctx.Err()
	}}, nil, nil, nil, testManagementCluster)
	del := deletedesire.New(deleteLister{list: func(context.Context) ([]desire.DeleteDesire, error) {
		close(deleteEntered)
		return nil, nil
	}}, nil, nil, nil, testManagementCluster)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() {
		done <- New(time.Hour, apply, del).Start(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() error = %v, want nil after caller cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not return after a canceled apply pass")
	}
	select {
	case <-deleteEntered:
		t.Fatal("delete pass ran after apply was aborted by cancellation")
	default:
	}
}

func TestStart_DeletesInTheSamePassAfterApplySuperseded(t *testing.T) {
	store := memory.New()
	const name = "cm-supersede"
	applyID := configMapIdentity(desire.TypeApply, name)
	deleteID := configMapIdentity(desire.TypeDelete, name)
	content := newConfigMapContent(t, name, "default", map[string]string{"key": "desired"})
	if _, err := store.CreateApplyDesire(t.Context(), desire.ApplyDesire{
		Identity: applyID, Owner: testOwner, Spec: desire.ApplySpec{KubeContent: content},
	}); err != nil {
		t.Fatalf("seed ApplyDesire: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("register core types: %v", err)
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme, nil,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}},
	)
	dyn.PrependReactor("patch", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		if _, createErr := store.CreateDeleteDesire(context.Background(), desire.DeleteDesire{
			Identity: deleteID, Owner: testOwner,
		}); createErr != nil && !errors.Is(createErr, desire.ErrAlreadyExists) {
			return true, nil, createErr
		}
		return false, nil, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		done <- New(
			time.Hour,
			applydesire.New(store, store, dyn, newTestMapper(), testManagementCluster),
			deletedesire.New(store, store, dyn, newTestMapper(), testManagementCluster),
		).Start(ctx)
	}()

	waitForDeleteReason(t, store, deleteID, desire.ReasonDeleted)
	_, err := dyn.Resource(configMapGVR).Namespace("default").Get(t.Context(), name, metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("object after the pass = %v, want NotFound", err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() error = %v, want nil after caller cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not stop after context cancellation")
	}
}

func configMapIdentity(typ desire.DesireType, name string) desire.Identity {
	return desire.Identity{
		ManagementCluster: testManagementCluster,
		Type:              typ,
		Resource:          "configmaps",
		Namespace:         "default",
		Name:              name,
	}
}

func newConfigMapContent(t *testing.T, name, namespace string, data map[string]string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": name, "namespace": namespace},
		"data":       data,
	})
	if err != nil {
		t.Fatalf("marshal configmap content: %v", err)
	}
	return raw
}

var configMapGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

func newTestMapper() meta.RESTMapper {
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{{Version: "v1"}})
	mapper.AddSpecific(
		schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"},
		configMapGVR,
		schema.GroupVersionResource{Version: "v1", Resource: "configmap"},
		meta.RESTScopeNamespace,
	)
	return mapper
}

func waitForDeleteReason(t *testing.T, store *memory.Store, id desire.Identity, want string) {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var last desire.Status
	for {
		got, err := store.GetDeleteDesire(t.Context(), id)
		if err != nil && !errors.Is(err, desire.ErrNotFound) {
			t.Fatalf("GetDeleteDesire: %v", err)
		}
		last = got.Status
		for _, condition := range got.Status.Conditions {
			if condition.Type == desire.TypeSuccessful && condition.Reason == want {
				return
			}
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatalf("waiting for delete reason %s; last status: %+v", want, last)
		}
	}
}
