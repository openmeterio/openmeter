package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/internal/notification/repository/clickhouseschema"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ClickhouseAdapterConfig struct {
	Connection clickhouse.Conn
	Logger     *slog.Logger

	Database                string
	EventsTableName         string
	DeliveryStatusTableName string
}

func (c *ClickhouseAdapterConfig) Validate() error {
	if c.Connection == nil {
		return errors.New("clickhouse client is required")
	}

	if c.Database == "" {
		return fmt.Errorf("invalid name for clickhouse database: %q", c.EventsTableName)
	}

	if c.EventsTableName == "" {
		return fmt.Errorf("invalid name for notification events table: %q", c.EventsTableName)
	}

	if c.DeliveryStatusTableName == "" {
		return fmt.Errorf("invalid name for notification events delivary status table: %q", c.DeliveryStatusTableName)
	}

	if c.Logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

type clickhouseRepository interface {
	notification.EventRepository
}

var _ clickhouseRepository = (*clickhouseAdapter)(nil)

type clickhouseAdapter struct {
	db                  clickhouse.Conn
	eventsTable         *clickhouseschema.EventsTable
	deliveryStatusTable *clickhouseschema.DeliveryStatusTable

	logger *slog.Logger
	once   sync.Once
}

func newClickhouseAdapter(config ClickhouseAdapterConfig) *clickhouseAdapter {
	adapter := &clickhouseAdapter{
		db:                  config.Connection,
		eventsTable:         clickhouseschema.NewEventsTable(config.Database, config.EventsTableName),
		deliveryStatusTable: clickhouseschema.NewDeliveryStatusTable(config.Database, config.DeliveryStatusTableName),
		logger:              config.Logger,
	}

	return adapter
}

func (c *clickhouseAdapter) ListEvents(ctx context.Context, params notification.ListEventsInput) (pagination.PagedResponse[notification.Event], error) {
	var resp pagination.PagedResponse[notification.Event]

	query := c.eventsTable.ListEvents(params)

	c.logger.Debug("List query with args", "query", query.SQL, "args", query.Args)

	rows, err := c.db.Query(ctx, query.SQL, query.Args...)
	if err != nil {
		return resp, fmt.Errorf("failed to fetch notification events from database: %w", err)
	}
	defer rows.Close()

	items := make([]notification.Event, 0)

	for rows.Next() {
		eventRow := clickhouseschema.EventDBEntity{}

		if err = rows.ScanStruct(&eventRow); err != nil {
			return resp, fmt.Errorf("failed to unmarshal notification event from row from query result: %w", err)
		}

		var event *notification.Event
		event, err = clickhouseschema.EventFromDBEntity(eventRow)
		if err != nil {
			return resp, fmt.Errorf("failed to unmarshal notification event from db entity: %w", err)
		}

		items = append(items, *event)
	}

	totals, fields, err := clickhouseschema.EventDBEntity{}.Totals(rows.Columns())
	if err != nil {
		return resp, fmt.Errorf("failed to generate totals object: %w", err)
	}

	if err = rows.Totals(fields...); err != nil {
		return resp, fmt.Errorf("failed to unmarshal totals object from query result: %w", err)
	}

	return pagination.PagedResponse[notification.Event]{
		Page: pagination.Page{
			PageSize:   params.PageSize,
			PageNumber: params.PageNumber,
		},
		TotalCount: int(totals.TotalCount),
		Items:      items,
	}, nil
}

func (c *clickhouseAdapter) GetEvent(ctx context.Context, params notification.GetEventInput) (*notification.Event, error) {
	var resp *notification.Event

	query, err := c.eventsTable.GetEvent(params)
	if err != nil {
		return resp, fmt.Errorf("failed to assemble SQL query for fetching notification event: %w", err)
	}

	c.logger.Debug("Get event query with args", "query", query.SQL, "args", query.Args)

	rows, err := c.db.Query(ctx, query.SQL, query.Args...)
	if err != nil {
		return resp, fmt.Errorf("failed to fetch notification event from database: %w", err)
	}
	defer rows.Close()

	items := make([]notification.Event, 0)

	for rows.Next() {
		eventRow := clickhouseschema.EventDBEntity{}

		if err = rows.ScanStruct(&eventRow); err != nil {
			return resp, fmt.Errorf("failed to unmarshal notification event from row from query result: %w", err)
		}

		var event *notification.Event
		event, err = clickhouseschema.EventFromDBEntity(eventRow)
		if err != nil {
			return resp, fmt.Errorf("failed to unmarshal notification event from db entity: %w", err)
		}

		items = append(items, *event)
	}

	if len(items) == 1 {
		resp = &items[0]
	} else {
		return resp, notification.NotFoundError{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		}
	}

	return resp, nil
}

func (c *clickhouseAdapter) CreateEvent(ctx context.Context, params notification.CreateEventInput) (*notification.Event, error) {
	var resp *notification.Event

	query, eventEntity, err := c.eventsTable.CreateEvent(params)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble SQL query for inserting notification event: %w", err)
	}

	if err = c.db.Exec(ctx, query.SQL, query.Args...); err != nil {
		return nil, fmt.Errorf("failed to store notification event to database: %w", err)
	}

	var deliveryStatus []notification.EventDeliveryStatus
	for _, channel := range params.Rule.Channels {
		// FIXME: implement batch insert
		var statusEntity *clickhouseschema.DeliveryStatusDBEntity
		query, statusEntity, err = c.deliveryStatusTable.CreateDeliveryStatus(notification.CreateEventDeliveryStatusInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: params.Namespace,
			},
			EventID:   eventEntity.ID,
			State:     notification.EventDeliveryStatusStateSending,
			ChannelID: channel.ID,
			Timestamp: eventEntity.CreatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to assemble SQL query for inserting notification delivery status: %w", err)
		}

		if statusEntity == nil {
			return nil, errors.New("failed to assemble SQL query for inserting notification delivery status")
		}

		if err = c.db.Exec(ctx, query.SQL, query.Args...); err != nil {
			return resp, fmt.Errorf("failed to insert notification delivery status to database: %w", err)
		}

		var status *notification.EventDeliveryStatus
		status, err = clickhouseschema.DeliveryStatusFromDBEntity(*statusEntity)
		if err != nil {
			return nil, fmt.Errorf("failed to cast notification delivery status from db entity: %w", err)
		}

		deliveryStatus = append(deliveryStatus, *status)
	}

	return &notification.Event{
		NamespacedModel: models.NamespacedModel{
			Namespace: params.Namespace,
		},
		ID:             eventEntity.ID,
		Type:           params.Type,
		CreatedAt:      params.CreatedAt,
		DeliveryStatus: deliveryStatus,
		Payload:        params.Payload,
		Rule:           params.Rule,
	}, nil
}

