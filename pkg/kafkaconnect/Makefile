openapi.yaml:
	curl https://kafka.apache.org/35/generated/connect_rest.yaml > openapi.yaml

.PHONY: generate
generate: openapi.yaml
	go generate
