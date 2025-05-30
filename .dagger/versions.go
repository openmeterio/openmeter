package main

const (
	golangciLintVersion = "v2.0.0"
	helmVersion         = "3.16.4"
	helmDocsVersion     = "v1.14.2"
	spectralVersion     = "6.13.1"
)

const (
	// COREPACK_VERSION defines the corepack version to be used in CI pipelines
	// NOTE: Temporary measure to overcome the following issue: https://github.com/nodejs/corepack/issues/612
	COREPACK_VERSION = "0.31.0"

	// NODEJS_CONTAINER_IMAGE defines the container image to be used for Nodejs.
	NODEJS_CONTAINER_IMAGE = "node:22.13-alpine3.21@sha256:e2b39f7b64281324929257d0f8004fb6cb4bf0fdfb9aa8cedb235a766aec31da"

	// AtlasContainerImage defines the container image to be used for testing database migrations.
	AtlasContainerImage = "arigaio/atlas:0.32.1@sha256:faff97dab166a6ccd0e76b724355d579b54c2e86386f5a3ef7582513ec07ad40"
)
