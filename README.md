# Kafka MQTT Source

Simple app to collect messages from MQTT broker (Mosquitto) and send it to Apache Kafka

## Requirements
 - [Golang 1.17 or more](https://go.dev/dl)
 - make (linux)

## Build with Makefile
 - Show the make help: `make help`
 - Build the binary: `make build`
 - The binary will be available at `out/bin` directory.
 - Build docker image: `make build-docker-image`

## Configure main.go
## Run using go command
 - `go mod download`
 - `go run cmd/main.go`

## Show help
`me-mqtt-source --help`
