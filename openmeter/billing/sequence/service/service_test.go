package service

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/openmeter/billing/sequence"
	sequenceadapter "github.com/openmeterio/openmeter/openmeter/billing/sequence/adapter"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestCustomerPrefix(t *testing.T) {
	require.Equal(t, "UNKN", getCustomerPrefix(""))

	require.Equal(t, "JOHN", getCustomerPrefix("John"))
	require.Equal(t, "JO", getCustomerPrefix("Jo"))

	require.Equal(t, "PETU", getCustomerPrefix("Peter Turi"))
	require.Equal(t, "PTU", getCustomerPrefix("P Turi"))

	require.Equal(t, "LIHO", getCustomerPrefix("Líb Hosted"))
	require.Equal(t, "ABOR", getCustomerPrefix("Ábel Őri"))
	require.Equal(t, "PETU", getCustomerPrefix("Péter Túri"))
}

type SequenceServiceSuite struct {
	suite.Suite

	testDB  *testutils.TestDB
	adapter sequence.Adapter
	service sequence.Service
}

func (s *SequenceServiceSuite) SetupSuite() {
	s.testDB = testutils.InitPostgresDB(s.T(), testutils.PostgresDBStateEntMigrated)

	dbClient := db.NewClient(db.Driver(s.testDB.EntDriver.Driver()))
	adapter, err := sequenceadapter.New(sequenceadapter.Config{
		Client: dbClient,
		Logger: testutils.NewDiscardLogger(s.T()),
	})
	s.Require().NoError(err)
	s.adapter = adapter

	service, err := New(Config{
		Adapter: adapter,
		Meter:   metricnoop.NewMeterProvider().Meter("test"),
	})
	s.Require().NoError(err)
	s.service = service
}

func (s *SequenceServiceSuite) TearDownSuite() {
	if s.testDB != nil {
		s.testDB.Close(s.T())
	}
}

func (s *SequenceServiceSuite) TestAllocationAfterCallerRollback() {
	// given:
	// - both commit modes backed by the real sequence adapter
	// when:
	// - two allocations occur before the caller transaction rolls back
	// then:
	// - caller-bound allocations are released and independent allocations remain
	tests := []struct {
		name       string
		commitMode sequence.CommitMode
		wantNext   string
	}{
		{
			name:       "with caller",
			commitMode: sequence.CommitModeWithCaller,
			wantNext:   "INV-ACIN-1",
		},
		{
			name:       "independent",
			commitMode: sequence.CommitModeIndependent,
			wantNext:   "INV-ACIN-3",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			namespace := ulid.Make().String()
			callerRollback := errors.New("roll back caller transaction")

			allocated := make([]string, 0, 2)
			_, err := transaction.Run(s.T().Context(), s.adapter, func(ctx context.Context) (struct{}, error) {
				for range 2 {
					next, err := s.generateInvoiceNumber(ctx, namespace, tt.commitMode)
					if err != nil {
						return struct{}{}, err
					}

					allocated = append(allocated, next)
				}

				return struct{}{}, callerRollback
			})
			s.Require().ErrorIs(err, callerRollback)
			s.Equal([]string{"INV-ACIN-1", "INV-ACIN-2"}, allocated)

			next, err := s.generateInvoiceNumber(s.T().Context(), namespace, tt.commitMode)
			s.Require().NoError(err)
			s.Equal(tt.wantNext, next)
		})
	}
}

func (s *SequenceServiceSuite) TestAllocationAfterCallerCommit() {
	// given:
	// - both commit modes backed by the real sequence adapter
	// when:
	// - two allocations occur before the caller transaction commits
	// then:
	// - both modes retain the allocations and continue from the third number
	for _, commitMode := range []sequence.CommitMode{
		sequence.CommitModeWithCaller,
		sequence.CommitModeIndependent,
	} {
		s.Run(string(commitMode), func() {
			namespace := ulid.Make().String()

			allocated, err := transaction.Run(s.T().Context(), s.adapter, func(ctx context.Context) ([]string, error) {
				result := make([]string, 0, 2)
				for range 2 {
					next, err := s.generateInvoiceNumber(ctx, namespace, commitMode)
					if err != nil {
						return nil, err
					}

					result = append(result, next)
				}

				return result, nil
			})
			s.Require().NoError(err)
			s.Equal([]string{"INV-ACIN-1", "INV-ACIN-2"}, allocated)

			next, err := s.generateInvoiceNumber(s.T().Context(), namespace, commitMode)
			s.Require().NoError(err)
			s.Equal("INV-ACIN-3", next)
		})
	}
}

func (s *SequenceServiceSuite) TestConcurrentAllocationsAreUnique() {
	// given:
	// - both commit modes backed by the real sequence adapter
	// - one sequence scope with no existing allocation row
	// when:
	// - concurrent callers allocate numbers through the sequence service
	// then:
	// - every number from 1 through allocationCount is returned exactly once
	const allocationCount = 20

	type allocationResult struct {
		next string
		err  error
	}

	for _, commitMode := range []sequence.CommitMode{
		sequence.CommitModeWithCaller,
		sequence.CommitModeIndependent,
	} {
		s.Run(string(commitMode), func() {
			namespace := ulid.Make().String()
			ctx := s.T().Context()
			results := make(chan allocationResult, allocationCount)

			for range allocationCount {
				go func() {
					var next string
					var err error

					if commitMode == sequence.CommitModeWithCaller {
						next, err = transaction.Run(ctx, s.adapter, func(ctx context.Context) (string, error) {
							return s.generateInvoiceNumber(ctx, namespace, commitMode)
						})
					} else {
						next, err = s.generateInvoiceNumber(ctx, namespace, commitMode)
					}

					results <- allocationResult{next: next, err: err}
				}()
			}

			allocated := make([]string, 0, allocationCount)
			for range allocationCount {
				select {
				case <-ctx.Done():
					s.T().Fatalf("test context canceled before allocations completed: %v", ctx.Err())
				case result := <-results:
					s.Require().NoError(result.err)
					allocated = append(allocated, result.next)
				}
			}

			s.Require().Len(lo.Uniq(allocated), allocationCount)

			expected := lo.Map(lo.RangeFrom(1, allocationCount), func(id int, _ int) string {
				return "INV-ACIN-" + strconv.Itoa(id)
			})
			s.ElementsMatch(expected, allocated)
		})
	}
}

func (s *SequenceServiceSuite) generateInvoiceNumber(ctx context.Context, namespace string, commitMode sequence.CommitMode) (string, error) {
	return s.service.GenerateInvoiceSequenceNumber(ctx, sequence.GenerationInput{
		Namespace:    namespace,
		CustomerName: "Acme Inc",
		Currency:     currencyx.FiatCode("USD"),
	}, sequence.Definition{
		Prefix:         "INV",
		SuffixTemplate: "{{.CustomerPrefix}}-{{.NextSequenceNumber}}",
		Scope:          "invoices/test",
		CommitMode:     commitMode,
	})
}

func TestSequenceServiceSuite(t *testing.T) {
	suite.Run(t, new(SequenceServiceSuite))
}
