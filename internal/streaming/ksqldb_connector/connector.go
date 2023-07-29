package ksqldb_connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/thmeitz/ksqldb-go"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

type KsqlDBConnector struct {
	ksqlDBClient *ksqldb.KsqldbClient
	partitions   int
	format       string
	logger       *slog.Logger
}

func NewKsqlDBConnector(ksqldbClient *ksqldb.KsqldbClient, partitions int, format string, logger *slog.Logger) (*KsqlDBConnector, error) {
	connector := &KsqlDBConnector{
		ksqlDBClient: ksqldbClient,
		partitions:   partitions,
		format:       format,
		logger:       logger,
	}

	return connector, nil
}

func (c *KsqlDBConnector) Init(meter *models.Meter, namespace string) error {
	queryData := meterTableQueryData{
		Format:          c.format,
		Namespace:       namespace,
		Meter:           meter,
		WindowRetention: "36500 DAYS",
		Partitions:      c.partitions,
	}

	err := c.MeterAssert(queryData)
	if err != nil {
		return err
	}

	q, err := GetTableQuery(queryData)
	if err != nil {
		return fmt.Errorf("get table query for meter: %w", err)
	}
	c.logger.Debug("ksqlDB create table query", "query", q)

	resp, err := c.ksqlDBClient.Execute(context.Background(), ksqldb.ExecOptions{
		KSql: q,
	})
	if err != nil {
		return fmt.Errorf("create ksql table for meter: %w", err)
	}
	c.logger.Debug("ksqlDB response", "response", resp)

	return nil
}

// MeterAssert ensures meter table immutability by checking that existing meter table is the same as new
func (c *KsqlDBConnector) MeterAssert(data meterTableQueryData) error {
	q, err := GetTableDescribeQuery(data.Meter, data.Namespace)
	if err != nil {
		return fmt.Errorf("get table describe query: %w", err)
	}

	resp, err := c.ksqlDBClient.Execute(context.Background(), ksqldb.ExecOptions{
		KSql: q,
	})
	if err != nil {
		// It's not an issue if the table doesn't exist yet
		// If the table we want to describe does not exist yet ksqldb returns a 40001 error code (bad statement)
		// which is not specific enough to check here.
		if strings.HasPrefix(err.Error(), "Could not find") {
			return nil
		}

		return fmt.Errorf("describe table: %w", err)
	}

	sourceDescription := (*resp)[0]

	if len(sourceDescription.SourceDescription.WriteQueries) > 0 {
		c.logger.Debug("ksqlDB meter assert", "exists", true)

		query := sourceDescription.SourceDescription.WriteQueries[0].QueryString

		err = MeterQueryAssert(query, data)
		if err != nil {
			return err
		}

		c.logger.Debug("ksqlDB meter assert", "equals", true)
	} else {
		c.logger.Debug("ksqlDB meter assert", "exists", false)
	}

	return nil
}

func (c *KsqlDBConnector) GetValues(meter *models.Meter, params *streaming.GetValuesParams, namespace string) ([]*models.MeterValue, error) {
	q, err := GetTableValuesQuery(meter, params, namespace)
	if err != nil {
		return nil, err
	}
	slog.Debug("detectedEventsTableQuery", "query", q)

	header, payload, err := c.ksqlDBClient.Pull(context.TODO(), ksqldb.QueryOptions{
		Sql: q,
	})
	if err != nil {
		return nil, err
	}

	c.logger.Debug("ksqlDB response", "header", header, "payload", payload)
	values, err := NewMeterValues(header, payload)
	if err != nil {
		return nil, fmt.Errorf("get meter values: %w", err)
	}

	return meter.AggregateMeterValues(values, params.WindowSize)
}
