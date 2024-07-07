package contexts

import (
	"context"
	"encoding/json"
	"time"
)

// DiagnosticLogContext is a context that holds diagnostic information.
type DiagnosticLogContext struct {
	traceId       string
	spanId        string
	correlationId string
	requestId     string
}

func NewDiagnosticLogContext(traceId, spanId, correlationId, requestId string) *DiagnosticLogContext {
	return &DiagnosticLogContext{
		traceId:       traceId,
		spanId:        spanId,
		correlationId: correlationId,
		requestId:     requestId,
	}
}

func (ctx *DiagnosticLogContext) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"trace-id":       ctx.traceId,
		"span-id":        ctx.spanId,
		"correlation-id": ctx.correlationId,
		"request-id":     ctx.requestId,
	}
}

func (ctx *DiagnosticLogContext) FromMap(m map[string]interface{}) {
	if m == nil {
		return
	}
	if m["trace-id"] != nil {
		ctx.traceId = m["trace-id"].(string)
	}
	if m["span-id"] != nil {
		ctx.spanId = m["span-id"].(string)
	}
	if m["correlation-id"] != nil {
		ctx.correlationId = m["correlation-id"].(string)
	}
	if m["request-id"] != nil {
		ctx.requestId = m["request-id"].(string)
	}
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
	case "trace-id":
		return ctx.traceId
	case "span-id":
		return ctx.spanId
	case "correlation-id":
		return ctx.correlationId
	case "request-id":
		return ctx.requestId
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

func (ctx *DiagnosticLogContext) GetTraceId() string {
	return ctx.traceId
}

func (ctx *DiagnosticLogContext) GetSpanId() string {
	return ctx.spanId
}

func (ctx *DiagnosticLogContext) GetCorrelationId() string {
	return ctx.correlationId
}

func (ctx *DiagnosticLogContext) GetRequestId() string {
	return ctx.requestId
}

func (ctx *DiagnosticLogContext) SetTraceId(traceId string) {
	ctx.traceId = traceId
}

func (ctx *DiagnosticLogContext) SetSpanId(spanId string) {
	ctx.spanId = spanId
}

func (ctx *DiagnosticLogContext) SetCorrelationId(correlationId string) {
	ctx.correlationId = correlationId
}

func (ctx *DiagnosticLogContext) SetRequestId(requestId string) {
	ctx.requestId = requestId
}

func PopulateTraceAndSpanToDiagnosticLogContext(traceId string, spanId string, parent context.Context) context.Context {
	if parent == nil {
		diagCtx := NewDiagnosticLogContext(traceId, spanId, "", "")
		return context.WithValue(context.Background(), DiagnosticLogContextKey, diagCtx)
	}
	if diagCtx, ok := parent.Value(DiagnosticLogContextKey).(*DiagnosticLogContext); ok {
		diagCtx.SetTraceId(traceId)
		diagCtx.SetSpanId(spanId)
		return context.WithValue(parent, DiagnosticLogContextKey, diagCtx)
	} else {
		diagCtx := NewDiagnosticLogContext(traceId, spanId, "", "")
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
