package ksqldbcheck

import (
	"context"
	"fmt"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/thmeitz/ksqldb-go"
)

type check struct {
	name   string
	client ksqldb.KsqldbClient
}

// NewCheck creates a new KSQLDB check.
func NewCheck(name string, client ksqldb.KsqldbClient) gosundheit.Check {
	return check{
		name:   name,
		client: client,
	}
}

func (c check) Name() string {
	return c.name
}

func (c check) Execute(ctx context.Context) (details any, err error) {
	status, err := c.client.GetServerStatus()
	if err != nil {
		return details, err
	}

	if !*status.IsHealthy {
		return details, fmt.Errorf("ksqldb server status is unhealthy")
	}

	return map[string]string{
		"service": status.KsqlServiceID,
	}, nil
}
