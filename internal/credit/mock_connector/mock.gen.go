// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/openmeterio/openmeter/internal/credit (interfaces: Connector)

// Package credit_mock is a generated GoMock package.
package credit_mock

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	credit "github.com/openmeterio/openmeter/internal/credit"
)

// MockConnector is a mock of Connector interface.
type MockConnector struct {
	ctrl     *gomock.Controller
	recorder *MockConnectorMockRecorder
}

// MockConnectorMockRecorder is the mock recorder for MockConnector.
type MockConnectorMockRecorder struct {
	mock *MockConnector
}

// NewMockConnector creates a new mock instance.
func NewMockConnector(ctrl *gomock.Controller) *MockConnector {
	mock := &MockConnector{ctrl: ctrl}
	mock.recorder = &MockConnectorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConnector) EXPECT() *MockConnectorMockRecorder {
	return m.recorder
}

// CreateFeature mocks base method.
func (m *MockConnector) CreateFeature(arg0 context.Context, arg1 credit.Feature) (credit.Feature, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateFeature", arg0, arg1)
	ret0, _ := ret[0].(credit.Feature)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateFeature indicates an expected call of CreateFeature.
func (mr *MockConnectorMockRecorder) CreateFeature(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFeature", reflect.TypeOf((*MockConnector)(nil).CreateFeature), arg0, arg1)
}

// CreateGrant mocks base method.
func (m *MockConnector) CreateGrant(arg0 context.Context, arg1 credit.Grant) (credit.Grant, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateGrant", arg0, arg1)
	ret0, _ := ret[0].(credit.Grant)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateGrant indicates an expected call of CreateGrant.
func (mr *MockConnectorMockRecorder) CreateGrant(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateGrant", reflect.TypeOf((*MockConnector)(nil).CreateGrant), arg0, arg1)
}

// CreateLedger mocks base method.
func (m *MockConnector) CreateLedger(arg0 context.Context, arg1 credit.Ledger) (credit.Ledger, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateLedger", arg0, arg1)
	ret0, _ := ret[0].(credit.Ledger)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateLedger indicates an expected call of CreateLedger.
func (mr *MockConnectorMockRecorder) CreateLedger(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateLedger", reflect.TypeOf((*MockConnector)(nil).CreateLedger), arg0, arg1)
}

// DeleteFeature mocks base method.
func (m *MockConnector) DeleteFeature(arg0 context.Context, arg1 credit.NamespacedFeatureID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFeature", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFeature indicates an expected call of DeleteFeature.
func (mr *MockConnectorMockRecorder) DeleteFeature(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFeature", reflect.TypeOf((*MockConnector)(nil).DeleteFeature), arg0, arg1)
}

// GetBalance mocks base method.
func (m *MockConnector) GetBalance(arg0 context.Context, arg1 credit.NamespacedLedgerID, arg2 time.Time) (credit.Balance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalance", arg0, arg1, arg2)
	ret0, _ := ret[0].(credit.Balance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBalance indicates an expected call of GetBalance.
func (mr *MockConnectorMockRecorder) GetBalance(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalance", reflect.TypeOf((*MockConnector)(nil).GetBalance), arg0, arg1, arg2)
}

// GetFeature mocks base method.
func (m *MockConnector) GetFeature(arg0 context.Context, arg1 credit.NamespacedFeatureID) (credit.Feature, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFeature", arg0, arg1)
	ret0, _ := ret[0].(credit.Feature)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFeature indicates an expected call of GetFeature.
func (mr *MockConnectorMockRecorder) GetFeature(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFeature", reflect.TypeOf((*MockConnector)(nil).GetFeature), arg0, arg1)
}

// GetGrant mocks base method.
func (m *MockConnector) GetGrant(arg0 context.Context, arg1 credit.NamespacedGrantID) (credit.Grant, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGrant", arg0, arg1)
	ret0, _ := ret[0].(credit.Grant)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGrant indicates an expected call of GetGrant.
func (mr *MockConnectorMockRecorder) GetGrant(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGrant", reflect.TypeOf((*MockConnector)(nil).GetGrant), arg0, arg1)
}

// GetHighWatermark mocks base method.
func (m *MockConnector) GetHighWatermark(arg0 context.Context, arg1 credit.NamespacedLedgerID) (credit.HighWatermark, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHighWatermark", arg0, arg1)
	ret0, _ := ret[0].(credit.HighWatermark)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHighWatermark indicates an expected call of GetHighWatermark.
func (mr *MockConnectorMockRecorder) GetHighWatermark(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHighWatermark", reflect.TypeOf((*MockConnector)(nil).GetHighWatermark), arg0, arg1)
}

// GetHistory mocks base method.
func (m *MockConnector) GetHistory(arg0 context.Context, arg1 credit.NamespacedLedgerID, arg2, arg3 time.Time, arg4 credit.Pagination, arg5 *credit.WindowParams) (credit.LedgerEntryList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHistory", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(credit.LedgerEntryList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHistory indicates an expected call of GetHistory.
func (mr *MockConnectorMockRecorder) GetHistory(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHistory", reflect.TypeOf((*MockConnector)(nil).GetHistory), arg0, arg1, arg2, arg3, arg4, arg5)
}

// ListFeatures mocks base method.
func (m *MockConnector) ListFeatures(arg0 context.Context, arg1 credit.ListFeaturesParams) ([]credit.Feature, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListFeatures", arg0, arg1)
	ret0, _ := ret[0].([]credit.Feature)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListFeatures indicates an expected call of ListFeatures.
func (mr *MockConnectorMockRecorder) ListFeatures(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFeatures", reflect.TypeOf((*MockConnector)(nil).ListFeatures), arg0, arg1)
}

// ListGrants mocks base method.
func (m *MockConnector) ListGrants(arg0 context.Context, arg1 credit.ListGrantsParams) ([]credit.Grant, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListGrants", arg0, arg1)
	ret0, _ := ret[0].([]credit.Grant)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListGrants indicates an expected call of ListGrants.
func (mr *MockConnectorMockRecorder) ListGrants(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListGrants", reflect.TypeOf((*MockConnector)(nil).ListGrants), arg0, arg1)
}

// ListLedgers mocks base method.
func (m *MockConnector) ListLedgers(arg0 context.Context, arg1 credit.ListLedgersParams) ([]credit.Ledger, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListLedgers", arg0, arg1)
	ret0, _ := ret[0].([]credit.Ledger)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLedgers indicates an expected call of ListLedgers.
func (mr *MockConnectorMockRecorder) ListLedgers(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLedgers", reflect.TypeOf((*MockConnector)(nil).ListLedgers), arg0, arg1)
}

// Reset mocks base method.
func (m *MockConnector) Reset(arg0 context.Context, arg1 credit.Reset) (credit.Reset, []credit.Grant, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reset", arg0, arg1)
	ret0, _ := ret[0].(credit.Reset)
	ret1, _ := ret[1].([]credit.Grant)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Reset indicates an expected call of Reset.
func (mr *MockConnectorMockRecorder) Reset(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reset", reflect.TypeOf((*MockConnector)(nil).Reset), arg0, arg1)
}

// VoidGrant mocks base method.
func (m *MockConnector) VoidGrant(arg0 context.Context, arg1 credit.Grant) (credit.Grant, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VoidGrant", arg0, arg1)
	ret0, _ := ret[0].(credit.Grant)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VoidGrant indicates an expected call of VoidGrant.
func (mr *MockConnectorMockRecorder) VoidGrant(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VoidGrant", reflect.TypeOf((*MockConnector)(nil).VoidGrant), arg0, arg1)
}
