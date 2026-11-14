package main

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// presetWindow is the part of an event the kubernetes-pod-exec-time-exact
// preset computes per window.
type presetWindow struct {
	// Start is the window start in seconds after the test epoch.
	Start int64
	// Seconds is the duration the pod ran in the window.
	Seconds int64
}

// presetEpoch is a multiple of every interval the test uses.
var presetEpoch = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func presetTime(seconds int64) time.Time {
	return presetEpoch.Add(time.Duration(seconds) * time.Second)
}

func presetMetaTime(seconds int64) metav1.Time {
	return metav1.NewTime(presetTime(seconds))
}

// runPreset feeds the pods to the processors of the preset and returns the events.
func runPreset(t *testing.T, env map[string]string, metadata map[string]string, pods ...corev1.Pod) [][]byte {
	t.Helper()

	// The preset reads cloudevents.spec.json from the working directory.
	t.Chdir("../..")

	raw, err := os.ReadFile("collector/benthos/presets/kubernetes-pod-exec-time-exact/config.yaml")
	require.NoError(t, err)

	var preset struct {
		Pipeline struct {
			Processors []any `json:"processors"`
		} `json:"pipeline"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &preset))
	require.NotEmpty(t, preset.Pipeline.Processors)

	builder := service.NewStreamBuilder()
	builder.SetEnvVarLookupFunc(func(name string) (string, bool) {
		value, ok := env[name]

		return value, ok
	})
	require.NoError(t, builder.SetLoggerYAML(`level: "OFF"`))

	for _, processor := range preset.Pipeline.Processors {
		encoded, err := yaml.Marshal(processor)
		require.NoError(t, err)
		require.NoError(t, builder.AddProcessorYAML(string(encoded)))
	}

	produce, err := builder.AddBatchProducerFunc()
	require.NoError(t, err)

	var (
		mu     sync.Mutex
		events [][]byte
	)

	require.NoError(t, builder.AddBatchConsumerFunc(func(_ context.Context, batch service.MessageBatch) error {
		mu.Lock()
		defer mu.Unlock()

		for _, msg := range batch {
			encoded, err := msg.AsBytes()
			if err != nil {
				return err
			}

			events = append(events, encoded)
		}

		return nil
	}))

	stream, err := builder.Build()
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- stream.Run(context.WithoutCancel(t.Context())) }()

	batch := make(service.MessageBatch, 0, len(pods))

	for _, pod := range pods {
		encoded, err := json.Marshal(pod)
		require.NoError(t, err)

		msg := service.NewMessage(encoded)
		for key, value := range metadata {
			msg.MetaSet(key, value)
		}

		batch = append(batch, msg)
	}

	// Returns once every event of the batch reached the consumer.
	require.NoError(t, produce(t.Context(), batch))
	require.NoError(t, stream.StopWithin(10*time.Second))
	require.NoError(t, <-done)

	return events
}

func TestKubernetesPodExecTimeExactPreset(t *testing.T) {
	const uid = "0b8a5c1e-7a6f-4c0d-9d0e-3f1a2b4c5d6e"

	running := func(startedAt int64) []corev1.ContainerStatus {
		return []corev1.ContainerStatus{{
			Name:  "main",
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: presetMetaTime(startedAt)}},
		}}
	}

	terminated := func(startedAt, finishedAt int64) []corev1.ContainerStatus {
		return []corev1.ContainerStatus{{
			Name: "main",
			State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
				StartedAt:  presetMetaTime(startedAt),
				FinishedAt: presetMetaTime(finishedAt),
			}},
		}}
	}

	pod := func(phase corev1.PodPhase, statuses []corev1.ContainerStatus) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod", UID: uid},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status:     corev1.PodStatus{Phase: phase, ContainerStatuses: statuses},
		}
	}

	// meta builds the metadata of a read. A negative lastAck means no batch
	// was acknowledged yet.
	meta := func(now, lastAck int64) map[string]string {
		metadata := map[string]string{
			"schedule_interval":    "15s",
			"kubernetes_read_time": presetTime(now).Format(time.RFC3339),
		}

		if lastAck >= 0 {
			metadata["kubernetes_last_ack_read_time"] = presetTime(lastAck).Format(time.RFC3339)
		}

		return metadata
	}

	with := func(metadata map[string]string, key, value string) map[string]string {
		metadata[key] = value

		return metadata
	}

	tests := []struct {
		name     string
		env      map[string]string
		metadata map[string]string
		pod      corev1.Pod
		want     []presetWindow
	}{
		{
			name:     "running pod, one window since the last acknowledged read",
			metadata: meta(47, 32),
			pod:      pod(corev1.PodRunning, running(-100)),
			want:     []presetWindow{{Start: 15, Seconds: 15}},
		},
		{
			name:     "running pod, the newest window has not settled",
			metadata: meta(33, 18),
			pod:      pod(corev1.PodRunning, running(-100)),
			want:     []presetWindow{{Start: 0, Seconds: 15}},
		},
		{
			name:     "running pod, the window that had not settled comes with the next read",
			metadata: meta(48, 33),
			pod:      pod(corev1.PodRunning, running(-100)),
			want:     []presetWindow{{Start: 15, Seconds: 15}},
		},
		{
			name:     "running pod, two reads in one window emit nothing new",
			metadata: meta(36, 35),
			pod:      pod(corev1.PodRunning, running(-100)),
		},
		{
			name:     "running pod, a skipped read is caught up",
			metadata: meta(77, 32),
			pod:      pod(corev1.PodRunning, running(-100)),
			want:     []presetWindow{{Start: 15, Seconds: 15}, {Start: 30, Seconds: 15}, {Start: 45, Seconds: 15}},
		},
		{
			name:     "running pod, no acknowledged read emits the lookback",
			env:      map[string]string{"SCRAPE_LOOKBACK": "1m"},
			metadata: meta(62, -1),
			pod:      pod(corev1.PodRunning, running(-100)),
			want:     []presetWindow{{Start: -15, Seconds: 15}, {Start: 0, Seconds: 15}, {Start: 15, Seconds: 15}, {Start: 30, Seconds: 15}},
		},
		{
			name:     "running pod, an old acknowledged read is capped by the lookback",
			env:      map[string]string{"SCRAPE_LOOKBACK": "1m"},
			metadata: meta(1062, 2),
			pod:      pod(corev1.PodRunning, running(-100)),
			want:     []presetWindow{{Start: 990, Seconds: 15}, {Start: 1005, Seconds: 15}, {Start: 1020, Seconds: 15}, {Start: 1035, Seconds: 15}},
		},
		{
			name:     "running pod, settle time",
			env:      map[string]string{"SCRAPE_SETTLE": "0s"},
			metadata: meta(30, 15),
			pod:      pod(corev1.PodRunning, running(-100)),
			want:     []presetWindow{{Start: 15, Seconds: 15}},
		},
		{
			name:     "pod started inside the window",
			metadata: meta(47, 32),
			pod:      pod(corev1.PodRunning, running(20)),
			want:     []presetWindow{{Start: 15, Seconds: 10}},
		},
		{
			name:     "pod started after the window",
			metadata: meta(47, 32),
			pod:      pod(corev1.PodRunning, running(31)),
		},
		{
			name:     "pod started and ended between two reads",
			metadata: meta(32, 17),
			pod:      pod(corev1.PodSucceeded, terminated(16, 19)),
			want:     []presetWindow{{Start: 15, Seconds: 3}},
		},
		{
			name:     "ended pod, every window up to its end",
			metadata: meta(62, -1),
			pod:      pod(corev1.PodFailed, terminated(10, 50)),
			want:     []presetWindow{{Start: 0, Seconds: 5}, {Start: 15, Seconds: 15}, {Start: 30, Seconds: 15}, {Start: 45, Seconds: 5}},
		},
		{
			name:     "ended pod, the window it ended in does not wait for the settle time",
			metadata: meta(47, 32),
			pod:      pod(corev1.PodSucceeded, terminated(-100, 46)),
			want:     []presetWindow{{Start: 15, Seconds: 15}, {Start: 30, Seconds: 15}, {Start: 45, Seconds: 1}},
		},
		{
			name:     "ended pod, already reported",
			metadata: meta(92, 77),
			pod:      pod(corev1.PodSucceeded, terminated(10, 50)),
		},
		{
			name:     "deleted pod, last seen running, ends when the deletion was observed",
			metadata: with(meta(47, 32), "kubernetes_deleted_time", presetTime(41).Format(time.RFC3339)),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, running(10))
				// The deletion timestamp is the end of the grace period.
				p.DeletionTimestamp = lo.ToPtr(presetMetaTime(70))

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 15}, {Start: 30, Seconds: 11}},
		},
		{
			name:     "deleted pod, last seen running, ends at the latest with its grace period",
			metadata: with(meta(47, 32), "kubernetes_deleted_time", presetTime(41).Format(time.RFC3339)),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, running(10))
				p.DeletionTimestamp = lo.ToPtr(presetMetaTime(36))

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 15}, {Start: 30, Seconds: 6}},
		},
		{
			name:     "deleted pod, last seen stopped, ends when its container finished",
			metadata: with(meta(47, 32), "kubernetes_deleted_time", presetTime(41).Format(time.RFC3339)),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, terminated(10, 38))
				p.DeletionTimestamp = lo.ToPtr(presetMetaTime(70))

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 15}, {Start: 30, Seconds: 8}},
		},
		{
			name:     "pod being deleted whose container still runs is running",
			metadata: meta(47, 32),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, running(10))
				p.DeletionTimestamp = lo.ToPtr(presetMetaTime(40))

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 15}},
		},
		{
			name:     "pod being deleted whose container stopped has ended",
			metadata: meta(47, 32),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, terminated(10, 38))
				p.DeletionTimestamp = lo.ToPtr(presetMetaTime(70))

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 15}, {Start: 30, Seconds: 8}},
		},
		{
			name:     "start annotation later than the container start",
			metadata: meta(62, -1),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, running(-100))
				p.Annotations = map[string]string{"openmeter.io/start-time": presetTime(20).Format(time.RFC3339)}

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 10}, {Start: 30, Seconds: 15}},
		},
		{
			name:     "start annotation earlier than the container start",
			metadata: meta(47, 32),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, running(20))
				p.Annotations = map[string]string{"openmeter.io/start-time": presetTime(-100).Format(time.RFC3339)}

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 10}},
		},
		{
			name:     "start annotation with a configured name",
			env:      map[string]string{"START_TIME_ANNOTATION": "example.com/claimed-at"},
			metadata: meta(47, 32),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, running(-100))
				p.Annotations = map[string]string{"example.com/claimed-at": presetTime(27).Format(time.RFC3339)}

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 3}},
		},
		{
			name:     "restarted container keeps the start of its previous run",
			metadata: meta(47, 32),
			pod: func() corev1.Pod {
				p := pod(corev1.PodRunning, running(25))
				p.Status.ContainerStatuses[0].RestartCount = 1
				p.Status.ContainerStatuses[0].LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					StartedAt:  presetMetaTime(-100),
					FinishedAt: presetMetaTime(24),
				}}

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 15}},
		},
		{
			name:     "two containers run from the first start to the last finish",
			metadata: meta(62, -1),
			pod: func() corev1.Pod {
				p := pod(corev1.PodSucceeded, terminated(20, 35))
				p.Status.ContainerStatuses = append(p.Status.ContainerStatuses, terminated(18, 31)...)

				return p
			}(),
			want: []presetWindow{{Start: 15, Seconds: 12}, {Start: 30, Seconds: 5}},
		},
		{
			name:     "pod whose containers never started",
			metadata: meta(62, -1),
			pod: pod(corev1.PodPending, []corev1.ContainerStatus{{
				Name:  "main",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"}},
			}}),
		},
		{
			name:     "stopped container without a finish time ends when the pod was read",
			metadata: meta(47, 32),
			pod: pod(corev1.PodFailed, []corev1.ContainerStatus{{
				Name: "main",
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					Reason:    "ContainerStatusUnknown",
					StartedAt: presetMetaTime(10),
				}},
			}}),
			want: []presetWindow{{Start: 15, Seconds: 15}, {Start: 30, Seconds: 15}, {Start: 45, Seconds: 2}},
		},
		{
			name:     "stopped container without a start or a finish time never ran",
			metadata: meta(47, 32),
			pod: pod(corev1.PodFailed, []corev1.ContainerStatus{{
				Name:  "main",
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "ContainerStatusUnknown"}},
			}}),
		},
		{
			name:     "failed pod without container statuses",
			metadata: meta(62, -1),
			pod:      pod(corev1.PodFailed, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The cases are written for a settle time of 5s.
			env := map[string]string{"SCRAPE_SETTLE": "5s"}
			maps.Copy(env, tt.env)

			events := runPreset(t, env, tt.metadata, tt.pod)

			got := make([]presetWindow, 0, len(events))
			seconds := map[string]int64{}

			for _, encoded := range events {
				var event struct {
					ID   string    `json:"id"`
					Time time.Time `json:"time"`
					Data struct {
						DurationSeconds int64 `json:"duration_seconds"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(encoded, &event))

				got = append(got, presetWindow{Start: int64(event.Time.Sub(presetEpoch).Seconds()), Seconds: event.Data.DurationSeconds})
				seconds[event.ID] = event.Data.DurationSeconds
			}

			assert.ElementsMatch(t, tt.want, got)
			assert.Len(t, seconds, len(tt.want), "every window must have its own id")
		})
	}
}

