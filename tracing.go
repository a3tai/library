package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// tracerName is the instrumentation library identifier reported on every
// span. Use a single name so all spans cluster under one service in
// downstream UIs.
const tracerName = "a3t-library"

// initTracing wires up an OpenTelemetry tracer provider based on the
// environment:
//
//	LIBRARY_OTLP_ENDPOINT — if set (e.g. "http://localhost:4318"), spans
//	  are exported over OTLP/HTTP. Use a local collector or a hosted
//	  backend (Honeycomb, etc.).
//	LIBRARY_TRACE_STDOUT  — if "1"/"true", spans are pretty-printed to
//	  stderr. Useful for one-off debugging without standing up a
//	  collector.
//
// If neither is set, the global tracer is a no-op and the otel.Tracer
// calls in the rest of the codebase are essentially free.
//
// Returns a shutdown function that flushes pending spans. main() should
// call it on exit.
func initTracing(ctx context.Context) func() {
	endpoint := firstEnv("LIBRARY_OTLP_ENDPOINT")
	stdout := envBool("LIBRARY_TRACE_STDOUT")

	if endpoint == "" && !stdout {
		return func() {}
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("a3t-library"),
			semconv.ServiceVersion(AppVersion),
		),
		resource.WithHost(),
		resource.WithProcess(),
	)
	if err != nil {
		log.Printf("[trace] resource init failed: %v", err)
		return func() {}
	}

	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}

	if endpoint != "" {
		// Strip scheme — otlptracehttp wants host:port without it. Keep
		// `insecure` since local collectors usually run plain HTTP.
		host := strings.TrimPrefix(endpoint, "http://")
		host = strings.TrimPrefix(host, "https://")
		exp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(host),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			log.Printf("[trace] OTLP HTTP exporter init failed: %v", err)
		} else {
			opts = append(opts, sdktrace.WithBatcher(exp))
			log.Printf("[trace] OTLP/HTTP exporter → %s", endpoint)
		}
	}

	if stdout {
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint(), stdouttrace.WithWriter(os.Stderr))
		if err != nil {
			log.Printf("[trace] stdout exporter init failed: %v", err)
		} else {
			// SimpleSpanProcessor for stdout — pretty-print immediately
			// instead of batching, which keeps the dev log readable.
			opts = append(opts, sdktrace.WithSyncer(exp))
			log.Printf("[trace] stdout exporter enabled")
		}
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("[trace] shutdown error: %v", err)
		}
	}
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func envBool(keys ...string) bool {
	value := firstEnv(keys...)
	return value == "1" || strings.EqualFold(value, "true")
}

func envUnsetOrNotZero(keys ...string) bool {
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if ok {
			return strings.TrimSpace(value) != "0"
		}
	}
	return true
}

// tracer returns the package-level tracer. Cheap (cached internally by
// otel) — safe to call in hot paths.
func tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// spanErr records err on the current span if non-nil. Idempotent and
// safe with a no-op tracer.
func spanErr(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
