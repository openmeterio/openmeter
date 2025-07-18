package input

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/collector/benthos/input/runai"
	"github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"
)

var resourceTypes = []string{"workload", "pod"}

const (
	fieldURL                  = "url"
	fieldAppID                = "app_id"
	fieldAppSecret            = "app_secret"
	fieldResourceType         = "resource_type"
	fieldMetrics              = "metrics"
	fieldSchedule             = "schedule"
	fieldMetricsOffset        = "metrics_offset"
	fieldPageSize             = "page_size"
	fieldHTTPConfig           = "http"
	fieldHTTPTimeout          = "timeout"
	fieldHTTPRetryCount       = "retry_count"
	fieldHTTPRetryWaitTime    = "retry_wait_time"
	fieldHTTPRetryMaxWaitTime = "retry_max_wait_time"
)

func runAIInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Summary("Run AI metrics input.").
		Fields(
			service.NewStringField(fieldURL).
				Description("Run AI base URL."),
			service.NewStringField(fieldAppID).
				Description("Run AI app ID."),
			service.NewStringField(fieldAppSecret).
				Description("Run AI app secret."),
			service.NewStringEnumField(fieldResourceType, resourceTypes...).
				Default("workload").
				Description("Run AI resource to collect metrics from."),
			service.NewIntField(fieldPageSize).
				Description("Run AI page size.").
				Default(500),
			service.NewStringListField(fieldMetrics).
				Description("Run AI metrics to collect.").
				Default(lo.Map([]runai.MetricType{
					runai.WorkloadMetricTypeCPULimitCores,
					runai.WorkloadMetricTypeCPUMemoryLimit,
					runai.WorkloadMetricTypeCPUMemoryRequest,
					runai.WorkloadMetricTypeCPUMemoryUsage,
					runai.WorkloadMetricTypeCPURequestCores,
					runai.WorkloadMetricTypeCPUUsageCores,
					runai.WorkloadMetricTypeGPUAllocation,
					runai.WorkloadMetricTypeGPUMemoryRequest,
					runai.WorkloadMetricTypeGPUMemoryUsage,
					runai.WorkloadMetricTypeGPUUtilization,
					runai.WorkloadMetricTypePodCount,
					runai.WorkloadMetricTypeRunningPodCount,
				}, func(metric runai.MetricType, _ int) string {
					return string(metric)
				})),
			service.NewStringField(fieldSchedule).
				Description("The cron expression to use for the scrape job.").
				Examples("*/30 * * * * *", "@every 30s").
				Default("*/30 * * * * *"),
			service.NewDurationField(fieldMetricsOffset).
				Description("Indicates how far back in time the scraping window should start to account for delays in metric availability.").
				Default("0s"),
			service.NewObjectField(fieldHTTPConfig,
				service.NewDurationField(fieldHTTPTimeout).
					Description("Request timeout.").
					Default("30s"),
				service.NewIntField(fieldHTTPRetryCount).
					Description("The number of retries to attempt.").
					Default(1),
				service.NewDurationField(fieldHTTPRetryWaitTime).
					Description("The wait time between retries.").
					Default("100ms"),
				service.NewDurationField(fieldHTTPRetryMaxWaitTime).
					Description("The maximum wait time between retries.").
					Default("1s"),
			).Description("HTTP client configuration"),
		).Example("Workload metrics", "Collect workload metrics from Run AI with a scrape interval of 30 seconds.", `
input:
  run_ai:
    url: "${RUNAI_URL:}"
    app_id: "${RUNAI_APP_ID:}"
    app_secret: "${RUNAI_APP_SECRET:}"
    schedule: "*/30 * * * * *"
    metrics_offset: "30s"
    resource_type: "workload"
    page_size: 500
    metrics:
      - CPU_LIMIT_CORES
      - CPU_MEMORY_LIMIT_BYTES
      - CPU_MEMORY_REQUEST_BYTES
      - CPU_MEMORY_USAGE_BYTES
      - CPU_REQUEST_CORES
      - CPU_USAGE_CORES
      - GPU_ALLOCATION
      - GPU_MEMORY_REQUEST_BYTES
      - GPU_MEMORY_USAGE_BYTES
      - GPU_UTILIZATION
      - POD_COUNT
      - RUNNING_POD_COUNT
    http:
      timeout: 30s
      retry_count: 1
      retry_wait_time: 100ms
      retry_max_wait_time: 1s
`)
}

