// Copyright 2022 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/pkg/cmd/server"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
	"k8s.io/klog/v2"
)

func main() {
	code := run()
	os.Exit(code)
}

func run() int {
	t := &telemetry{}
	t.Start()
	defer t.Stop()

	http.DefaultTransport = otelhttp.NewTransport(http.DefaultClient.Transport)
	http.DefaultClient.Transport = http.DefaultTransport

	ctx := genericapiserver.SetupSignalContext()

	options := server.NewPorchServerOptions(os.Stdout, os.Stderr)
	cmd := server.NewCommandStartPorchServer(ctx, options)
	code := cli.Run(cmd)
	return code
}

type telemetry struct {
	tp *trace.TracerProvider
}

func (t *telemetry) Start() error {
	config := os.Getenv("OTEL")
	if config == "" {
		return nil
	}

	if config == "stdout" {
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return fmt.Errorf("error initializing stdout exporter: %w", err)
		}
		t.tp = trace.NewTracerProvider(trace.WithBatcher(exporter))

		otel.SetTracerProvider(t.tp)
		return nil
	}

	// TODO: Is there any convention here?
	if strings.HasPrefix(config, "otel://") {
		ctx := context.Background()

		u, err := url.Parse(config)
		if err != nil {
			return fmt.Errorf("error parsing url %q: %w", config, err)
		}

		endpoint := u.Host

		klog.Infof("tracing to %q", config)

		// See https://github.com/open-telemetry/opentelemetry-go/issues/1484
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return fmt.Errorf("failed to create gRPC connection to collector: %w", err)
		}

		// Set up a trace exporter
		traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			return fmt.Errorf("failed to create trace exporter: %w", err)
		}
		// Register the trace exporter with a TracerProvider, using a batch
		// span processor to aggregate spans before export.
		bsp := trace.NewBatchSpanProcessor(traceExporter)
		t.tp = trace.NewTracerProvider(
			trace.WithSpanProcessor(bsp),
		)
		otel.SetTracerProvider(t.tp)

		// set global propagator to tracecontext (the default is no-op).
		otel.SetTextMapPropagator(propagation.TraceContext{})

		return nil
	}

	return fmt.Errorf("unknown OTEL configuration %q", config)
}

func (t *telemetry) Stop() {
	if t.tp != nil {
		if err := t.tp.Shutdown(context.Background()); err != nil {
			klog.Warningf("failed to shut down telemetry: %v", err)
		}
		t.tp = nil
	}
}
