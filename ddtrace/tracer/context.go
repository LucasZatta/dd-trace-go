// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	"context"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	traceinternal "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/internal"
	"gopkg.in/DataDog/dd-trace-go.v1/internal"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/orchestrion"
)

// ContextWithSpan returns a copy of the given context which includes the span s.
func ContextWithSpan(ctx context.Context, s Span) context.Context {
	return orchestrion.CtxWithValue(ctx, internal.ActiveSpanKey, s)
}

// SpanFromContext returns the span contained in the given context. A second return
// value indicates if a span was found in the context. If no span is found, a no-op
// span is returned.
func SpanFromContext(ctx context.Context) (Span, bool) {
	if ctx == nil {
		return &traceinternal.NoopSpan{}, false
	}
	v := orchestrion.WrapContext(ctx).Value(internal.ActiveSpanKey)
	switch s := v.(type) {
	case *traceinternal.NoopSpan:
		return s, false
	case traceinternal.NoopSpan:
		return s, false
	case ddtrace.Span:
		return s, true
	default:
		return traceinternal.NoopSpan{}, false
	}
}

// StartSpanFromContext returns a new span with the given operation name and options. If a span
// is found in the context, it will be used as the parent of the resulting span. If the ChildOf
// option is passed, it will only be used as the parent if there is no span found in `ctx`.
func StartSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	// copy opts in case the caller reuses the slice in parallel
	// we will add at least 1, at most 2 items
	optsLocal := make([]StartSpanOption, len(opts), len(opts)+2)
	copy(optsLocal, opts)

	if ctx == nil {
		// default to context.Background() to avoid panics on Go >= 1.15
		ctx = context.Background()
	} else if s, ok := SpanFromContext(ctx); ok {
		optsLocal = append(optsLocal, ChildOf(s.Context()))
	}
	optsLocal = append(optsLocal, withContext(ctx))
	s := StartSpan(operationName, optsLocal...)
	if span, ok := s.(*span); ok && span.pprofCtxActive != nil {
		// If pprof labels were applied for this span, use the derived ctx that
		// includes them. Otherwise a child of this span wouldn't be able to
		// correctly restore the labels of its parent when it finishes.
		ctx = span.pprofCtxActive
	}
	return s, ContextWithSpan(ctx, s)
}
