package main

const (
	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goVersion = "1.22.5"

	kafkaVersion      = "3.6"
	clickhouseVersion = "24.5.5.78"
	redisVersion      = "7.0.12"
	postgresVersion   = "14.9"

	// TODO: add update mechanism for versions below

	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goBuildVersion = goVersion + "-alpine3.19@sha256:0642d4f809abf039440540de1f0e83502401686e3946ed8e7398a1d94648aa6d"
	xxBaseImage    = "tonistiigi/xx:1.4.0@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4"

	alpineBaseImage = "alpine:3.20.1@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0"
)
