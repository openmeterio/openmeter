package leaderelection

import (
	"testing"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestServiceStartDelay(t *testing.T) {
	const identity = "collector-0"

	// A lease that a previous run of this replica holds and that did not
	// expire. client-go lets the same identity take it over at once.
	heldLease := func() *coordinationv1.Lease {
		now := metav1.NewMicroTime(time.Now())

		return &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "collector"},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       lo.ToPtr(identity),
				LeaseDurationSeconds: lo.ToPtr(int32(60)),
				AcquireTime:          &now,
				RenewTime:            &now,
			},
		}
	}

	tests := []struct {
		name       string
		startDelay time.Duration
	}{
		{
			name: "no delay",
		},
		{
			name:       "delay",
			startDelay: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := service.MockResources()

			s := newService(res, Config{
				StartDelay:         tt.startDelay,
				LeaseLockNamespace: "default",
				LeaseLockName:      "collector",
				LeaseDuration:      3 * time.Second,
				LeaseRenewDeadline: 2 * time.Second,
				LeaseRetryPeriod:   100 * time.Millisecond,
				Identity:           identity,
			}, fake.NewClientset(heldLease()).CoordinationV1())

			require.NoError(t, s.Start(t.Context()))

			isLeader := func() bool { return IsLeader(res) }

			if tt.startDelay > 0 {
				assert.Never(t, isLeader, tt.startDelay/2, 10*time.Millisecond, "became the leader before the start delay passed")
			}

			assert.Eventually(t, isLeader, tt.startDelay+5*time.Second, 10*time.Millisecond)
		})
	}
}
