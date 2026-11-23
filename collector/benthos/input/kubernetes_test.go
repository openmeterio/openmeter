package input

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"
)

func newTestPod(namespace, name string, phase corev1.PodPhase, podLabels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Labels: podLabels},
		Status:     corev1.PodStatus{Phase: phase},
	}
}

func newTestKubernetesResourcesInput(t *testing.T, in *kubernetesResourcesInput, objects ...client.Object) *kubernetesResourcesInput {
	t.Helper()

	if in == nil {
		in = &kubernetesResourcesInput{}
	}

	in.resourceType = "pod"
	in.inFlight = make(chan struct{}, 1)
	in.resources = service.MockResources()
	in.logger = in.resources.Logger()
	in.client = fake.NewClientBuilder().WithObjects(objects...).Build()

	if len(in.namespaces) == 0 {
		in.namespaces = []string{corev1.NamespaceAll}
	}

	return in
}

// readPodNames reads one batch and returns the names of the pods in it.
func readPodNames(t *testing.T, in *kubernetesResourcesInput) ([]string, service.MessageBatch, service.AckFunc) {
	t.Helper()

	batch, ack, err := in.ReadBatch(t.Context())
	require.NoError(t, err)

	names := lo.Map(batch, func(msg *service.Message, _ int) string {
		structured, err := msg.AsStructured()
		require.NoError(t, err)

		metadata, ok := structured.(map[string]any)["metadata"].(map[string]any)
		require.True(t, ok)

		name, ok := metadata["name"].(string)
		require.True(t, ok)

		return name
	})

	return names, batch, ack
}

func TestKubernetesResourcesInputPodPhases(t *testing.T) {
	pods := []client.Object{
		newTestPod("default", "pending", corev1.PodPending, nil),
		newTestPod("default", "running", corev1.PodRunning, nil),
		newTestPod("default", "succeeded", corev1.PodSucceeded, nil),
		newTestPod("default", "failed", corev1.PodFailed, nil),
		newTestPod("default", "unknown", corev1.PodUnknown, nil),
	}

	tests := []struct {
		name  string
		input *kubernetesResourcesInput
		want  []string
	}{
		{
			name: "running pods only by default",
			want: []string{"running"},
		},
		{
			name:  "pending pods",
			input: &kubernetesResourcesInput{includePendingPods: true},
			want:  []string{"pending", "running"},
		},
		{
			name:  "terminated pods",
			input: &kubernetesResourcesInput{includeTerminatedPods: true},
			want:  []string{"failed", "running", "succeeded"},
		},
		{
			name:  "pending and terminated pods",
			input: &kubernetesResourcesInput{includePendingPods: true, includeTerminatedPods: true},
			want:  []string{"failed", "pending", "running", "succeeded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := newTestKubernetesResourcesInput(t, tt.input, pods...)

			names, _, _ := readPodNames(t, in)
			assert.ElementsMatch(t, tt.want, names)
		})
	}
}

func TestKubernetesResourcesInputDeletedPodSelection(t *testing.T) {
	selector, err := labels.Parse("app=seed")
	require.NoError(t, err)

	seed := map[string]string{"app": "seed"}

	tests := []struct {
		name     string
		input    *kubernetesResourcesInput
		isLeader *bool
		deleted  any
		want     []string
	}{
		{
			name:    "pod",
			deleted: newTestPod("default", "pod", corev1.PodRunning, nil),
			want:    []string{"pod"},
		},
		{
			name:    "pod the cache missed the delete event of",
			deleted: toolscache.DeletedFinalStateUnknown{Key: "default/pod", Obj: newTestPod("default", "pod", corev1.PodRunning, nil)},
			want:    []string{"pod"},
		},
		{
			name:    "not a pod",
			deleted: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node"}},
		},
		{
			name:    "pod in a listed namespace",
			input:   &kubernetesResourcesInput{namespaces: []string{"default"}},
			deleted: newTestPod("default", "pod", corev1.PodRunning, nil),
			want:    []string{"pod"},
		},
		{
			name:    "pod in another namespace",
			input:   &kubernetesResourcesInput{namespaces: []string{"default"}},
			deleted: newTestPod("other", "pod", corev1.PodRunning, nil),
		},
		{
			name:    "pod matching the label selector",
			input:   &kubernetesResourcesInput{labelSelector: selector},
			deleted: newTestPod("default", "pod", corev1.PodRunning, seed),
			want:    []string{"pod"},
		},
		{
			name:    "pod not matching the label selector",
			input:   &kubernetesResourcesInput{labelSelector: selector},
			deleted: newTestPod("default", "pod", corev1.PodRunning, nil),
		},
		{
			name:     "not the leader",
			isLeader: lo.ToPtr(false),
			deleted:  newTestPod("default", "pod", corev1.PodRunning, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := newTestKubernetesResourcesInput(t, tt.input)

			if tt.isLeader != nil {
				in.resources.SetGeneric(leaderelection.IsLeaderKey, *tt.isLeader)
			}

			in.onPodDeleted(tt.deleted)

			// Read as the leader to see what was kept.
			in.resources.SetGeneric(leaderelection.IsLeaderKey, true)

			names, _, _ := readPodNames(t, in)
			assert.ElementsMatch(t, tt.want, names)
		})
	}
}

