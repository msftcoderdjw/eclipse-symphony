package contexts

import (
	"context"
	"encoding/json"
	"time"
)

const (
	Diagnostics_CorrelationId        string = "correlationId"
	Diagnostics_ResourceCloudId      string = "resourceId"
	Diagnostics_TraceContext         string = "traceContext"
	Diagnostics_TraceContext_TraceId string = "traceId"
	Diagnostics_TraceContext_SpanId  string = "spanId"
)

type TraceContext struct {
	traceId string
	spanId  string
}

// DiagnosticLogContext is a context that holds diagnostic information.
type DiagnosticLogContext struct {
	correlationId   string
	resourceCloudId string
	traceContext    TraceContext
}

func NewDiagnosticLogContext(correlationId, resourceCloudId, traceId, spanId string) *DiagnosticLogContext {
	return &DiagnosticLogContext{
		correlationId:   correlationId,
		resourceCloudId: resourceCloudId,
		traceContext: TraceContext{
			traceId: traceId,
			spanId:  spanId,
		},
	}
}

func (ctx *DiagnosticLogContext) ToMap() map[string]interface{} {
	return map[string]interface{}{
		Diagnostics_CorrelationId:   ctx.correlationId,
		Diagnostics_ResourceCloudId: ctx.resourceCloudId,
		Diagnostics_TraceContext: map[string]interface{}{
			Diagnostics_TraceContext_TraceId: ctx.traceContext.traceId,
			Diagnostics_TraceContext_SpanId:  ctx.traceContext.spanId,
		},
	}
}

func (ctx *DiagnosticLogContext) FromMap(m map[string]interface{}) {
	if m == nil {
		return
	}
	if m[Diagnostics_CorrelationId] != nil {
		ctx.correlationId = m[Diagnostics_CorrelationId].(string)
	}
	if m[Diagnostics_ResourceCloudId] != nil {
		ctx.resourceCloudId = m[Diagnostics_ResourceCloudId].(string)
	}
	if m[Diagnostics_TraceContext] != nil {
		traceContext := m[Diagnostics_TraceContext].(map[string]interface{})
		if traceContext[Diagnostics_TraceContext_TraceId] != nil {
			ctx.traceContext.traceId = traceContext[Diagnostics_TraceContext_TraceId].(string)
		}
		if traceContext[Diagnostics_TraceContext_SpanId] != nil {
			ctx.traceContext.spanId = traceContext[Diagnostics_TraceContext_SpanId].(string)
		}
	}
}

func (ctx *DiagnosticLogContext) String() string {
	b, _ := json.Marshal(ctx.ToMap())
	return string(b)
}

// Deadline returns the time when work done on behalf of this context
func (ctx *DiagnosticLogContext) Deadline() (deadline time.Time, ok bool) {
	// No deadline set
	return time.Time{}, false
}

// Done returns a channel that's closed when work done on behalf of this context should be canceled.
func (ctx *DiagnosticLogContext) Done() <-chan struct{} {
	// No cancellation set
	return nil
}

// Err returns an error if this context has been canceled or timed out.
func (a *DiagnosticLogContext) Err() error {
	// No error set
	return nil
}

// Value returns the value associated with this context for key, or nil if no value is associated with key.
func (ctx *DiagnosticLogContext) Value(key interface{}) interface{} {
	switch key {
	case Diagnostics_CorrelationId:
		return ctx.correlationId
	case Diagnostics_ResourceCloudId:
		return ctx.resourceCloudId
	case Diagnostics_TraceContext:
		return ctx.traceContext
	default:
		return nil
	}
}

func (ctx *DiagnosticLogContext) MarshalJSON() ([]byte, error) {
	return json.Marshal(ctx.ToMap())
}

func (ctx *DiagnosticLogContext) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	ctx.FromMap(m)
	return nil
}

func (ctx *DiagnosticLogContext) SetCorrelationId(correlationId string) {
	ctx.correlationId = correlationId
}

func (ctx *DiagnosticLogContext) SetResourceId(resourceCloudId string) {
	ctx.resourceCloudId = resourceCloudId
}

func (ctx *DiagnosticLogContext) SetTraceId(traceId string) {
	ctx.traceContext.traceId = traceId
}

func (ctx *DiagnosticLogContext) SetSpanId(spanId string) {
	ctx.traceContext.spanId = spanId
}

func (ctx *DiagnosticLogContext) SetTraceContext(traceContext TraceContext) {
	ctx.traceContext = traceContext
}

func (ctx *DiagnosticLogContext) GetCorrelationId() string {
	return ctx.correlationId
}

func (ctx *DiagnosticLogContext) GetResourceId() string {
	return ctx.resourceCloudId
}

func (ctx *DiagnosticLogContext) GetTraceId() string {
	return ctx.traceContext.traceId
}

func (ctx *DiagnosticLogContext) GetSpanId() string {
	return ctx.traceContext.spanId
}

func (ctx *DiagnosticLogContext) GetTraceContext() TraceContext {
	return ctx.traceContext
}

func PopulateResourceIdAndCorrelationIdToDiagnosticLogContext(correlationId string, resourceCloudId string, parent context.Context) context.Context {
	if parent == nil {
		diagCtx := NewDiagnosticLogContext(correlationId, resourceCloudId, "", "")
		return context.WithValue(context.Background(), DiagnosticLogContextKey, diagCtx)
	}
	if diagCtx, ok := parent.Value(DiagnosticLogContextKey).(*DiagnosticLogContext); ok {
		diagCtx.SetCorrelationId(correlationId)
		diagCtx.SetResourceId(resourceCloudId)
		return context.WithValue(parent, DiagnosticLogContextKey, diagCtx)
	} else {
		diagCtx := NewDiagnosticLogContext(correlationId, resourceCloudId, "", "")
		return context.WithValue(parent, DiagnosticLogContextKey, diagCtx)
	}
}

func ClearResourceIdAndCorrelationIdFromDiagnosticLogContext(parent *context.Context) {
	if parent == nil {
		return
	}
	if diagCtx, ok := (*parent).Value(DiagnosticLogContextKey).(*DiagnosticLogContext); ok {
		if diagCtx != nil {
			diagCtx.SetCorrelationId("")
			diagCtx.SetResourceId("")
		}
	}
}

func PopulateTraceAndSpanToDiagnosticLogContext(traceId string, spanId string, parent context.Context) context.Context {
	if parent == nil {
		diagCtx := NewDiagnosticLogContext("", "", traceId, spanId)
		return context.WithValue(context.Background(), DiagnosticLogContextKey, diagCtx)
	}
	if diagCtx, ok := parent.Value(DiagnosticLogContextKey).(*DiagnosticLogContext); ok {
		diagCtx.SetTraceId(traceId)
		diagCtx.SetSpanId(spanId)
		return context.WithValue(parent, DiagnosticLogContextKey, diagCtx)
	} else {
		diagCtx := NewDiagnosticLogContext("", "", traceId, spanId)
		return context.WithValue(parent, DiagnosticLogContextKey, diagCtx)
	}
}

func ClearTraceAndSpanFromDiagnosticLogContext(parent *context.Context) {
	if parent == nil {
		return
	}
	if diagCtx, ok := (*parent).Value(DiagnosticLogContextKey).(*DiagnosticLogContext); ok {
		if diagCtx != nil {
			diagCtx.SetTraceId("")
			diagCtx.SetSpanId("")
		}
	}
}
