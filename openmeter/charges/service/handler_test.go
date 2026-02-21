package service

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/charges"
)

var _ charges.Handler = (*MockHandler)(nil)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) OnStandardInvoiceRealizationCreated(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) (charges.Charge, error) {
	args := m.Called(ctx, charge, realization)
	return args.Get(0).(charges.Charge), args.Error(1)
}

func (m *MockHandler) OnStandardInvoiceRealizationAuthorized(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) (charges.Charge, error) {
	args := m.Called(ctx, charge, realization)
	return args.Get(0).(charges.Charge), args.Error(1)
}

func (m *MockHandler) OnStandardInvoiceRealizationSettled(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) (charges.Charge, error) {
	args := m.Called(ctx, charge, realization)
	return args.Get(0).(charges.Charge), args.Error(1)
}

func (m *MockHandler) OnRealizeUsageBasedCreditChargePeriodically(ctx context.Context, input charges.UsageBasedRealizationInput) ([]charges.CreditRealizationCreateInput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).([]charges.CreditRealizationCreateInput), args.Error(1)
}

type chargeAndRealization struct {
	charge      charges.Charge
	realization charges.StandardInvoiceRealizationWithLine
}

var _ charges.Handler = (*RecordingHandler)(nil)

type RecordingHandler struct {
	standardInvoiceRealizationCreated    []chargeAndRealization
	standardInvoiceRealizationAuthorized []chargeAndRealization
	standardInvoiceRealizationSettled    []chargeAndRealization
	usageBasedRealizationInput           []charges.UsageBasedRealizationInput
}

func (r *RecordingHandler) OnStandardInvoiceRealizationCreated(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) (charges.Charge, error) {
	r.standardInvoiceRealizationCreated = append(r.standardInvoiceRealizationCreated, chargeAndRealization{charge: charge, realization: realization})
	return charge, nil
}

func (r *RecordingHandler) OnStandardInvoiceRealizationAuthorized(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) (charges.Charge, error) {
	r.standardInvoiceRealizationAuthorized = append(r.standardInvoiceRealizationAuthorized, chargeAndRealization{charge: charge, realization: realization})
	return charge, nil
}

func (r *RecordingHandler) OnStandardInvoiceRealizationSettled(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) (charges.Charge, error) {
	r.standardInvoiceRealizationSettled = append(r.standardInvoiceRealizationSettled, chargeAndRealization{charge: charge, realization: realization})
	return charge, nil
}

func (r *RecordingHandler) OnRealizeUsageBasedCreditChargePeriodically(ctx context.Context, input charges.UsageBasedRealizationInput) ([]charges.CreditRealizationCreateInput, error) {
	r.usageBasedRealizationInput = append(r.usageBasedRealizationInput, input)
	return nil, nil
}

func (r *RecordingHandler) Reset() {
	r.standardInvoiceRealizationCreated = nil
	r.standardInvoiceRealizationAuthorized = nil
	r.standardInvoiceRealizationSettled = nil
	r.usageBasedRealizationInput = nil
}

type recordingHandlerExpectationItem struct {
	chargeID      string
	realizationID string
	status        charges.StandardInvoiceRealizationStatus
}

type recordingHandlerExpectation struct {
	standardInvoiceRealizationCreated    []recordingHandlerExpectationItem
	standardInvoiceRealizationAuthorized []recordingHandlerExpectationItem
	standardInvoiceRealizationSettled    []recordingHandlerExpectationItem
	usageBasedRealizationInput           []charges.UsageBasedRealizationInput
}

func (r *RecordingHandler) Expect(t *testing.T, expectations recordingHandlerExpectation) {
	t.Helper()

	compareExpectationWithChargeAndRealization(t, expectations.standardInvoiceRealizationCreated, r.standardInvoiceRealizationCreated, "standardInvoiceRealizationCreated")
	compareExpectationWithChargeAndRealization(t, expectations.standardInvoiceRealizationAuthorized, r.standardInvoiceRealizationAuthorized, "standardInvoiceRealizationAuthorized")
	compareExpectationWithChargeAndRealization(t, expectations.standardInvoiceRealizationSettled, r.standardInvoiceRealizationSettled, "standardInvoiceRealizationSettled")

	// realizationTriggers
	mapped := lo.Map(r.usageBasedRealizationInput, func(item charges.UsageBasedRealizationInput, _ int) string {
		return item.Charge.ID
	})
	require.ElementsMatch(t, expectations.usageBasedRealizationInput, mapped, "usageBasedRealizationInput")
}

func compareExpectationWithChargeAndRealization(t *testing.T, expectation []recordingHandlerExpectationItem, actual []chargeAndRealization, hookName string) {
	t.Helper()

	mapped := lo.Map(actual, func(item chargeAndRealization, _ int) recordingHandlerExpectationItem {
		return recordingHandlerExpectationItem{
			chargeID:      item.charge.ID,
			realizationID: item.realization.ID,
			status:        item.realization.Status,
		}
	})

	require.ElementsMatch(t, expectation, mapped, "hook %s", hookName)
}
