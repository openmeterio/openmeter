package main

import (
	"fmt"
	"path"
	"time"
)

const (
	perfTestDefaultConfig = "dagger.config.yaml"
)

func (m *Ci) Perf(configName Optional[string]) *Container {
	localStack := NewLocalStack(m.Source, m.Source.File(path.Join("perf", "configs", configName.GetOr(perfTestDefaultConfig))))

	// build k6 tests with node20 and pnpm
	testBuilder := dag.Container().
		From("node:20-alpine").
		WithMountedDirectory("/mnt/src", m.Source).
		WithWorkdir("/mnt/src/perf/k6").
		WithExec([]string{"pnpm", "install"}).
		WithExec([]string{"pnpm", "build"})

	k6Container := dag.Container().
		From("grafana/k6:latest").
		WithDirectory("/tests/dist", testBuilder.Directory("/mnt/src/perf/k6/dist")).
		WithFile("/tests/run-all.sh", m.Source.File("perf/k6/run-all.sh")).
		WithWorkdir("/tests").
		WithServiceBinding("api", localStack.Api).
		WithServiceBinding("clickhouse", localStack.Clickhouse).
		WithEnvVariable("CLICKHOUSE_BASE_URL", "http://clickhouse:8123").
		WithEnvVariable("OPENMETER_BASE_URL", "http://api:8888").
		WithEnvVariable("OPENMETER_TELEMETRY_URL", "http://api:10000").
		WithEntrypoint([]string{"/bin/ash", "-c"})

	// seeding the system
	dag.Container().
		From("jeffail/benthos:latest").
		WithMountedDirectory("/mnt/src", m.Source).
		WithServiceBinding("api", localStack.Api).
		WithEnvVariable("OPENMETER_BASE_URL", "http://api:8888").
		WithEnvVariable("SEEDER_COUNT", "100").
		WithExec([]string{"-c", "/mnt/src/perf/configs/seed.benthos.yaml"})

	// TODO: add querying clickhouse to get the number of records instead
	time.Sleep(10 * time.Second)

	return k6Container.WithExec([]string{"./run-all.sh"})
}

func (m *Ci) Etoe(test Optional[string]) *Container {
	localStack := NewLocalStack(m.Source, m.Source.File("e2e/config.yaml"))

	args := []string{"go", "test", "-v"}

	if t, ok := test.Get(); ok {
		args = append(args, "-run", fmt.Sprintf("Test%s", t))
	}

	args = append(args, "./e2e/...")

	return dag.Go(GoOpts{
		Container: goModule().
			WithSource(m.Source).
			Container().
			WithServiceBinding("api", localStack.Api).
			WithServiceBinding("sink-worker", localStack.SinkWorker).
			WithEnvVariable("OPENMETER_ADDRESS", "http://api:8080"),
	}).
		Exec(args)
}

type AppStack struct {
	Api        *Service
	SinkWorker *Service
	Clickhouse *Service
	Redis      *Service
	Kafka      *Service
}

func NewLocalStack(source *Directory, omConfig *File) *AppStack {
	builder := &Build{
		Source: source,
	}

	configPath := "/etc/openmeter/config.yaml"

	kafka := dag.Kafka(KafkaOpts{Version: kafkaVersion}).Service()
	clickhouse := clickhouse()
	redis := redis()

	base := builder.ContainerImage("").
		WithExposedPort(10000).
		WithMountedFile(configPath, omConfig).
		WithServiceBinding("kafka", kafka).
		WithServiceBinding("clickhouse", clickhouse)

	api := base.
		WithExposedPort(8080).
		WithExec([]string{"openmeter", "--config", configPath}).
		AsService()

	sinkWorker := base.
		WithServiceBinding("redis", redis).
		WithServiceBinding("api", api). // Make sure api is up before starting sink worker
		WithExec([]string{"openmeter-sink-worker", "--config", configPath}).
		AsService()

	return &AppStack{
		Api:        api,
		SinkWorker: sinkWorker,
		Clickhouse: clickhouse,
		Redis:      redis,
		Kafka:      kafka,
	}
}

func clickhouse() *Service {
	return dag.Container().
		From(fmt.Sprintf("clickhouse/clickhouse-server:%s-alpine", clickhouseVersion)).
		WithEnvVariable("CLICKHOUSE_USER", "default").
		WithEnvVariable("CLICKHOUSE_PASSWORD", "default").
		WithEnvVariable("CLICKHOUSE_DB", "openmeter").
		WithExposedPort(9000).
		WithExposedPort(9009).
		WithExposedPort(8123).
		AsService()
}

func redis() *Service {
	return dag.Container().
		From(fmt.Sprintf("redis:%s-alpine", redisVersion)).
		WithExposedPort(6379).
		AsService()
}
