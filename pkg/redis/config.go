package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// Config stores the user provided configuration parameters
type Config struct {
	Address  string
	Database int
	Username string
	Password string
	Sentinel struct {
		Enabled    bool
		MasterName string
	}
	TLS struct {
		Enabled            bool
		InsecureSkipVerify bool
	}
	PoolSize              int
	MaxRetries            int
	DialTimeout           time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	PoolTimeout           time.Duration
	ConnMaxIdleTime       time.Duration
	ConnMaxLifetime       time.Duration
	ConnMaxLifetimeJitter time.Duration
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	var errs []error

	if c.Address == "" {
		errs = append(errs, errors.New("address is required"))
	}

	if c.Sentinel.Enabled {
		if c.Sentinel.MasterName == "" {
			errs = append(errs, errors.New("sentinel: master name is required"))
		}
	}

	if c.PoolSize < 0 {
		errs = append(errs, errors.New("pool size must be positive or 0"))
	}

	if c.MaxRetries < 0 {
		errs = append(errs, errors.New("max retries must be positive or 0"))
	}

	if c.DialTimeout < 0 {
		errs = append(errs, errors.New("dial timeout must be positive or 0"))
	}

	if c.ReadTimeout < 0 {
		errs = append(errs, errors.New("read timeout must be positive or 0"))
	}

	if c.WriteTimeout < 0 {
		errs = append(errs, errors.New("write timeout must be positive or 0"))
	}

	if c.PoolTimeout < 0 {
		errs = append(errs, errors.New("pool timeout must be positive or 0"))
	}

	if c.ConnMaxIdleTime < 0 {
		errs = append(errs, errors.New("connection max idle time must be positive or 0"))
	}

	if c.ConnMaxLifetime < 0 {
		errs = append(errs, errors.New("connection max lifetime must be positive or 0"))
	}

	if c.ConnMaxLifetimeJitter < 0 {
		errs = append(errs, errors.New("connection max lifetime jitter must be positive or 0"))
	}

	return errors.Join(errs...)
}

// New client returns a redis client returns a redis client from the provided configuration
func (c Config) NewClient() (*redis.Client, error) {
	return NewClient(Options{Config: c})
}

// Configure sets the default values for the Redis configuration
func Configure(v *viper.Viper, prefix string) {
	// Redis driver
	v.SetDefault(fmt.Sprintf("%s.address", prefix), "127.0.0.1:6379")
	v.SetDefault(fmt.Sprintf("%s.database", prefix), 0)
	v.SetDefault(fmt.Sprintf("%s.username", prefix), "")
	v.SetDefault(fmt.Sprintf("%s.password", prefix), "")
	v.SetDefault(fmt.Sprintf("%s.expiration", prefix), "24h")
	v.SetDefault(fmt.Sprintf("%s.sentinel.enabled", prefix), false)
	v.SetDefault(fmt.Sprintf("%s.sentinel.masterName", prefix), "")
	v.SetDefault(fmt.Sprintf("%s.tls.enabled", prefix), false)
	v.SetDefault(fmt.Sprintf("%s.tls.insecureSkipVerify", prefix), false)
	v.SetDefault(fmt.Sprintf("%s.poolSize", prefix), 3)
	v.SetDefault(fmt.Sprintf("%s.maxRetries", prefix), 3)
	v.SetDefault(fmt.Sprintf("%s.dialTimeout", prefix), "5s")
	v.SetDefault(fmt.Sprintf("%s.readTimeout", prefix), "3s")
	v.SetDefault(fmt.Sprintf("%s.writeTimeout", prefix), "3s")
	v.SetDefault(fmt.Sprintf("%s.poolTimeout", prefix), "4s")
	v.SetDefault(fmt.Sprintf("%s.connMaxIdleTime", prefix), "30m")
	v.SetDefault(fmt.Sprintf("%s.connMaxLifetime", prefix), "0s")
	v.SetDefault(fmt.Sprintf("%s.connMaxLifetimeJitter", prefix), "0s")
}
