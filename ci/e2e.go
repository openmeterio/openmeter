// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"path"
)

func (m *Ci) Etoe(test Optional[string]) *Container {
	image := m.Build().ContainerImage("").
		WithExposedPort(10000).
		WithMountedFile("/etc/openmeter/config.yaml", dag.Host().File(path.Join(root(), "e2e", "config.yaml"))).
		WithServiceBinding("kafka", dag.Kafka(KafkaOpts{Version: kafkaVersion}).Service()).
		WithServiceBinding("clickhouse", clickhouse())

	api := image.
		WithExposedPort(8080).
		WithExec([]string{"openmeter", "--config", "/etc/openmeter/config.yaml"}).
		AsService()

	sinkWorker := image.
		WithServiceBinding("redis", redis()).
		WithServiceBinding("api", api). // Make sure api is up before starting sink worker
		WithExec([]string{"openmeter-sink-worker", "--config", "/etc/openmeter/config.yaml"}).
		AsService()

	args := []string{"go", "test", "-v"}

	if t, ok := test.Get(); ok {
		args = append(args, "-run", fmt.Sprintf("Test%s", t))
	}

	args = append(args, "./e2e/...")

	return dag.Go(GoOpts{
		Container: goModule().
			WithSource(m.Source).
			Container().
			WithServiceBinding("api", api).
			WithServiceBinding("sink-worker", sinkWorker).
			WithEnvVariable("OPENMETER_ADDRESS", "http://api:8080"),
	}).
		Exec(args)
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
