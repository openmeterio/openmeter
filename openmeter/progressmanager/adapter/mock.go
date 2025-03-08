package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	progressmanagerentity "github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
	"github.com/stretchr/testify/mock"
)

var _ progressmanager.Service = &MockProgressManager{}

func NewMockProgressManager() *MockProgressManager {
	return &MockProgressManager{}
}

// MockProgressManager is a mock for the ProgressManager
type MockProgressManager struct {
	mock.Mock
}

func (m *MockProgressManager) GetProgress(ctx context.Context, input progressmanagerentity.GetProgressInput) (*progressmanagerentity.Progress, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*progressmanagerentity.Progress), args.Error(1)
}

func (m *MockProgressManager) DeleteProgressByRuntimeID(ctx context.Context, runtimeID string) error {
	args := m.Called(ctx, runtimeID)
	return args.Error(0)
}

func (m *MockProgressManager) UpsertProgress(ctx context.Context, input progressmanagerentity.UpsertProgressInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}
