package input

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/robfig/cron/v3"
)

const (
	fieldPrometheusURL      = "url"
	fieldPrometheusQueries  = "queries"
	fieldPrometheusSchedule = "schedule"
)

func prometheusInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Summary("Prometheus metrics input using PromQL.").
		Fields(
			service.NewStringField(fieldPrometheusURL).
				Description("Prometheus server URL.").
				Example("http://localhost:9090"),
			service.NewObjectListField(
				fieldPrometheusQueries,
				service.NewObjectField(
					"query",
					service.NewStringField("name").
						Description("A name for the query to be used as a metric identifier."),
					service.NewStringField("promql").
						Description("The PromQL query to execute."),
				),
			).Description("List of PromQL queries to execute."),
			service.NewStringField(fieldPrometheusSchedule).
				Description("The cron expression to use for the scrape job.").
				Examples("0 * * * * *", "@every 1m").
				Default("0 * * * * *"),
		).Example("Basic Configuration", "Collect Prometheus metrics with a scrape interval of 1 minute.", `
input:
  prometheus:
    url: "${PROMETHEUS_URL:http://localhost:9090}"
    schedule: "0 * * * * *"
    queries:
      - query:
          name: "node_cpu_usage"
          promql: "sum(increase(node_cpu_seconds_total{mode!='idle'}[1m])) by (instance)"
`)
}

func init() {
	err := service.RegisterBatchInput("prometheus", prometheusInputConfig(), func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) {
		return newPrometheusInput(conf, mgr.Logger())
	})
	if err != nil {
		panic(err)
	}
}

type PromQuery struct {
	Name   string `json:"name"`
	PromQL string `json:"promql"`
}

type QueryResult struct {
	Name      string       `json:"name"`
	Query     string       `json:"query"`
	Timestamp time.Time    `json:"timestamp"`
	Values    model.Vector `json:"values"`
}

var _ service.BatchInput = (*prometheusInput)(nil)

type prometheusInput struct {
	logger    *service.Logger
	client    v1.API
	queries   []PromQuery
	interval  time.Duration
	schedule  string
	scheduler gocron.Scheduler
	store     map[time.Time][]QueryResult
	mu        sync.Mutex
}

func newPrometheusInput(conf *service.ParsedConfig, logger *service.Logger) (*prometheusInput, error) {
	url, err := conf.FieldString(fieldPrometheusURL)
	if err != nil {
		return nil, err
	}

	schedule, err := conf.FieldString(fieldPrometheusSchedule)
	if err != nil {
		return nil, err
	}

	// Parse queries
	queriesConf, err := conf.FieldObjectList(fieldPrometheusQueries)
	if err != nil {
		return nil, err
	}

	queries := make([]PromQuery, len(queriesConf))
	for i, queryConf := range queriesConf {
		// Get the name field directly from the query object
		name, err := queryConf.FieldString("query", "name")
		if err != nil {
			return nil, err
		}

		// Get the promql field directly from the query object
		promql, err := queryConf.FieldString("query", "promql")
		if err != nil {
			return nil, err
		}

		queries[i] = PromQuery{
			Name:   name,
			PromQL: promql,
		}
	}

	// Calculate interval from schedule
	var interval time.Duration
	{
		// Create a cron scheduler
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		cronSchedule, err := parser.Parse(schedule)
		if err != nil {
			return nil, err
		}

		// Get current time
		now := time.Now()

		// Get next two occurrences
		nextRun := cronSchedule.Next(now)
		secondRun := cronSchedule.Next(nextRun)

		// Calculate the duration between runs
		interval = secondRun.Sub(nextRun)
	}

	// Create Prometheus client
	client, err := api.NewClient(api.Config{
		Address: url,
	})
	if err != nil {
		return nil, err
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &prometheusInput{
		logger:    logger,
		client:    v1.NewAPI(client),
		queries:   queries,
		interval:  interval,
		schedule:  schedule,
		scheduler: scheduler,
		store:     make(map[time.Time][]QueryResult),
		mu:        sync.Mutex{},
	}, nil
}

// scrape executes the PromQL queries and stores the results.
func (in *prometheusInput) scrape(ctx context.Context, t time.Time) error {
	// Convert time to UTC
	t = t.UTC()

	in.logger.Debugf("executing PromQL queries at %s", t.Format(time.RFC3339))

	results := make([]QueryResult, 0, len(in.queries))

	for _, query := range in.queries {
		in.logger.Tracef("executing query: %s", query.PromQL)

		// Execute the PromQL query
		result, warnings, err := in.client.Query(ctx, query.PromQL, t)
		if err != nil {
			in.logger.Errorf("error executing query %s: %v", query.PromQL, err)
			return err
		}

		if len(warnings) > 0 {
			for _, warning := range warnings {
				in.logger.Warnf("warning for query %s: %s", query.PromQL, warning)
			}
		}

		// Convert to vector
		vector, ok := result.(model.Vector)
		if !ok {
			in.logger.Warnf("result for query %s is not a vector, skipping", query.PromQL)
			continue
		}

		results = append(results, QueryResult{
			Name:      query.Name,
			Query:     query.PromQL,
			Timestamp: t,
			Values:    vector,
		})
	}

	in.mu.Lock()
	in.store[t] = results
	in.mu.Unlock()

	return nil
}

func (in *prometheusInput) Connect(ctx context.Context) error {
	// Add a job to the scheduler
	_, err := in.scheduler.NewJob(
		gocron.CronJob(in.schedule, true),
		gocron.NewTask(
			func(ctx context.Context) error {
				t := time.Now()
				err := in.scrape(ctx, t)
				if err != nil {
					in.logger.Errorf("error executing PromQL queries: %v", err)
					return err
				}

				return nil
			},
		),
		gocron.WithContext(ctx),
	)
	if err != nil {
		return err
	}

	// Start the scheduler
	in.scheduler.Start()

	return nil
}

func (in *prometheusInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	if len(in.store) == 0 {
		return nil, func(context.Context, error) error { return nil }, nil
	}

	in.mu.Lock()
	defer in.mu.Unlock()

	processing := make(map[time.Time][]QueryResult)
	batch := make([]*service.Message, 0)

	for t, results := range in.store {
		in.logger.Tracef("reading metrics from %s", t.Format(time.RFC3339))

		for _, result := range results {
			encoded, err := json.Marshal(result)
			if err != nil {
				return nil, func(context.Context, error) error { return nil }, err
			}

			msg := service.NewMessage(encoded)
			msg.MetaSet("scrape_time", t.Format(time.RFC3339))
			msg.MetaSet("scrape_interval", in.interval.String())
			msg.MetaSet("query_name", result.Name)
			batch = append(batch, msg)
		}

		processing[t] = results
		delete(in.store, t)
	}

	in.logger.Debugf("read %d metric results", len(batch))

	return batch, func(ctx context.Context, err error) error {
		if err != nil {
			in.mu.Lock()
			defer in.mu.Unlock()

			for t := range processing {
				in.logger.Tracef("nack received, readding unprocessed metrics to store: %s", t.Format(time.RFC3339))
				in.store[t] = processing[t]
			}

			return nil
		}

		in.logger.Tracef("ack received, discarding processed metrics")

		return nil
	}, nil
}

func (in *prometheusInput) Close(ctx context.Context) error {
	err := in.scheduler.StopJobs()
	if err != nil {
		return err
	}

	return nil
}
