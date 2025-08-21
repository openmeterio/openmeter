package leaderelection

import (
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	leaderElectionEnabledFlag   = "leader-election"
	leaseLockNamespaceFlag      = "lease-lock-namespace"
	leaseLockNameFlag           = "lease-lock-name"
	leaseLockIdentityFlag       = "lease-lock-identity"
	leaseDurationFlag           = "lease-duration"
	leaseRenewDeadlineFlag      = "lease-renew-deadline"
	leaseRetryPeriodFlag        = "lease-retry-period"
	leaseHealthCheckTimeoutFlag = "lease-health-check-timeout"
)

var hostname, _ = os.Hostname()

var leaderElectionCLIFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    leaderElectionEnabledFlag,
		Usage:   "Enable leader election",
		EnvVars: []string{"LEADER_ELECTION"},
	},
	&cli.StringFlag{
		Name:    leaseLockNamespaceFlag,
		Usage:   "Namespace of the lease lock",
		EnvVars: []string{"K8S_NAMESPACE", "LEASE_LOCK_NAMESPACE"},
	},
	&cli.StringFlag{
		Name:    leaseLockNameFlag,
		Usage:   "Name of the lease lock",
		EnvVars: []string{"K8S_APP_INSTANCE", "LEASE_LOCK_NAME"},
	},
	&cli.StringFlag{
		Name:    leaseLockIdentityFlag,
		Usage:   "Identity of the lease lock",
		EnvVars: []string{"K8S_POD_NAME", "LEASE_LOCK_IDENTITY"},
		Value:   hostname,
	},
	&cli.DurationFlag{
		Name:    leaseDurationFlag,
		Usage:   "Duration of the lease",
		EnvVars: []string{"LEASE_DURATION"},
		Value:   15 * time.Second,
	},
	&cli.DurationFlag{
		Name:    leaseRenewDeadlineFlag,
		Usage:   "Renew deadline of the lease",
		EnvVars: []string{"LEASE_RENEW_DEADLINE"},
		Value:   10 * time.Second,
	},
	&cli.DurationFlag{
		Name:    leaseRetryPeriodFlag,
		Usage:   "Retry period of the lease",
		EnvVars: []string{"LEASE_RETRY_PERIOD"},
		Value:   2 * time.Second,
	},
	&cli.DurationFlag{
		Name:    leaseHealthCheckTimeoutFlag,
		Usage:   "Timeout for lease health check. Should be longer than lease duration for inactive leader detection",
		EnvVars: []string{"LEASE_HEALTH_CHECK_TIMEOUT"},
		Value:   0,
	},
}