func (c *clickhouseAdapter) ListEventsDeliveryStatus(ctx context.Context, params notification.ListEventsDeliveryStatusInput) (pagination.PagedResponse[notification.EventDeliveryStatus], error) {
	var resp pagination.PagedResponse[notification.EventDeliveryStatus]

	if len(params.EventIDs) > 0 {
		slices.Sort(params.EventIDs)
	}

	if (params.From.IsZero() || params.To.IsZero()) && len(params.EventIDs) > 0 {
		period, err := getTimePeriodFromEventIDs(params.EventIDs)
		if err != nil {
			return resp, fmt.Errorf("failed to get time period from event ids: %w", err)
		}

		if params.From.IsZero() {
			params.From = period.From
		}

		if params.To.IsZero() {
			params.To = period.To
		}
	}

	query := c.deliveryStatusTable.ListDeliveryStatus(params)

	c.logger.Debug("List delivery status", "query", query.SQL, "args", query.Args)

	rows, err := c.db.Query(ctx, query.SQL, query.Args...)
	if err != nil {
		return resp, fmt.Errorf("failed to fetch delivery status for notification events from database: %w", err)
	}
	defer rows.Close()

	items := make([]notification.EventDeliveryStatus, 0)

	for rows.Next() {
		statusRow := clickhouseschema.DeliveryStatusDBEntity{}

		if err = rows.ScanStruct(&statusRow); err != nil {
			return resp, fmt.Errorf("failed to unmarshal delivery status from row from query result: %w", err)
		}

		var status *notification.EventDeliveryStatus
		status, err = clickhouseschema.DeliveryStatusFromDBEntity(statusRow)
		if err != nil {
			return resp, fmt.Errorf("failed to unmarshal delivery status from db entity: %w", err)
		}

		items = append(items, *status)
	}

	totals, fields, err := clickhouseschema.DeliveryStatusDBEntity{}.Totals(rows.Columns())
	if err != nil {
		return resp, fmt.Errorf("failed to generate totals object: %w", err)
	}

	if err = rows.Totals(fields...); err != nil {
		return resp, fmt.Errorf("failed to unmarshal totals object from query result: %w", err)
	}

	return pagination.PagedResponse[notification.EventDeliveryStatus]{
		Page: pagination.Page{
			PageSize:   params.PageSize,
			PageNumber: params.PageNumber,
		},
		TotalCount: int(totals.TotalCount),
		Items:      items,
	}, nil
}

