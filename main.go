package main

import (
	"context"
	"flag"
	"net/http"
	"time"

	"github.com/Shopify/sarama"
	opentracing "github.com/opentracing/opentracing-go"
	// opentracing "github.com/opentracing/opentracing-go"
	// opentracing "github.com/opentracing/opentracing-go"
	// // "github.com/opentracing-contrib/go-stdlib/nethttp"
	// // opentracing "github.com/opentracing/opentracing-go"
	// // "github.com/opentracing/opentracing-go/ext"
	// otlog "github.com/opentracing/opentracing-go/log"
	zipkingo "github.com/evo3cx/zipkin-go-opentracing"
	// // "github.com/uber/jaeger-client-go"
)

var (
	brokers = []string{"localhost:9092"}
)

func main() {

	flag.Parse()
	producer := kafkaClient()
	createCollector(producer)
}

func kafkaClient() sarama.AsyncProducer {
	config := sarama.NewConfig()
	config.Producer.Retry.Max = 5
	config.Producer.RequiredAcks = sarama.WaitForAll

	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		panic(err)
	}

	return producer
}

func createCollector(prod sarama.AsyncProducer) {
	collector, err := zipkingo.NewKafkaCollector(brokers, zipkingo.KafkaProducer(prod))
	if err != nil {
		panic(err)
	}

	recorder := zipkingo.NewRecorder(collector, true, "hostPort", "fresh-ask-service")
	tracer, err := zipkingo.NewTracer(
		recorder,
		zipkingo.ClientServerSameSpan(false),
		zipkingo.TraceID128Bit(true),
	)
	if err != nil {
		panic(err)
	}

	// Explicitly set our tracer to be the default tracer.
	opentracing.InitGlobalTracer(tracer)

	// Create Root Span for duration of the interaction with svc1
	span := opentracing.StartSpan("Run")
	// Put root span in context so it will be used in our calls to the client.
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	// Call the Ask Google
	span.LogEvent("Ask Google")
	if err = AskGoogle(ctx); err != nil {
		panic("ARRR google" + err.Error())
	}

	// Call the Ask Quora
	span.LogEvent("Ask Quora")
	if err = AskQuora(ctx); err != nil {
		panic("ARRR google" + err.Error())
	}

	span.SetOperationName("ask_service")

	// Finish our CLI span
	span.Finish()

	// Close collector to ensure spans are sent before exiting.
	collector.Close()
}

func AskGoogle(ctx context.Context) error {

	// retrieve current Span from Context
	var parentCtx opentracing.SpanContext
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan != nil {
		parentCtx = parentSpan.Context()
	}

	// start a new Span to wrap HTTP request
	span := opentracing.StartSpan(
		"ask google",
		opentracing.ChildOf(parentCtx),
	)
	time.Sleep(time.Second * 2)
	// make sure the Span is finished once we're done
	defer span.Finish()

	// make the Span current in the context
	ctx = opentracing.ContextWithSpan(ctx, span)

	// now prepare the request
	req, err := http.NewRequest("GET", "http://google.com", nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	// execute the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	span = span.SetTag("db.type", "redis")
	span = span.SetTag("db.statement", "test")
	span = span.SetTag("span.kind", "client")

	// Google home page is not too exciting, so ignore the result
	res.Body.Close()
	return nil
}

func AskQuora(ctx context.Context) error {

	// retrieve current Span from Context
	var parentCtx opentracing.SpanContext
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan != nil {
		parentCtx = parentSpan.Context()
	}

	// start a new Span to wrap HTTP request
	span := opentracing.StartSpan(
		"ask quora",
		opentracing.ChildOf(parentCtx),
	)
	time.Sleep(time.Second * 2)
	// make sure the Span is finished once we're done
	defer span.Finish()

	// make the Span current in the context
	ctx = opentracing.ContextWithSpan(ctx, span)

	// now prepare the request
	req, err := http.NewRequest("GET", "http://quora.com", nil)
	if err != nil {
		return err
	}

	// execute the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	span = span.SetTag("db.type", "redis")
	span = span.SetTag("db.statement", "test")
	span = span.SetTag("span.kind", "client")

	span.LogKV(
		"event", "error",
		"message", "vc",
		"latency", "20s",
	)
	// Quora home page is not too exciting, so ignore the result
	res.Body.Close()
	return nil
}
