package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/transport/zipkin"
	"golang.org/x/net/context"
)

var (
	zipkinURL = flag.String("url",
		"http://localhost:9411/api/v1/spans", "Zipkin server URL")
	serverPort = flag.String("port", "8000", "server port")
	actorKind  = flag.String("actor", "server", "server or client")
)

const (
	server = "server"
	client = "client"
)

func main() {

	flag.Parse()

	if *actorKind != server && *actorKind != client {
		log.Fatal("Please specify '-actor server' or '-actor client'")
	}

	// Jaeger tracer can be initialized with a transport that will
	// report tracing Spans to a Zipkin backend
	transport, err := zipkin.NewHTTPTransport(
		*zipkinURL,
		zipkin.HTTPBatchSize(1),
		zipkin.HTTPLogger(jaeger.StdLogger),
	)
	if err != nil {
		log.Fatalf("Cannot initialize HTTP transport: %v", err)
	}
	// create Jaeger tracer
	tracer, closer := jaeger.NewTracer(
		*actorKind,
		jaeger.NewConstSampler(true), // sample all traces
		jaeger.NewRemoteReporter(transport, nil),
	)

	if *actorKind == server {
		runServer(tracer)
		return
	}

	runClient(tracer)

	// Close the tracer to guarantee that all spans that could
	// be still buffered in memory are sent to the tracing backend
	closer.Close()
}

func runClient(tracer opentracing.Tracer) {
	// nethttp.Transport from go-stdlib will do the tracing
	c := &http.Client{Transport: &nethttp.Transport{}}

	// create a top-level span to represent full work of the client
	span := tracer.StartSpan(client)
	span.SetTag(string(ext.Component), client)
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("http://localhost:%s/", *serverPort),
		nil,
	)
	if err != nil {
		onError(span, err)
		return
	}

	req = req.WithContext(ctx)
	// wrap the request in nethttp.TraceRequest
	req, ht := nethttp.TraceRequest(tracer, req)
	defer ht.Finish()

	res, err := c.Do(req)
	if err != nil {
		onError(span, err)
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		onError(span, err)
		return
	}
	fmt.Printf("Received result: %s\n", string(body))
}

func onError(span opentracing.Span, err error) {
	// handle errors by recording them in the span
	span.SetTag(string(ext.Error), true)
	span.LogKV(otlog.Error(err))
	log.Print(err)
}

func getTime(w http.ResponseWriter, r *http.Request) {
	log.Print("Received getTime request")
	t := time.Now()
	ts := t.Format("Mon Jan _2 15:04:05 2006")
	io.WriteString(w, fmt.Sprintf("The time is %s", ts))
}

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r,
		fmt.Sprintf("http://localhost:%s/gettime", *serverPort), 301)
}

func runServer(tracer opentracing.Tracer) {
	http.HandleFunc("/gettime", getTime)
	http.HandleFunc("/", redirect)
	log.Printf("Starting server on port %s", *serverPort)
	err := http.ListenAndServe(
		fmt.Sprintf(":%s", *serverPort),
		// use nethttp.Middleware to enable OpenTracing for server
		nethttp.Middleware(tracer, http.DefaultServeMux))
	if err != nil {
		log.Fatalf("Cannot start server: %s", err)
	}
}
