# Testing OpenTracing use a Kafka Tranport with Zipkin

## Prerequirements

* Golang 1.8^
* Docker-Compose

## Geting Started

Run Kafka and Zipkin service with docker compose
`docker-compose up -d`

Run http trace client
`go run main.go`

Open a browser and go to Zipkin UI at ` http:\\localhost:9411`
 

