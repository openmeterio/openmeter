package clickhouse

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/mock"
)

var _ clickhouse.Conn = &MockClickHouse{}

func NewMockClickHouse() *MockClickHouse {
	return &MockClickHouse{}
}

// MockClickHouse is a mock for the ClickHouse connection
type MockClickHouse struct {
	mock.Mock
}

func (m *MockClickHouse) Query(ctx context.Context, query string, queryArgs ...interface{}) (driver.Rows, error) {
	callArgs := m.Called(ctx, query, queryArgs)
	return callArgs.Get(0).(driver.Rows), callArgs.Error(1)
}

func (m *MockClickHouse) QueryRow(ctx context.Context, query string, queryArgs ...interface{}) driver.Row {
	callArgs := m.Called(ctx, query, queryArgs)
	return callArgs.Get(0).(driver.Row)
}

func (m *MockClickHouse) Select(ctx context.Context, dest any, query string, queryArgs ...any) error {
	callArgs := m.Called(ctx, dest, query, queryArgs)
	return callArgs.Error(0)
}

func (m *MockClickHouse) ServerVersion() (*clickhouse.ServerVersion, error) {
	callArgs := m.Called()
	return callArgs.Get(0).(*clickhouse.ServerVersion), callArgs.Error(1)
}

func (m *MockClickHouse) Contributors() []string {
	callArgs := m.Called()
	return callArgs.Get(0).([]string)
}

func (m *MockClickHouse) Stats() driver.Stats {
	callArgs := m.Called()
	return callArgs.Get(0).(driver.Stats)
}

func (m *MockClickHouse) PrepareBatch(ctx context.Context, query string, options ...driver.PrepareBatchOption) (driver.Batch, error) {
	callArgs := m.Called(ctx, query, options, options)
	return callArgs.Get(0).(driver.Batch), callArgs.Error(1)
}

func (m *MockClickHouse) AsyncInsert(ctx context.Context, query string, wait bool, args ...interface{}) error {
	callArgs := m.Called(ctx, query, wait, args)
	return callArgs.Error(0)
}

func (m *MockClickHouse) Exec(ctx context.Context, query string, args ...interface{}) error {
	callArgs := append([]interface{}{ctx, query}, args)
	return m.Called(callArgs...).Error(0)
}

func (m *MockClickHouse) Ping(ctx context.Context) error {
	callArgs := m.Called(ctx)
	return callArgs.Error(0)
}

func (m *MockClickHouse) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

var _ driver.Rows = &MockRows{}

func NewMockRows() *MockRows {
	return &MockRows{}
}

// MockRows is a mock for the Rows interface
type MockRows struct {
	mock.Mock
}

func (m *MockRows) Next() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockRows) Scan(dest ...interface{}) error {
	args := m.Called(dest)
	return args.Error(0)
}

func (m *MockRows) ScanStruct(dest any) error {
	args := m.Called(dest)
	return args.Error(0)
}

func (m *MockRows) Totals(dest ...interface{}) error {
	args := m.Called(dest)
	return args.Error(0)
}

func (m *MockRows) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRows) Columns() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockRows) ColumnTypes() []driver.ColumnType {
	args := m.Called()
	return args.Get(0).([]driver.ColumnType)
}

func (m *MockRows) Err() error {
	args := m.Called()
	return args.Error(0)
}