func TestKubernetesResourcesInputDeletedPodHandOver(t *testing.T) {
	in := newTestKubernetesResourcesInput(t, nil, newTestPod("default", "running", corev1.PodRunning, nil))

	before := time.Now().UTC().Truncate(time.Second)

	// A pod deleted between two reads is no longer listed. Its last state is a
	// terminal phase, which the list would skip.
	in.onPodDeleted(newTestPod("default", "deleted", corev1.PodSucceeded, nil))

	names, batch, ack := readPodNames(t, in)
	require.Equal(t, []string{"running", "deleted"}, names)

	_, ok := batch[0].MetaGet("kubernetes_deleted_time")
	assert.False(t, ok, "a listed pod must not have kubernetes_deleted_time")

	value, ok := batch[1].MetaGet("kubernetes_deleted_time")
	require.True(t, ok)

	deletedAt, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	assert.WithinRange(t, deletedAt, before, time.Now().UTC())

	// A rejected batch hands the deleted pod over again, with the same time.
	require.NoError(t, ack(t.Context(), errors.New("rejected")))

	names, batch, ack = readPodNames(t, in)
	require.Equal(t, []string{"running", "deleted"}, names)

	again, _ := batch[1].MetaGet("kubernetes_deleted_time")
	assert.Equal(t, value, again)

	// An acknowledged batch hands it over no more.
	require.NoError(t, ack(t.Context(), nil))

	names, _, _ = readPodNames(t, in)
	assert.Equal(t, []string{"running"}, names)
}

func TestKubernetesResourcesInputDeletedPodsNotLeader(t *testing.T) {
	in := newTestKubernetesResourcesInput(t, nil)

	in.onPodDeleted(newTestPod("default", "deleted", corev1.PodRunning, nil))

	// Losing the leadership drops what was not handed over.
	in.resources.SetGeneric(leaderelection.IsLeaderKey, false)

	names, _, _ := readPodNames(t, in)
	assert.Empty(t, names)

	in.resources.SetGeneric(leaderelection.IsLeaderKey, true)

	names, _, _ = readPodNames(t, in)
	assert.Empty(t, names)
}

func TestKubernetesResourcesInputDeletedPodsBound(t *testing.T) {
	in := newTestKubernetesResourcesInput(t, nil)

	in.onPodDeleted(newTestPod("default", "oldest", corev1.PodRunning, nil))

	for range maxDeletedPods {
		in.onPodDeleted(newTestPod("default", "newer", corev1.PodRunning, nil))
	}

	names, _, _ := readPodNames(t, in)
	assert.Len(t, names, maxDeletedPods)
	assert.NotContains(t, names, "oldest")
}

func TestKubernetesResourcesInputReadTime(t *testing.T) {
	in := newTestKubernetesResourcesInput(t, nil, newTestPod("default", "running", corev1.PodRunning, nil))

	meta := func(msg *service.Message, key string) string {
		value, _ := msg.MetaGet(key)

		return value
	}

	before := time.Now().UTC().Truncate(time.Second)

	_, first, ack := readPodNames(t, in)
	require.Len(t, first, 1)

	readTime, err := time.Parse(time.RFC3339, meta(first[0], "kubernetes_read_time"))
	require.NoError(t, err)
	assert.WithinRange(t, readTime, before, time.Now().UTC())

	// Nothing was acknowledged yet.
	_, ok := first[0].MetaGet("kubernetes_last_ack_read_time")
	assert.False(t, ok, "kubernetes_last_ack_read_time must not be set before a batch is acknowledged")

	// A rejected batch does not move the time.
	require.NoError(t, ack(t.Context(), errors.New("rejected")))

	// The metadata has a resolution of one second.
	time.Sleep(1100 * time.Millisecond)

	_, second, ack := readPodNames(t, in)

	_, ok = second[0].MetaGet("kubernetes_last_ack_read_time")
	assert.False(t, ok, "a rejected batch must not set kubernetes_last_ack_read_time")

	// An acknowledged batch does.
	require.NoError(t, ack(t.Context(), nil))

	_, third, _ := readPodNames(t, in)
	assert.Equal(t, meta(second[0], "kubernetes_read_time"), meta(third[0], "kubernetes_last_ack_read_time"))
	assert.NotEqual(t, meta(first[0], "kubernetes_read_time"), meta(third[0], "kubernetes_last_ack_read_time"))
}

func TestKubernetesResourcesInputReadsAfterAck(t *testing.T) {
	in := newTestKubernetesResourcesInput(t, nil, newTestPod("default", "running", corev1.PodRunning, nil))

	_, _, ack := readPodNames(t, in)

	// The next read waits for the batch to be acknowledged or rejected.
	read := make(chan service.MessageBatch)

	go func() {
		batch, _, err := in.ReadBatch(t.Context())
		assert.NoError(t, err)

		read <- batch
	}()

	select {
	case <-read:
		require.Fail(t, "read a batch while the previous one was in flight")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, ack(t.Context(), nil))

	select {
	case batch := <-read:
		require.Len(t, batch, 1)

		_, ok := batch[0].MetaGet("kubernetes_last_ack_read_time")
		assert.True(t, ok, "the batch must know the previous one was acknowledged")
	case <-time.After(5 * time.Second):
		require.Fail(t, "no batch after the previous one was acknowledged")
	}

	// A canceled read gives up waiting.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := in.ReadBatch(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestKubernetesResourcesInputEmptyBatchIsNotInFlight(t *testing.T) {
	in := newTestKubernetesResourcesInput(t, nil)

	// An empty batch is never acknowledged, so it must not block the next read.
	for range 3 {
		names, _, _ := readPodNames(t, in)
		assert.Empty(t, names)
	}
}
