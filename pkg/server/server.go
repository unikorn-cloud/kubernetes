/*
Copyright 2022-2024 EscherCloud.
Copyright 2024 the Unikorn Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"flag"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler"
	"github.com/unikorn-cloud/unikorn/pkg/server/middleware/cors"
	"github.com/unikorn-cloud/unikorn/pkg/server/middleware/openapi"
	"github.com/unikorn-cloud/unikorn/pkg/server/middleware/opentelemetry"
	"github.com/unikorn-cloud/unikorn/pkg/server/middleware/timeout"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Server struct {
	// Options are server specific options e.g. listener address etc.
	Options Options

	// ZapOptions configure logging.
	ZapOptions zap.Options

	// HandlerOptions sets options for the HTTP handler.
	HandlerOptions handler.Options

	// AuthorizerOptions allow configuration of the OIDC backend.
	AuthorizerOptions openapi.Options

	// CORSOptions are for remote resource sharing.
	CORSOptions cors.Options
}

func (s *Server) AddFlags(goflags *flag.FlagSet, flags *pflag.FlagSet) {
	s.ZapOptions.BindFlags(goflags)

	s.Options.AddFlags(flags)
	s.HandlerOptions.AddFlags(flags)
	s.AuthorizerOptions.AddFlags(flags)
	s.CORSOptions.AddFlags(flags)
}

func (s *Server) SetupLogging() {
	log.SetLogger(zap.New(zap.UseFlagOptions(&s.ZapOptions)))
}

// SetupOpenTelemetry adds a span processor that will print root spans to the
// logs by default, and optionally ship the spans to an OTLP listener.
// TODO: move config into an otel specific options struct.
func (s *Server) SetupOpenTelemetry(ctx context.Context) error {
	otel.SetLogger(log.Log)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	opts := []trace.TracerProviderOption{
		trace.WithSpanProcessor(&opentelemetry.LoggingSpanProcessor{}),
	}

	if s.Options.OTLPEndpoint != "" {
		exporter, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(s.Options.OTLPEndpoint),
			otlptracehttp.WithInsecure(),
		)

		if err != nil {
			return err
		}

		opts = append(opts, trace.WithBatcher(exporter))
	}

	otel.SetTracerProvider(trace.NewTracerProvider(opts...))

	return nil
}

func (s *Server) GetServer(client client.Client) (*http.Server, error) {
	schema, err := openapi.NewSchema()
	if err != nil {
		return nil, err
	}

	// Middleware specified here is applied to all requests pre-routing.
	router := chi.NewRouter()
	router.Use(timeout.Middleware(s.Options.RequestTimeout))
	router.Use(opentelemetry.Middleware())
	router.Use(cors.Middleware(schema, &s.CORSOptions))
	router.NotFound(http.HandlerFunc(handler.NotFound))
	router.MethodNotAllowed(http.HandlerFunc(handler.MethodNotAllowed))

	// Setup middleware.
	authorizer := openapi.NewAuthorizer(&s.AuthorizerOptions)

	// Middleware specified here is applied to all requests post-routing.
	// NOTE: these are applied in reverse order!!
	chiServerOptions := generated.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.HandleError,
		Middlewares: []generated.MiddlewareFunc{
			openapi.Middleware(authorizer, schema),
		},
	}

	handlerInterface, err := handler.New(client, &s.HandlerOptions)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		Addr:              s.Options.ListenAddress,
		ReadTimeout:       s.Options.ReadTimeout,
		ReadHeaderTimeout: s.Options.ReadHeaderTimeout,
		WriteTimeout:      s.Options.WriteTimeout,
		Handler:           generated.HandlerWithOptions(handlerInterface, chiServerOptions),
	}

	return server, nil
}
