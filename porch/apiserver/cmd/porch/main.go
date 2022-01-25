// Copyright 2022 Google LLC
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
	"os"

	"github.com/GoogleContainerTools/kpt/porch/apiserver/pkg/cmd/server"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
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

	stopCh := genericapiserver.SetupSignalHandler()

	options := server.NewPorchServerOptions(os.Stdout, os.Stderr)
	cmd := server.NewCommandStartPorchServer(options, stopCh)
	code := cli.Run(cmd)
	return code
}

type telemetry struct {
	tp       *trace.TracerProvider
	pusher   *basic.Controller
	exporter trace.SpanExporter
}

func (t *telemetry) Start() error {
	config := os.Getenv("OTEL")
	if config == "" {
		return nil
	}

	if config == "stdout" {
		exportOpts := []stdout.Option{
			stdout.WithPrettyPrint(),
		}
		// Registers both a trace and meter Provider globally.
		tracerProvider, pusher, err := stdout.InstallNewPipeline(exportOpts, nil)
		if err != nil {
			return fmt.Errorf("error initializing stdout exporter: %w", err)
		}

		t.tp = tracerProvider
		t.pusher = pusher
		return nil
	}

	if config == "otel" {
		ctx := context.Background()

		// See https://github.com/open-telemetry/opentelemetry-go/issues/1484
		driver := otlpgrpc.NewDriver(
			otlpgrpc.WithInsecure(),
			otlpgrpc.WithEndpoint("localhost:4317"),
			otlpgrpc.WithDialOption(grpc.WithBlock()), // useful for testing
		)

		// set global propagator to tracecontext (the default is no-op).
		otel.SetTextMapPropagator(propagation.TraceContext{})

		// Registers both a trace and meter Provider globally.
		exporter, tracerProvider, pusher, err := otlp.InstallNewPipeline(ctx, driver)
		if err != nil {
			return fmt.Errorf("error initializing otel exporter: %w", err)
		}

		t.tp = tracerProvider
		t.pusher = pusher
		t.exporter = exporter
		return nil
	}

	return fmt.Errorf("unknown OTEL configuration %q", config)
}

func (t *telemetry) Stop() {
	if t.pusher != nil {
		if err := t.pusher.Stop(context.Background()); err != nil {
			klog.Warningf("failed to shut down telemetry: %v", err)
		}
		t.pusher = nil
	}

	if t.tp != nil {
		if err := t.tp.Shutdown(context.Background()); err != nil {
			klog.Warningf("failed to shut down telemetry: %v", err)
		}
		t.tp = nil
	}

	if t.exporter != nil {
		if err := t.exporter.Shutdown(context.Background()); err != nil {
			klog.Warningf("failed to shut down telemetry exporter: %v", err)
		}
		t.exporter = nil
	}
}
