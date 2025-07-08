package leaderelection

import (
	"context"
	"fmt"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/urfave/cli/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // import kubernetes auth plugins
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type genericKey string

const (
	IsLeaderKey genericKey = "leaderElection.isLeader"
)

type Config struct {
	Enabled            bool
	LeaseLockNamespace string
	LeaseLockName      string
	LeaseDuration      time.Duration
	LeaseRenewDeadline time.Duration
	LeaseRetryPeriod   time.Duration
	Identity           string
}

type Service struct {
	config               Config
	logger               *service.Logger
	resources            *service.Resources
	leaderElectionConfig *leaderelection.LeaderElectionConfig
}

// TODO: add metrics to leader election

func NewService(res *service.Resources, cfg Config) (*Service, error) {
	logger := res.Logger().With("component", "leader election")

	kubeconfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Namespace: cfg.LeaseLockNamespace,
			Name:      cfg.LeaseLockName,
		},
		Client:     client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: cfg.Identity},
	}

	leaderElectionConfig := &leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   cfg.LeaseDuration,
		RenewDeadline:   cfg.LeaseRenewDeadline,
		RetryPeriod:     cfg.LeaseRetryPeriod,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Info("lease acquired")
				res.SetGeneric(IsLeaderKey, true)
			},
			OnStoppedLeading: func() {
				logger.Info("lease lost")
				res.SetGeneric(IsLeaderKey, false)
			},
			OnNewLeader: func(newID string) {
				if newID != cfg.Identity {
					logger.Infof("current leader: %s", newID)
				}
			},
		},
	}

	return &Service{
		config:               cfg,
		logger:               logger,
		resources:            res,
		leaderElectionConfig: leaderElectionConfig,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	s.resources.SetGeneric(IsLeaderKey, false)

	leaderelection.RunOrDie(ctx, *s.leaderElectionConfig)
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.logger.Debug("stopping leader election service")
	return nil
}

func GetLeaderElectionCLIOpts(ctx context.Context) []service.CLIOptFunc {
	var leaderElectionConfig Config

	return []service.CLIOptFunc{
		service.CLIOptCustomRunFlags(
			leaderElectionCLIFlags,
			func(ctx *cli.Context) error {
				leaderElectionConfig = Config{
					Enabled:            ctx.Bool(leaderElectionEnabledFlag),
					LeaseLockNamespace: ctx.String(leaseLockNamespaceFlag),
					LeaseLockName:      ctx.String(leaseLockNameFlag),
					LeaseDuration:      ctx.Duration(leaseDurationFlag),
					LeaseRenewDeadline: ctx.Duration(leaseRenewDeadlineFlag),
					LeaseRetryPeriod:   ctx.Duration(leaseRetryPeriodFlag),
					Identity:           ctx.String(leaseLockIdentityFlag),
				}
				return nil
			}),
		service.CLIOptOnConfigParse(func(conf *service.ParsedConfig) error {
			if !leaderElectionConfig.Enabled {
				return nil
			}

			if leaderElectionConfig.LeaseLockNamespace == "" {
				return fmt.Errorf("lease lock namespace is required when leader election is enabled")
			}

			if leaderElectionConfig.LeaseLockName == "" {
				return fmt.Errorf("lease lock name is required when leader election is enabled")
			}

			s, err := NewService(conf.Resources(), leaderElectionConfig)
			if err != nil {
				return err
			}

			go func() {
				if err := s.Start(ctx); err != nil {
					s.logger.Errorf("failed to start leader election service: %v", err)
				}
			}()
			go func() {
				<-ctx.Done()
				if err := s.Stop(ctx); err != nil {
					s.logger.Errorf("failed to stop leader election service: %v", err)
				}
			}()

			return nil
		}),
	}
}

func IsLeader(res *service.Resources) bool {
	leader, ok := res.GetGeneric(IsLeaderKey)
	// If the key is not set, we are not using leader election, so we are the leader
	if !ok {
		return true
	}

	isLeader, ok := leader.(bool)
	// If the key is set but not a bool, we are not the leader
	if !ok {
		return false
	}

	return isLeader
}
