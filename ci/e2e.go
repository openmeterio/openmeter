package main

import (
	"fmt"
)

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
