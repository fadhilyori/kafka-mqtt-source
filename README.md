# Kafka MQTT Source

Simple app to collect messages from MQTT broker (Mosquitto) and send it to Apache Kafka

## Requirements
 - [Golang](https://go.dev/dl)
 - make (linux)

## Build with Makefile
 - Show the make help: `make help`
 - Build the binary: `make build`
 - The binary will be available at `out/bin` directory.

## Run using go command
 - `go mod download`
 - `go run main.go`

## Show help
`me-mqtt-source --help`
