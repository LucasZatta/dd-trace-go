// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Package gin provides functions to trace the gin-gonic/gin package (https://github.com/gin-gonic/gin).
package gin // import "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"

import (
	"fmt"
	"math"

	"gopkg.in/DataDog/dd-trace-go.v1/contrib/internal/httptrace"
	"gopkg.in/DataDog/dd-trace-go.v1/contrib/internal/options"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/telemetry"

	"github.com/gin-gonic/gin"
)

const componentName = "gin-gonic/gin"

func init() {
	telemetry.LoadIntegration(componentName)
	tracer.MarkIntegrationImported("github.com/gin-gonic/gin")
}

// Middleware returns middleware that will trace incoming requests. If service is empty then the
// default service name will be used.
func Middleware(service string, opts ...Option) gin.HandlerFunc {
	cfg := newConfig(service)
	for _, opt := range opts {
		opt(cfg)
	}
	log.Debug("contrib/gin-gonic/gin: Configuring Middleware: Service: %s, %#v", cfg.serviceName, cfg)
	spanOpts := []tracer.StartSpanOption{
		tracer.ServiceName(cfg.serviceName),
		tracer.Tag(ext.Component, componentName),
		tracer.Tag(ext.SpanKind, ext.SpanKindServer),
	}
	return func(c *gin.Context) {
		if cfg.ignoreRequest(c) {
			return
		}
		opts := options.Copy(spanOpts...) // opts must be a copy of cfg.spanOpts, locally scoped, to avoid races.
		opts = append(opts, tracer.ResourceName(cfg.resourceNamer(c)))
		if !math.IsNaN(cfg.analyticsRate) {
			opts = append(opts, tracer.Tag(ext.EventSampleRate, cfg.analyticsRate))
		}
		opts = append(opts, tracer.Tag(ext.HTTPRoute, c.FullPath()))
		opts = append(opts, httptrace.HeaderTagsFromRequest(c.Request, cfg.headerTags))
		sctx, err := tracer.Extract(tracer.HTTPHeadersCarrier(c.Request.Header))
		if err != nil {
			//dont know yet
		}

		opts = append(opts, tracer.ChildOf(sctx))

		span, ctx, finishSpans := httptrace.StartRequestSpan(c.Request, opts...)
		defer func() {
			finishSpans(c.Writer.Status(), nil)
		}()

		// pass the span through the request context
		c.Request = c.Request.WithContext(ctx)

		// Use AppSec if enabled by user
		if appsec.Enabled() {
			useAppSec(c, span)
		}

		// serve the request to the next middleware
		c.Next()

		if len(c.Errors) > 0 {
			span.SetTag("gin.errors", c.Errors.String())
		}
	}
}

// HTML will trace the rendering of the template as a child of the span in the given context.
func HTML(c *gin.Context, code int, name string, obj interface{}) {
	span, _ := tracer.StartSpanFromContext(c.Request.Context(), "gin.render.html")
	span.SetTag("go.template", name)
	span.SetTag(ext.Component, componentName)
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("error rendering tmpl:%s: %s", name, r)
			span.Finish(tracer.WithError(err))
			panic(r)
		} else {
			span.Finish()
		}
	}()
	c.HTML(code, name, obj)
}