// TestKubernetesPodExecTimeExactPresetDefaults runs the preset as it ships: a
// settle time of 15s and a lookback of 5m.
func TestKubernetesPodExecTimeExactPresetDefaults(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod", UID: "0b8a5c1e-7a6f-4c0d-9d0e-3f1a2b4c5d6e"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "main",
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: presetMetaTime(-1000)}},
			}},
		},
	}

	tests := []struct {
		name     string
		metadata map[string]string
		// Start of the first and the last window, in seconds after the test epoch.
		wantFirst int64
		wantLast  int64
	}{
		{
			name: "no acknowledged read emits the windows of the last 5m that closed 15s before the read",
			metadata: map[string]string{
				"schedule_interval":    "15s",
				"kubernetes_read_time": presetTime(62).Format(time.RFC3339),
			},
			wantFirst: -255,
			wantLast:  30,
		},
		{
			name: "one window since the last acknowledged read",
			metadata: map[string]string{
				"schedule_interval":             "15s",
				"kubernetes_read_time":          presetTime(62).Format(time.RFC3339),
				"kubernetes_last_ack_read_time": presetTime(47).Format(time.RFC3339),
			},
			wantFirst: 30,
			wantLast:  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := runPreset(t, nil, tt.metadata, pod)

			starts := make([]int64, 0, len(events))

			for _, encoded := range events {
				var event struct {
					Time time.Time `json:"time"`
					Data struct {
						DurationSeconds int64 `json:"duration_seconds"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(encoded, &event))
				assert.EqualValues(t, 15, event.Data.DurationSeconds)

				starts = append(starts, int64(event.Time.Sub(presetEpoch).Seconds()))
			}

			require.Len(t, starts, int((tt.wantLast-tt.wantFirst)/15)+1)
			assert.Equal(t, tt.wantFirst, slices.Min(starts))
			assert.Equal(t, tt.wantLast, slices.Max(starts))
		})
	}
}

func TestKubernetesPodExecTimeExactPresetEvent(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "pod",
			UID:       "0b8a5c1e-7a6f-4c0d-9d0e-3f1a2b4c5d6e",
			Annotations: map[string]string{
				"openmeter.io/subject":   "customer-1",
				"data.openmeter.io/team": "a",
				"example.com/ignored":    "x",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Name: "main",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
					Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
				},
			},
			{
				Name: "sidecar",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m")},
				},
			},
		}},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "main", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: presetMetaTime(20)}}},
				{Name: "sidecar", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: presetMetaTime(21)}}},
			},
		},
	}

	metadata := map[string]string{
		"schedule_interval":             "15s",
		"kubernetes_read_time":          presetTime(47).Format(time.RFC3339),
		"kubernetes_last_ack_read_time": presetTime(32).Format(time.RFC3339),
	}

	// The same pod and window give the same event whenever they are read.
	for _, read := range []struct {
		name     string
		metadata map[string]string
	}{
		{name: "after an acknowledged read", metadata: metadata},
		{name: "after a restart", metadata: map[string]string{
			"schedule_interval":    "15s",
			"kubernetes_read_time": presetTime(47).Format(time.RFC3339),
		}},
	} {
		t.Run(read.name, func(t *testing.T) {
			events := runPreset(t, nil, read.metadata, pod)
			require.Len(t, events, 1)

			assert.JSONEq(t, `{
				"id": "0b8a5c1e-7a6f-4c0d-9d0e-3f1a2b4c5d6e-1767225615",
				"specversion": "1.0",
				"type": "kubernetes",
				"source": "kubernetes-api",
				"time": "2026-01-01T00:00:15Z",
				"subject": "customer-1",
				"data": {
					"team": "a",
					"pod_name": "pod",
					"pod_namespace": "default",
					"duration_seconds": 10,
					"cpu_request_millicores": 0.3,
					"cpu_request_millicores_per_second": 0.02,
					"cpu_limit_millicores": 0.5,
					"cpu_limit_millicores_per_second": 0.03333333333333333,
					"memory_request_bytes": 67108864,
					"memory_request_bytes_per_second": 4473924.266666667,
					"memory_limit_bytes": 134217728,
					"memory_limit_bytes_per_second": 8947848.533333333,
					"gpu_request_count": 0,
					"gpu_request_count_per_second": 0,
					"gpu_limit_count": 0,
					"gpu_limit_count_per_second": 0
				}
			}`, string(events[0]))
		})
	}
}
