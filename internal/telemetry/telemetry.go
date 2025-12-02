package telemetry

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"kabsa/internal/config"
	"kabsa/internal/logging"
	"os"
	"time"
)

type ShutdownFunc func(ctx context.Context) error

// Setup configures OpenTelemetry (traces + metrics) and returns a shutdown function.
// It sends data via OTLP/gRPC to an OTel Collector.
func Setup(
	ctx context.Context,
	cfg config.ObservabilityConfig,
	logger logging.Logger,
) (ShutdownFunc, error) {
	// 1) Resource: service name, env, and common attributes.
	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			attribute.String("deployment.environment", cfg.ServiceEnv),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	// 2) Resolve OTLP endpoint (collector).
	endpoint := cfg.OtelEndpoint
	if endpoint == "" {
		if e := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); e != "" {
			endpoint = e
		} else {
			// Default to local collector
			endpoint = "localhost:4317"
		}
	}

	// Shared gRPC options.
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// 3) Trace exporter + provider.
	traceExp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpcOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// 4) Metric exporter + provider.
	metricExp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithDialOption(grpcOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("create metric exporter: %w", err)
	}

	// Periodic reader from the sdk/metric package (no separate reader subpackage)
	metricReader := sdkmetric.NewPeriodicReader(
		metricExp,
		sdkmetric.WithInterval(10*time.Second),
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(metricReader),
	)
	otel.SetMeterProvider(mp)

	logger.Info("otel configured",
		"otlp_endpoint", endpoint,
		"service_name", cfg.ServiceName,
		"service_env", cfg.ServiceEnv,
	)

	// 5) Shutdown function flushes traces & metrics.
	shutdown := func(ctx context.Context) error {
		var firstErr error

		// TracerProvider shutdown.
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown tracer provider", "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}

		// MeterProvider shutdown.
		if err := mp.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown meter provider", "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}

		return firstErr
	}

	return shutdown, nil
}