func (c *clickhouseAdapter) GetEventDeliveryStatus(ctx context.Context, params notification.GetEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	query, err := c.deliveryStatusTable.GetDeliveryStatus(params)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble SQL query for fetching delivery status for notification event: %w", err)
	}

	c.logger.Debug("Get delivery status for notification event", "query", query.SQL, "args", query.Args)

	rows, err := c.db.Query(ctx, query.SQL, query.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch delivery status for notification events from database: %w", err)
	}
	defer rows.Close()

	var resp *notification.EventDeliveryStatus
	for rows.Next() {
		statusRow := clickhouseschema.DeliveryStatusDBEntity{}

		if err = rows.ScanStruct(&statusRow); err != nil {
			return nil, fmt.Errorf("failed to unmarshal delivery status from row from query result: %w", err)
		}

		var status *notification.EventDeliveryStatus
		status, err = clickhouseschema.DeliveryStatusFromDBEntity(statusRow)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal delivery status from db entity: %w", err)
		}

		// Get the latest delivery status
		if resp == nil {
			resp = status
		} else if status.UpdatedAt.After(resp.UpdatedAt) {
			resp = status
		}
	}

	return resp, nil
}

func (c *clickhouseAdapter) CreateEventDeliveryStatus(ctx context.Context, params notification.CreateEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	query, status, err := c.deliveryStatusTable.CreateDeliveryStatus(params)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble SQL query for fetching notification event: %w", err)
	}

	if err = c.db.Exec(ctx, query.SQL, query.Args...); err != nil {
		return nil, fmt.Errorf("failed to store notification event to database: %w", err)
	}

	return &notification.EventDeliveryStatus{
		NamespacedModel: models.NamespacedModel{
			Namespace: params.Namespace,
		},
		EventID:   status.EventID,
		ChannelID: status.ChannelID,
		State:     notification.EventDeliveryStatusState(status.State),
		UpdatedAt: status.Timestamp,
	}, nil
}

func (c *clickhouseAdapter) init(ctx context.Context) error {
	var err error
	c.once.Do(func() {
		err = c.db.Exec(ctx, c.eventsTable.CreateTable().SQL)
		if err != nil {
			err = fmt.Errorf("failed to initialize %s table: %w", c.eventsTable.Name(), err)

			return
		}

		err = c.db.Exec(ctx, c.deliveryStatusTable.CreateTable())
		if err != nil {
			err = fmt.Errorf("failed to initialize %s table: %w", c.deliveryStatusTable.Name(), err)

			return
		}

		return
	})

	if err != nil {
		return err
	}

	return nil
}

type timePeriod struct {
	From time.Time
	To   time.Time
}

func getTimePeriodFromEventIDs(eventIDs []string) (timePeriod, error) {
	var resp timePeriod

	if len(eventIDs) == 0 {
		return resp, nil
	}

	slices.Sort(eventIDs)

	first := eventIDs[0]
	last := eventIDs[len(eventIDs)-1]

	firstULID, err := ulid.Parse(first)
	if err != nil {
		return resp, notification.ValidationError{
			Err: fmt.Errorf("failed to parse event id: %w", err),
		}
	}
	resp.From = time.UnixMilli(int64(firstULID.Time()))
	resp.To = resp.From

	if first != last {
		lastULID, err := ulid.Parse(last)
		if err != nil {
			return resp, notification.ValidationError{
				Err: fmt.Errorf("failed to parse event id: %w", err),
			}
		}
		resp.To = time.UnixMilli(int64(lastULID.Time()))
	}

	return resp, nil
}
