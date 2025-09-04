package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/subject/adapter"
	"github.com/openmeterio/openmeter/openmeter/subject/manager"
	"github.com/openmeterio/openmeter/openmeter/subject/service"
	subjecthooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"
)

var Subject = wire.NewSet(
	NewSubjectService,
	NewSubjectAdapter,
)

var SubjectManager = wire.NewSet(
	NewSubjectManager,
)

func NewSubjectService(
	adapter subject.Adapter,
) (subject.Service, error) {
	return service.New(adapter)
}

func NewSubjectManager(
	ctx context.Context,
	ent *entdb.Client,
	logger *slog.Logger,
	subjectConfig config.SubjectManagerConfig,
) (*manager.Manager, error) {
	subjectManager, err := manager.NewManager(&manager.Config{
		Ent:                 ent,
		Logger:              logger,
		CacheReloadInterval: subjectConfig.CacheReloadInterval,
		CacheReloadTimeout:  subjectConfig.CacheReloadTimeout,
		CachePrefillCount:   subjectConfig.CachePrefillCount,
		CacheSize:           subjectConfig.CacheSize,
		PaginationSize:      subjectConfig.PaginationSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subject manager: %w", err)
	}

	return subjectManager, nil
}

func NewSubjectAdapter(
	db *entdb.Client,
) (subject.Adapter, error) {
	return adapter.New(db)
}

func NewSubjectCustomerHook(
	subject subject.Service,
	customer customer.Service,
	logger *slog.Logger,
) (subjecthooks.CustomerSubjectHook, error) {
	h, err := subjecthooks.NewCustomerSubjectHook(subjecthooks.CustomerSubjectHookConfig{
		Subject: subject,
		Logger:  logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer subject hook: %w", err)
	}

	customer.RegisterHooks(h)

	return h, nil
}
