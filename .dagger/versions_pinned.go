package main

const (
	kafkaVersion      = "3.6"
	clickhouseVersion = "24.10"
	redisVersion      = "7.0.12"
	postgresVersion   = "14.9"
	svixVersion       = "v1.44"

	// TODO: add update mechanism for versions below

	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goBuildVersion = "1.25.1-alpine3.22@sha256:b6ed3fd0452c0e9bcdef5597f29cc1418f61672e9d3a2f55bf02e7222c014abd"
	xxBaseImage    = "tonistiigi/xx:1.6.1@sha256:923441d7c25f1e2eb5789f82d987693c47b8ed987c4ab3b075d6ed2b5d6779a3"

	alpineBaseImage = "alpine:3.22@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412"
)
