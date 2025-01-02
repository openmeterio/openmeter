package main

const (
	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goVersion = "1.23.4"

	kafkaVersion      = "3.6"
	clickhouseVersion = "24.5.5.78"
	redisVersion      = "7.0.12"
	postgresVersion   = "14.9"
	svixVersion       = "v1.44"

	// TODO: add update mechanism for versions below

	// Alpine is required for our current build (due to Kafka and CGO), but it doesn't seem to work well with golangci-lint
	goBuildVersion = "1.23.4-alpine3.21@sha256:6c5c9590f169f77c8046e45c611d3b28fe477789acd8d3762d23d4744de69812"
	xxBaseImage    = "tonistiigi/xx:1.6.1@sha256:923441d7c25f1e2eb5789f82d987693c47b8ed987c4ab3b075d6ed2b5d6779a3"

	alpineBaseImage = "alpine:3.21.0@sha256:21dc6063fd678b478f57c0e13f47560d0ea4eeba26dfc947b2a4f81f686b9f45"

	atlasImage = "arigaio/atlas:0.29.1-alpine@sha256:c81f42fb734ce70de3b27268d88902c2a6f493c6a9b72149623920e228597846"
)
