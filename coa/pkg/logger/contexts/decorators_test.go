package contexts

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestActivityLogContextDecorator_Decorate(t *testing.T) {
	d := &ActivityLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	entry = entry.WithContext(ctx)
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	innerEntry := entry.Data[string(ActivityLogContextKey)]
	assert.NotNil(t, innerEntry)

	innerCtx := innerEntry.(*ActivityLogContext)

	assert.Equal(t, "operationId", innerCtx.operationId)
	assert.Equal(t, "callerId", innerCtx.callerId)
	assert.Equal(t, "resourceCloudId", innerCtx.resourceCloudId)
	assert.Equal(t, "resourceK8SId", innerCtx.resourceK8SId)
}

func TestDiagnosticLogContextDecorator_Decorate(t *testing.T) {
	d := &DiagnosticLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	entry = entry.WithContext(ctx)
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	innerEntry := entry.Data[string(DiagnosticLogContextKey)]
	assert.NotNil(t, innerEntry)

	innerCtx := innerEntry.(*DiagnosticLogContext)

	assert.Equal(t, "traceId", innerCtx.traceId)
	assert.Equal(t, "spanId", innerCtx.spanId)
	assert.Equal(t, "correlationId", innerCtx.correlationId)
	assert.Equal(t, "requestId", innerCtx.requestId)
}

func TestActivityLogContextDecorator_DecorateWithKey(t *testing.T) {
	d := &ActivityLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	entry = entry.WithField(string(ActivityLogContextKey), ctx)
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	innerEntry := entry.Data[string(ActivityLogContextKey)]
	assert.NotNil(t, innerEntry)

	innerCtx := innerEntry.(*ActivityLogContext)

	assert.Equal(t, "operationId", innerCtx.operationId)
	assert.Equal(t, "callerId", innerCtx.callerId)
	assert.Equal(t, "resourceCloudId", innerCtx.resourceCloudId)
	assert.Equal(t, "resourceK8SId", innerCtx.resourceK8SId)
}

func TestDiagnosticLogContextDecorator_DecorateWithKey(t *testing.T) {
	d := &DiagnosticLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	entry = entry.WithField(string(DiagnosticLogContextKey), ctx)
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	innerEntry := entry.Data[string(DiagnosticLogContextKey)]
	assert.NotNil(t, innerEntry)

	innerCtx := innerEntry.(*DiagnosticLogContext)

	assert.Equal(t, "traceId", innerCtx.traceId)
	assert.Equal(t, "spanId", innerCtx.spanId)
	assert.Equal(t, "correlationId", innerCtx.correlationId)
	assert.Equal(t, "requestId", innerCtx.requestId)
}

func TestActivityLogContextDecorator_DecorateWithNil(t *testing.T) {
	d := &ActivityLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	assert.Nil(t, entry.Data[string(ActivityLogContextKey)])
}

func TestDiagnosticLogContextDecorator_DecorateWithNil(t *testing.T) {
	d := &DiagnosticLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	assert.Nil(t, entry.Data[string(DiagnosticLogContextKey)])
}

func TestActivityLogContextDecorator_DecorateWithInvalidContext(t *testing.T) {
	d := &ActivityLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	entry = entry.WithContext(context.TODO())
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	assert.Nil(t, entry.Data[string(ActivityLogContextKey)])
}

func TestDiagnosticLogContextDecorator_DecorateWithInvalidContext(t *testing.T) {
	d := &DiagnosticLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	entry = entry.WithContext(context.TODO())
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	assert.Nil(t, entry.Data[string(DiagnosticLogContextKey)])
}

func TestActivityLogContextDecorator_DecorateWithInvalidKey(t *testing.T) {
	d := &ActivityLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	entry = entry.WithField("invalid", ctx)
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	assert.Nil(t, entry.Data[string(ActivityLogContextKey)])
}

func TestDiagnosticLogContextDecorator_DecorateWithInvalidKey(t *testing.T) {
	d := &DiagnosticLogContextDecorator{}
	entry := logrus.NewEntry(logrus.StandardLogger())
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	entry = entry.WithField("invalid", ctx)
	entry = d.Decorate(entry)
	assert.NotNil(t, entry)
	assert.Nil(t, entry.Data[string(DiagnosticLogContextKey)])
}
