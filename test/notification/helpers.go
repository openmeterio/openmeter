package notification

import (
	"crypto/rand"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhousedriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
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

func NewMeterService() meter.Service {
	return meteradapter.New([]meter.Meter{
		{
			Namespace:     TestNamespace,
			ID:            ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String(),
			Slug:          TestMeterSlug,
			Aggregation:   meter.MeterAggregationSum,
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
