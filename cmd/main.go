package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("MQTT Client Connected.")
}

var connectionLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection Lost: %s\n", err.Error())
}

func validatePort(p int) (int, error) {
	const minPort, maxPort = 1, 65535
	if p < minPort || p > maxPort {
		return p, fmt.Errorf("port %d out of range [%d:%d]", p, minPort, maxPort)
	}

	return p, nil
}

func main() {
	var (
		successCount = 0
		messageCount = 0
	)

	mqttBrokerHost := flag.String("H", "127.0.0.1", "MQTT Broker Host")
	mqttBrokerPort := flag.Int("P", 1883, "MQTT Broker Port")
	mqttBrokerUsername := flag.String("u", "", "MQTT Broker Username")
	mqttBrokerPassword := flag.String("p", "", "MQTT Broker Password")
	mqttTopic := flag.String("t", "mataelang/sensor/v3/#", "MQTT Broker Topic")
	kafkaBootstrapServers := flag.String("K", "127.0.0.1:9092", "Kafka Boostrap Server")
	kafkaTopic := flag.String("T", "sensor_events", "Kafka Target Topic")
	useEnvVar := flag.Bool("b", false, "Wheter to use flag or environment variable")
	verboseLog := flag.Bool("v", false, "Verbose payload to stdout")
	statsIntervalSec := flag.Int("d", 10, "Log Statistics interval in second")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	if *useEnvVar {
		var err error
		*mqttBrokerHost = os.Getenv("MQTT_HOST")
		*mqttBrokerPort, err = strconv.Atoi(os.Getenv("MQTT_PORT"))
		if err != nil {
			*mqttBrokerPort = 1883
		}

		*mqttBrokerUsername = os.Getenv("MQTT_USERNAME")
		*mqttBrokerPassword = os.Getenv("MQTT_PASSWORD")

		*mqttTopic = os.Getenv("MQTT_TOPIC")
		if mt := os.Getenv("MQTT_TOPIC"); mt != "" {
			*mqttTopic = mt
		}

		if ks := os.Getenv("KAFKA_BOOSTRAP_SERVERS"); ks != "" {
			*kafkaBootstrapServers = ks
		}

		if kt := os.Getenv("KAFKA_PRODUCE_TOPIC"); kt != "" {
			*kafkaTopic = kt
		}
	}

	if _, err := validatePort(*mqttBrokerPort); err != nil {
		log.Fatal(err)
	}

	log.Printf("Loading configuration.")

	mqttClientID := uuid.New().String()

	log.Printf("MQTT Broker Host\t: %s\n", *mqttBrokerHost)
	log.Printf("MQTT Broker Port\t: %d\n", *mqttBrokerPort)
	log.Printf("MQTT Broker Topic\t: %s\n", *mqttTopic)

	var broker = fmt.Sprintf("tcp://%s:%d", *mqttBrokerHost, *mqttBrokerPort)

	options := mqtt.NewClientOptions()
	options.AddBroker(broker)
	options.SetClientID(mqttClientID)
	if *mqttBrokerUsername != "" {
		options.SetUsername(*mqttBrokerUsername)
		options.SetPassword(*mqttBrokerPassword)
	}
	options.OnConnect = connectHandler
	options.OnConnectionLost = connectionLostHandler

	client := mqtt.NewClient(options)
	token := client.Connect()

	if token.Wait() && token.Error() != nil {
		log.Fatalln(token.Error())
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": *kafkaBootstrapServers,
		"client.id":         "me-mqtt-source",
		"acks":              "all",
	})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	messages := make(chan map[string]interface{})
	delivery_chan := make(chan kafka.Event, 10000)

	log.Printf("Start consuming logs.")

	p := message.NewPrinter(language.AmericanEnglish)

	// Create routine for sending message from messages channel
	go func() {
		for textLine := range messages {
			messageCount += 1
			payload, err := json.Marshal(textLine)
			if err != nil {
				log.Println(err)
			}
			_ = producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: kafkaTopic, Partition: kafka.PartitionAny},
				Value:          []byte(payload)},
				delivery_chan,
			)
		}
	}()

	ticker := time.NewTicker(time.Duration(*statsIntervalSec) * time.Second)
	quit := make(chan struct{})

	go func() {
		for e := range delivery_chan {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Failed to deliver message: %v\n", ev.TopicPartition)

				} else {
					successCount += 1
					if *verboseLog {
						log.Printf("Successfully produced record to topic %s partition [%d] @ offset %v\n",
							*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
					}
				}
			}
		}
	}()

	t := client.Subscribe(*mqttTopic, 1, func(c mqtt.Client, m mqtt.Message) {
		var payload map[string]interface{}
		err := json.Unmarshal(m.Payload(), &payload)
		if err != nil {
			log.Printf("ERROR - Cannot parse the event log")
			return
		}

		if *verboseLog {
			log.Printf("PAYLOAD - %s\n", fmt.Sprint(payload))
		}

		messages <- payload
	})
	if t.Wait() && t.Error() != nil {
		log.Fatalln(t.Error())
	}

	log.Printf("Subscribed to topic: %s", *mqttTopic)

	for {
		select {
		case <-ticker.C:
			tempMessageCount := messageCount
			tempSuccessCount := successCount
			messageCount = 0
			successCount = 0
			tempErrorCount := tempMessageCount - tempSuccessCount
			tempMessageRate := tempMessageCount / *statsIntervalSec
			log.Printf("Total=%d\tSuccess=%d\tFailed=%d\tAvgRate=%s message/second\n", tempMessageCount, tempSuccessCount, tempErrorCount, p.Sprintf("%v", tempMessageRate))
		case <-quit:
			ticker.Stop()
			return
		}
	}
}
