package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// ListEvents returns a list of events.
func (a *adapter) ListEvents(ctx context.Context, params meterevent.ListEventsParams) ([]meterevent.Event, error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("validate input: %w", err),
		)
	}

	// Params
	listParams := streaming.ListEventsParams{
		Namespace: params.Namespace,
		ClientID:  params.ClientID,
		From:      params.From,
		To:        params.To,
		ID:        params.ID,
		Subject:   params.Subject,
		Limit:     params.Limit,
	}

	// Resolve customer IDs to customers if provided
	if params.CustomerIDs != nil {
		customerList, err := a.customerService.ListCustomers(ctx, customer.ListCustomersInput{
			Namespace:   params.Namespace,
			CustomerIDs: *params.CustomerIDs,
		})
		if err != nil {
			return nil, fmt.Errorf("list customers: %w", err)
		}

		customers := make([]streaming.Customer, 0, len(customerList.Items))

		for _, c := range customerList.Items {
			customers = append(customers, c)
		}

		// If no customers are found, return an empty list
		if len(customers) == 0 {
			return []meterevent.Event{}, nil
		}

		listParams.Customers = &customers
	}

	// Get all events
	rawEvents, err := a.streamingConnector.ListEvents(ctx, params.Namespace, listParams)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	// Map events to meter events
	meterEvents := mapEventsToMeterEvents(rawEvents)

	// Validate events
	validatedEvents, err := a.validateEvents(ctx, params.Namespace, meterEvents)
	if err != nil {
		return nil, fmt.Errorf("validate events: %w", err)
	}

	// Enrich events with customer ID
	eventsWithCustomerID, err := a.enrichEventsWithCustomerID(ctx, params.Namespace, validatedEvents)
	if err != nil {
		return nil, fmt.Errorf("enrich events with customer id: %w", err)
	}

	return eventsWithCustomerID, nil
}

// ListEventsV2 returns a list of events.
func (a *adapter) ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) (pagination.Result[meterevent.Event], error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return pagination.Result[meterevent.Event]{}, models.NewGenericValidationError(
			fmt.Errorf("validate input: %w", err),
		)
	}

	// Get all events v2
	events, err := a.streamingConnector.ListEventsV2(ctx, streaming.ListEventsV2Params{
		Namespace:  params.Namespace,
		ClientID:   params.ClientID,
		Cursor:     params.Cursor,
		Limit:      params.Limit,
		ID:         params.ID,
		Source:     params.Source,
		Subject:    params.Subject,
		Type:       params.Type,
		Time:       params.Time,
		IngestedAt: params.IngestedAt,
	})
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("query events: %w", err)
	}

	// Map events to meter events
	meterEvents := mapEventsToMeterEvents(events)

	// Validate events
	validatedEvents, err := a.validateEvents(ctx, params.Namespace, meterEvents)
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("validate events: %w", err)
	}

	// Enrich events with customer ID
	eventsWithCustomerID, err := a.enrichEventsWithCustomerID(ctx, params.Namespace, validatedEvents)
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("enrich events with customer id: %w", err)
	}

	return pagination.NewResult(eventsWithCustomerID), nil
}

// mapEventsToMeterEvents maps a list of raw events to a list of meter events.
func mapEventsToMeterEvents(rawEvents []streaming.RawEvent) []meterevent.Event {
	meterEvents := make([]meterevent.Event, 0, len(rawEvents))

	for _, rawEvent := range rawEvents {
		meterEvent := meterevent.Event{
			ID:               rawEvent.ID,
			Type:             rawEvent.Type,
			Source:           rawEvent.Source,
			Subject:          rawEvent.Subject,
			Time:             rawEvent.Time,
			Data:             rawEvent.Data,
			CustomerID:       rawEvent.CustomerID,
			IngestedAt:       rawEvent.IngestedAt,
			StoredAt:         rawEvent.StoredAt,
			ValidationErrors: make([]error, 0),
		}

		meterEvents = append(meterEvents, meterEvent)
	}

	return meterEvents
}

// validateEvents validates a list of raw events against a list of meters.
func (a *adapter) validateEvents(ctx context.Context, namespace string, events []meterevent.Event) ([]meterevent.Event, error) {
	// Get all meters
	meterList, err := a.meterService.ListMeters(ctx, meter.ListMetersParams{
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("get meters: %w", err)
	}

	// Validate events against meters
	validatedEvents := make([]meterevent.Event, 0, len(events))

	for _, event := range events {
		meterMatch := false
		validationErrors := []error{}

		// Check if the event type matches a meter
		for _, m := range meterList.Items {
			// Check if the event type matches the meter event type
			if event.Type == m.EventType {
				meterMatch = true

				// Validate the event data against the meter
				_, err = meter.ParseEventString(m, event.Data)
				if err != nil {
					validationErrors = append(validationErrors, err)
				}
			}
		}

		// If no meter matches the event type, add an error to the event
		if !meterMatch {
			validationErrors = append(validationErrors, fmt.Errorf("no meter found for event type: %s", event.Type))
		}

		event.ValidationErrors = validationErrors
		validatedEvents = append(validatedEvents, event)
	}

	return validatedEvents, nil
}

// enrichEventsWithCustomerID enriches events with a customer ID if it exists.
func (a *adapter) enrichEventsWithCustomerID(ctx context.Context, namespace string, events []meterevent.Event) ([]meterevent.Event, error) {
	eventsWithCustomerID := make([]meterevent.Event, 0, len(events))
	cache := make(map[string]string)

	for _, event := range events {
		// If the event already has a customer ID, add it to the list
		if event.CustomerID != nil {
			eventsWithCustomerID = append(eventsWithCustomerID, event)
			continue
		}

		// Check if the customer ID for the subject is in the cache
		if customerId, ok := cache[event.Subject]; ok {
			event.CustomerID = &customerId
			eventsWithCustomerID = append(eventsWithCustomerID, event)
			continue
		}

		// FIXME: do this in a batches to avoid hitting the database for each event
		// Get the customer by usage attribution subject key
		customer, err := a.customerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
			Namespace:  namespace,
			SubjectKey: event.Subject,
		})
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				eventsWithCustomerID = append(eventsWithCustomerID, event)
				continue
			}

			return nil, fmt.Errorf("get customer by usage attribution: %w", err)
		}

		// Add the customer ID to the cache
		cache[event.Subject] = customer.ID

		// Add the event to the list
		event.CustomerID = &customer.ID
		eventsWithCustomerID = append(eventsWithCustomerID, event)
	}

	return eventsWithCustomerID, nil
}
