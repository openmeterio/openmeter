package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/samber/lo"

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
		Namespace:      params.Namespace,
		ClientID:       params.ClientID,
		From:           params.From,
		To:             params.To,
		ID:             params.ID,
		Subject:        params.Subject,
		Limit:          params.Limit,
		IngestedAtFrom: params.IngestedAtFrom,
		IngestedAtTo:   params.IngestedAtTo,
	}

	// Resolve customer IDs to customers if provided
	if params.CustomerIDs != nil && len(*params.CustomerIDs) > 0 {
		customers, err := a.listCustomers(ctx, params.Namespace, *params.CustomerIDs)
		if err != nil {
			return nil, fmt.Errorf("list customers: %w", err)
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
	meterEvents, err := a.eventPostProcess(ctx, params.Namespace, rawEvents)
	if err != nil {
		return nil, fmt.Errorf("post process events: %w", err)
	}

	return meterEvents, nil
}

// ListEventsV2 returns a list of events.
func (a *adapter) ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) (pagination.Result[meterevent.Event], error) {
	// Validate input
	if err := params.Validate(); err != nil {
		return pagination.Result[meterevent.Event]{}, models.NewGenericValidationError(
			fmt.Errorf("validate input: %w", err),
		)
	}

	listParams := streaming.ListEventsV2Params{
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
	}

	// Resolve customer IDs to customers if provided
	if params.CustomerID != nil && len(*params.CustomerID.In) > 0 {
		customers, err := a.listCustomers(ctx, params.Namespace, *params.CustomerID.In)
		if err != nil {
			return pagination.Result[meterevent.Event]{}, fmt.Errorf("list customers: %w", err)
		}

		// If no customers are found, return an empty list
		if len(customers) == 0 {
			return pagination.Result[meterevent.Event]{}, nil
		}

		listParams.Customers = &customers
	}

	// Get all events v2
	events, err := a.streamingConnector.ListEventsV2(ctx, listParams)
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("query events: %w", err)
	}

	// Map events to meter events
	meterEvents, err := a.eventPostProcess(ctx, params.Namespace, events)
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("post process events: %w", err)
	}

	return pagination.NewResult(meterEvents), nil
}

// listCustomers returns a list of customers.
func (a *adapter) listCustomers(ctx context.Context, namespace string, customerIDs []string) ([]streaming.Customer, error) {
	customerList, err := a.customerService.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace:   namespace,
		CustomerIDs: customerIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	// If some customers are not found, return an error
	if len(customerList.Items) != len(customerIDs) {
		var notFoundCustomerIDs []string

		for _, c := range customerIDs {
			_, found := lo.Find(customerList.Items, func(item customer.Customer) bool {
				return item.ID == c
			})

			if !found {
				notFoundCustomerIDs = append(notFoundCustomerIDs, c)
			}
		}

		if len(notFoundCustomerIDs) > 0 {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("customers not found in namespace %s: %v", namespace, strings.Join(notFoundCustomerIDs, ", ")),
			)
		}
	}

	customers := make([]streaming.Customer, 0, len(customerList.Items))

	for _, c := range customerList.Items {
		customers = append(customers, c)
	}

	return customers, nil
}

// eventPostProcess is a helper function to post-process events.
func (a *adapter) eventPostProcess(ctx context.Context, namespace string, rawEvents []streaming.RawEvent) ([]meterevent.Event, error) {
	var err error

	// Map events to meter events
	meterEvents := mapEventsToMeterEvents(rawEvents)

	// Enrich events with customer ID
	meterEvents, err = a.enrichEventsWithCustomerID(ctx, namespace, meterEvents)
	if err != nil {
		return nil, fmt.Errorf("enrich events with customer id: %w", err)
	}

	// Validate events
	meterEvents, err = a.validateEvents(ctx, namespace, meterEvents)
	if err != nil {
		return nil, fmt.Errorf("validate events: %w", err)
	}

	return meterEvents, nil
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

		// If the event does not have a customer ID, add an error to the event
		if event.CustomerID == nil {
			validationErrors = append(validationErrors, fmt.Errorf("no customer found for event subject: %s", event.Subject))
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
		if customerID, ok := cache[event.Subject]; ok {
			// Create a stable copy to take address of
			id := customerID
			event.CustomerID = &id
			eventsWithCustomerID = append(eventsWithCustomerID, event)
			continue
		}

		// FIXME: do this in a batches to avoid hitting the database for each event
		// Get the customer by usage attribution subject key
		cust, err := a.customerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
			Namespace: namespace,
			Key:       event.Subject,
		})
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				eventsWithCustomerID = append(eventsWithCustomerID, event)
				continue
			}

			return nil, fmt.Errorf("get customer by usage attribution: %w", err)
		}

		// Add the customer ID to the cache
		cache[event.Subject] = cust.ID

		// Add the event to the list (use stable copy for pointer)
		customerID := cust.ID
		event.CustomerID = &customerID
		eventsWithCustomerID = append(eventsWithCustomerID, event)
	}

	return eventsWithCustomerID, nil
}
