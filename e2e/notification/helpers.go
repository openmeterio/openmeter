package notification

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhousedriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewSvixAuthToken(signingSecret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "svix-server",
		Subject:   "org_23rb8YdGqMT0qIzpgGwdXfHirMu",
		ExpiresAt: jwt.NewNumericDate(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)),
		NotBefore: jwt.NewNumericDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
	})

	return token.SignedString([]byte(signingSecret))
}

func NewClickhouseClient(addr string) (clickhousedriver.Conn, error) {
	return clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "openmeter",
			Username: "default",
			Password: "default",
		},
		DialTimeout:      time.Duration(10) * time.Second,
		MaxOpenConns:     5,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Duration(10) * time.Minute,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		BlockBufferSize:  10,
	})
}

func NewMeterRepository() meter.Repository {
	return meter.NewInMemoryRepository([]models.Meter{
		{
			Namespace:     "default",
			ID:            ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String(),
			Slug:          "api-request",
			Aggregation:   models.MeterAggregationSum,
			EventType:     "request",
			ValueProperty: "$.duration_ms",
			GroupBy: map[string]string{
				"method": "$.method",
				"path":   "$.path",
			},
			WindowSize: "MINUTE",
		},
	})
}

func NewPGClient(url string) (*db.Client, error) {
	driver, err := entutils.GetPGDriver(url)
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres driver: %w", err)
	}

	// initialize client & run migrations
	dbClient := db.NewClient(db.Driver(driver))

	if err := dbClient.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to migrate credit db: %w", err)
	}

	return dbClient, nil
}