func init() {
	err := service.RegisterBatchInput("run_ai", runAIInputConfig(), func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) {
		httpMetrics := mgr.Metrics().NewTimer("run_ai_http_request_ns", "url", "status_code")
		resourceTypeMetrics := mgr.Metrics().NewGauge("run_ai_resource_count", "type")
		in, err := newRunAIInput(conf, mgr, httpMetrics, resourceTypeMetrics)
		if err != nil {
			return nil, err
		}

		in.timingMetrics = httpMetrics

		return in, nil
	})
	if err != nil {
		panic(err)
	}
}

var _ service.BatchInput = (*runAIInput)(nil)

type runAIInput struct {
	service       *runai.Service
	resourceType  string
	metrics       []runai.MetricType
	interval      time.Duration
	schedule      string
	metricsOffset time.Duration
	scheduler     gocron.Scheduler
	store         map[time.Time][]runai.ResourceWithMetrics
	mu            sync.Mutex
	resources     *service.Resources
	logger        *service.Logger

	timingMetrics       *service.MetricTimer
	resourceTypeMetrics *service.MetricGauge
}

func newRunAIInput(conf *service.ParsedConfig, resources *service.Resources, httpMetrics *service.MetricTimer, resourceTypeMetrics *service.MetricGauge) (*runAIInput, error) {
	logger := resources.Logger().With("component", "run_ai")

	url, err := conf.FieldString(fieldURL)
	if err != nil {
		return nil, err
	}

	appID, err := conf.FieldString(fieldAppID)
	if err != nil {
		return nil, err
	}

	appSecret, err := conf.FieldString(fieldAppSecret)
	if err != nil {
		return nil, err
	}

	resourceType, err := conf.FieldString(fieldResourceType)
	if err != nil {
		return nil, err
	}

	metrics, err := conf.FieldStringList(fieldMetrics)
	if err != nil {
		return nil, err
	}

	schedule, err := conf.FieldString(fieldSchedule)
	if err != nil {
		return nil, err
	}

	metricsOffset, err := conf.FieldDuration(fieldMetricsOffset)
	if err != nil {
		return nil, err
	}

	pageSize, err := conf.FieldInt(fieldPageSize)
	if err != nil {
		return nil, err
	}

	if pageSize < 100 || pageSize > 500 {
		return nil, errors.New("page size must be between 100 and 500")
	}

	var interval time.Duration
	{
		// Create a cron scheduler
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		cronSchedule, err := parser.Parse(schedule)
		if err != nil {
			return nil, err
		}

		// Get current time in UTC
		now := time.Now().UTC()

		// Get next two occurrences
		nextRun := cronSchedule.Next(now)
		secondRun := cronSchedule.Next(nextRun)

		// Calculate the duration between runs
		interval = secondRun.Sub(nextRun)
	}

	requestTimeout, err := conf.FieldDuration(fieldHTTPConfig, fieldHTTPTimeout)
	if err != nil {
		return nil, err
	}

	retryCount, err := conf.FieldInt(fieldHTTPConfig, fieldHTTPRetryCount)
	if err != nil {
		return nil, err
	}

	retryWaitTime, err := conf.FieldDuration(fieldHTTPConfig, fieldHTTPRetryWaitTime)
	if err != nil {
		return nil, err
	}

	retryMaxWaitTime, err := conf.FieldDuration(fieldHTTPConfig, fieldHTTPRetryMaxWaitTime)
	if err != nil {
		return nil, err
	}

	service, err := runai.NewService(url, appID, appSecret, resources.Logger(), runai.ServiceConfig{
		Timeout:             requestTimeout,
		RetryCount:          retryCount,
		RetryWaitTime:       retryWaitTime,
		RetryMaxWaitTime:    retryMaxWaitTime,
		PageSize:            pageSize,
		TimingMetrics:       httpMetrics,
		ResourceTypeMetrics: resourceTypeMetrics,
	})
	if err != nil {
		return nil, err
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &runAIInput{
		service:       service,
		resourceType:  resourceType,
		interval:      interval,
		schedule:      schedule,
		metricsOffset: metricsOffset,
		scheduler:     scheduler,
		metrics: lo.Map(metrics, func(metric string, _ int) runai.MetricType {
			return runai.MetricType(metric)
		}),
		store:     make(map[time.Time][]runai.ResourceWithMetrics),
		mu:        sync.Mutex{},
		resources: resources,
		logger:    logger,
	}, nil
}

// scrape scrapes the metrics for the given time and adds them to the store.
func (in *runAIInput) scrape(ctx context.Context, t time.Time) error {
	in.logger.Debugf("scraping %s metrics between %s and %s", in.resourceType, t.Add(-in.interval).Format(time.RFC3339), t.Format(time.RFC3339))

	startTime := t.Add(-in.interval).Add(-in.metricsOffset)
	endTime := t.Add(-in.metricsOffset)

	switch in.resourceType {
	case "workload":
		workloadsWithMetrics, err := in.service.GetAllWorkloadWithMetrics(ctx, runai.MeasurementParams{
			MetricType: in.metrics,
			StartTime:  startTime,
			EndTime:    endTime,
		})
		if err != nil {
			return err
		}

		in.mu.Lock()
		in.store[t] = lo.Map(workloadsWithMetrics, func(workloadWithMetrics runai.WorkloadWithMetrics, _ int) runai.ResourceWithMetrics {
			return &workloadWithMetrics
		})
		in.mu.Unlock()
	case "pod":
		podsWithMetrics, err := in.service.GetAllPodWithMetrics(ctx, runai.MeasurementParams{
			MetricType: in.metrics,
			StartTime:  startTime,
			EndTime:    endTime,
		})
		if err != nil {
			return err
		}

		in.mu.Lock()
		in.store[t] = lo.Map(podsWithMetrics, func(podWithMetrics runai.PodWithMetrics, _ int) runai.ResourceWithMetrics {
			return &podWithMetrics
		})
		in.mu.Unlock()
	}

	return nil
}

func (in *runAIInput) Connect(ctx context.Context) error {
	// Add a job to the scheduler
	_, err := in.scheduler.NewJob(
		gocron.CronJob(in.schedule, true),
		gocron.NewTask(
			func(ctx context.Context) error {
				// Get current time in UTC
				t := time.Now().UTC()
				err := in.scrape(ctx, t)
				if err != nil {
					in.logger.Errorf("error scraping metrics: %v", err)
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

	go func() {
		running := false
		for {
			select {
			case <-ctx.Done():
				if running {
					_ = in.scheduler.StopJobs()
				}
				return
			case <-time.After(1 * time.Second):
				switch leaderelection.IsLeader(in.resources) {
				case false:
					if running {
						err := in.scheduler.StopJobs()
						if err != nil {
							in.logger.Errorf("error stopping jobs: %v", err)
						}
						running = false
					}
				case true:
					if !running {
						in.scheduler.Start()
						running = true
					}
				}
			}
		}
	}()

	return nil
}

func (in *runAIInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	if len(in.store) == 0 {
		return nil, func(context.Context, error) error { return nil }, nil
	}

	in.mu.Lock()
	defer in.mu.Unlock()

	processing := make(map[time.Time][]runai.ResourceWithMetrics)
	batch := make([]*service.Message, 0)

	for t, resourceWithMetrics := range in.store {
		in.logger.Tracef("reading metrics of %s", t.Format(time.RFC3339))

		for _, resourceWithMetrics := range resourceWithMetrics {
			encoded, err := json.Marshal(resourceWithMetrics)
			if err != nil {
				return nil, func(context.Context, error) error { return nil }, err
			}

			msg := service.NewMessage(encoded)
			msg.MetaSet("scrape_time", t.Format(time.RFC3339))
			msg.MetaSet("scrape_interval", in.interval.String())
			msg.MetaSet("metrics_offset", in.metricsOffset.String())
			msg.MetaSet("resource_type", in.resourceType)
			msg.MetaSet("metrics_time", resourceWithMetrics.GetMetrics().Timestamp.Format(time.RFC3339))
			batch = append(batch, msg)
		}

		processing[t] = resourceWithMetrics
		delete(in.store, t)
	}

	in.logger.Debugf("read %d metrics", len(batch))

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

func (in *runAIInput) Close(ctx context.Context) error {
	return in.scheduler.Shutdown()
}
