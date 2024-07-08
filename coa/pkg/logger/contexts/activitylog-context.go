package contexts

import (
	"encoding/json"
	"time"
)

// ActivityLogContext is a context that holds activity information.
type ActivityLogContext struct {
	operationId     string
	callerId        string
	resourceCloudId string
	resourceK8SId   string
}

func NewActivityLogContext(operationId, callerId, resourceCloudId, resourceK8SId string) *ActivityLogContext {
	return &ActivityLogContext{
		operationId:     operationId,
		callerId:        callerId,
		resourceCloudId: resourceCloudId,
		resourceK8SId:   resourceK8SId,
	}
}

func (ctx *ActivityLogContext) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"operation-id":      ctx.operationId,
		"caller-id":         ctx.callerId,
		"resource-cloud-id": ctx.resourceCloudId,
		"resource-k8s-id":   ctx.resourceK8SId,
	}
}

func (ctx *ActivityLogContext) FromMap(m map[string]interface{}) {
	if m == nil {
		return
	}
	if m["operation-id"] != nil {
		ctx.operationId = m["operation-id"].(string)
	}
	if m["caller-id"] != nil {
		ctx.callerId = m["caller-id"].(string)
	}
	if m["resource-cloud-id"] != nil {
		ctx.resourceCloudId = m["resource-cloud-id"].(string)
	}
	if m["resource-k8s-id"] != nil {
		ctx.resourceK8SId = m["resource-k8s-id"].(string)
	}
}

func (ctx *ActivityLogContext) String() string {
	b, _ := json.Marshal(ctx.ToMap())
	return string(b)
}

// Deadline returns the time when work done on behalf of this context
func (ctx *ActivityLogContext) Deadline() (deadline time.Time, ok bool) {
	// No deadline set
	return time.Time{}, false
}

// Done returns a channel that's closed when work done on behalf of this context should be canceled.
func (ctx *ActivityLogContext) Done() <-chan struct{} {
	// No cancellation set
	return nil
}

// Err returns an error if this context has been canceled or timed out.
func (a *ActivityLogContext) Err() error {
	// No error set
	return nil
}

// Value returns the value associated with this context for key, or nil if no value is associated with key.
func (ctx *ActivityLogContext) Value(key interface{}) interface{} {
	switch key {
	case "operation-id":
		return ctx.operationId
	case "caller-id":
		return ctx.callerId
	case "resource-cloud-id":
		return ctx.resourceCloudId
	case "resource-k8s-id":
		return ctx.resourceK8SId
	default:
		return nil
	}
}

func (ctx *ActivityLogContext) MarshalJSON() ([]byte, error) {
	return json.Marshal(ctx.ToMap())
}

func (ctx *ActivityLogContext) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	ctx.FromMap(m)
	return nil
}

func (ctx *ActivityLogContext) GetOperationId() string {
	return ctx.operationId
}

func (ctx *ActivityLogContext) GetCallerId() string {
	return ctx.callerId
}

func (ctx *ActivityLogContext) GetResourceCloudId() string {
	return ctx.resourceCloudId
}

func (ctx *ActivityLogContext) GetResourceK8SId() string {
	return ctx.resourceK8SId
}

func (ctx *ActivityLogContext) SetOperationId(operationId string) {
	ctx.operationId = operationId
}

func (ctx *ActivityLogContext) SetCallerId(callerId string) {
	ctx.callerId = callerId
}

func (ctx *ActivityLogContext) SetResourceCloudId(resourceCloudId string) {
	ctx.resourceCloudId = resourceCloudId
}

func (ctx *ActivityLogContext) SetResourceK8SId(resourceK8SId string) {
	ctx.resourceK8SId = resourceK8SId
}
