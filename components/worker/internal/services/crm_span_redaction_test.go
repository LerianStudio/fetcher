// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	workerCrypto "github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"
	portDS "github.com/LerianStudio/fetcher/pkg/ports/datasource"
	observability "github.com/LerianStudio/lib-observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/mock/gomock"
)

// crmHostPort is the topology detail the MongoDB driver embeds in its connection /
// server-selection errors. The test asserts it NEVER reaches an exported span.
const crmHostPort = "mongo-crm.internal:27017"

// TestQueryPluginCRMDatabase_ConnectError_DoesNotLeakHostPortOnSpan is the FIX-3
// (HIGH) regression guard. The raw MongoDB driver connect error embeds host:port;
// sanitizeSpanMessage only strips Bearer/Basic, so feeding the raw error to
// HandleSpanError would leak host:port onto an exported span. After FIX-3 the CRM
// path records a STATIC, host-free message on the span (mirroring the generic
// engine adapter), keeping the verbatim error only in the returned error.
func TestQueryPluginCRMDatabase_ConnectError_DoesNotLeakHostPortOnSpan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Real SDK tracer with an in-memory recorder so we can inspect exported spans.
	recorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx := observability.ContextWithTracer(context.Background(), tp.Tracer("test"))

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	// MongoDB datasource whose Connect fails with a driver-style error embedding host:port.
	connectErr := errors.New("server selection error: context deadline exceeded, current topology: { Type: Single, Servers: [{ Addr: " + crmHostPort + ", Type: Unknown }] }")
	mockDS := modelDatasource.NewMockDataSource(ctrl)
	mockDS.EXPECT().Connect(gomock.Any(), gomock.Any()).Return(connectErr)
	// Connect failed, so the datasource must be released on the error path.
	mockDS.EXPECT().Close(gomock.Any()).Return(nil)
	uc.SetDataSourceFactory(func(context.Context, *model.Connection, workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
		return mockDS, nil
	})

	message := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{"plugin_crm": {"holders": {"id"}}},
	}
	connections := []*model.Connection{{ConfigName: "plugin_crm", Type: model.TypeMongoDB, Host: "mongo-crm.internal", Port: 27017}}
	result := make(map[string]map[string][]map[string]any)

	err := uc.queryPluginCRMDatabase(ctx, message, connections, result)

	// The returned error MAY (and does) carry the verbatim driver detail — that is the
	// host's own error, not an exported span.
	require.Error(t, err)
	require.Contains(t, err.Error(), crmHostPort, "returned error should keep the verbatim driver detail for local diagnostics")

	// The EXPORTED span must NOT leak host:port in its status or recorded events.
	spans := recorder.Ended()
	require.NotEmpty(t, spans, "expected at least one exported span")

	for _, span := range spans {
		assert.NotContains(t, span.Status().Description, crmHostPort,
			"span status leaked host:port: %q", span.Status().Description)
		assert.NotContains(t, span.Status().Description, "mongo-crm.internal",
			"span status leaked host: %q", span.Status().Description)

		for _, ev := range span.Events() {
			for _, attr := range ev.Attributes {
				val := attr.Value.AsString()
				assert.NotContains(t, val, crmHostPort, "span event %q leaked host:port: %q", ev.Name, val)
				assert.NotContains(t, val, "mongo-crm.internal", "span event %q leaked host: %q", ev.Name, val)
			}
			assert.False(t, strings.Contains(ev.Name, crmHostPort), "span event name leaked host:port")
		}
	}
}

// assertNoHostPortOnSpans verifies that no exported span leaks the host:port (or
// the bare host) in its status description or recorded error events, mirroring the
// assertions in the Connect-failure test above.
func assertNoHostPortOnSpans(t *testing.T, spans []trace.ReadOnlySpan) {
	t.Helper()

	require.NotEmpty(t, spans, "expected at least one exported span")

	for _, span := range spans {
		assert.NotContains(t, span.Status().Description, crmHostPort,
			"span status leaked host:port: %q", span.Status().Description)
		assert.NotContains(t, span.Status().Description, "mongo-crm.internal",
			"span status leaked host: %q", span.Status().Description)

		for _, ev := range span.Events() {
			for _, attr := range ev.Attributes {
				val := attr.Value.AsString()
				assert.NotContains(t, val, crmHostPort, "span event %q leaked host:port: %q", ev.Name, val)
				assert.NotContains(t, val, "mongo-crm.internal", "span event %q leaked host: %q", ev.Name, val)
			}
			assert.False(t, strings.Contains(ev.Name, crmHostPort), "span event name leaked host:port")
		}
	}
}

// spanStatusContains reports whether any exported span's status description
// contains the given substring. Used to assert that the STATIC host-free message
// is still recorded (the event is redacted, not suppressed). HandleSpanError
// formats the description as "<message>: <error>", so a substring match is used.
func spanStatusContains(spans []trace.ReadOnlySpan, substr string) bool {
	for _, span := range spans {
		if strings.Contains(span.Status().Description, substr) {
			return true
		}
	}

	return false
}

// TestQueryPluginCRM_ListCollectionsError_DoesNotLeakHostPortOnSpan guards the
// ListCollectionNames sink. The driver's ListCollectionNames error wraps the
// MongoDB topology error (which embeds host:port); feeding it raw to
// HandleSpanError would leak host:port onto an exported span. The fix records a
// STATIC, host-free message on the span while keeping the verbatim error only in
// the returned error and the local log.
func TestQueryPluginCRM_ListCollectionsError_DoesNotLeakHostPortOnSpan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx := observability.ContextWithTracer(context.Background(), tp.Tracer("test"))

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	// Connect SUCCEEDS (crmCompatDataSource.Connect returns nil); ListCollectionNames
	// fails with a driver-style error embedding host:port.
	listErr := errors.New("server selection error: context deadline exceeded, current topology: { Type: Single, Servers: [{ Addr: " + crmHostPort + ", Type: Unknown }] }")
	mockCRM := portDS.NewMockCRMQueryable(ctrl)
	mockCRM.EXPECT().ListCollectionNames(gomock.Any()).Return(nil, listErr)
	uc.SetDataSourceFactory(func(context.Context, *model.Connection, workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
		return crmCompatDataSource{MockCRMQueryable: mockCRM}, nil
	})

	message := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{"plugin_crm": {"holders": {"id"}}},
	}
	connections := []*model.Connection{{ConfigName: "plugin_crm", Type: model.TypeMongoDB, Host: "mongo-crm.internal", Port: 27017}}
	result := make(map[string]map[string][]map[string]any)

	err := uc.queryPluginCRMDatabase(ctx, message, connections, result)

	// The returned error keeps the verbatim driver detail for local diagnostics.
	require.Error(t, err)
	require.Contains(t, err.Error(), crmHostPort, "returned error should keep the verbatim driver detail for local diagnostics")

	spans := recorder.Ended()
	assertNoHostPortOnSpans(t, spans)

	// The STATIC host-free message must still be recorded on a span (redacted, not dropped).
	assert.True(t, spanStatusContains(spans, "Error listing plugin_crm collections"),
		"the static, host-free message must still be recorded on the span")
}

// TestQueryPluginCRM_QueryCollectionError_DoesNotLeakHostPortOnSpan guards the
// collection-processing sink. ListCollectionNames succeeds and the prefix fan-out
// resolves a physical collection, so the real processing chain reaches
// QueryCollection, which fails with a driver-style error embedding host:port.
// Feeding that raw error to HandleSpanError on the QueryPluginCRM span would leak
// host:port; the fix records a STATIC, host-free message instead.
func TestQueryPluginCRM_QueryCollectionError_DoesNotLeakHostPortOnSpan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	recorder := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx := observability.ContextWithTracer(context.Background(), tp.Tracer("test"))

	mocks := newTestMocks(ctrl)
	uc := newTestUseCase(mocks)

	// Connect + ListCollectionNames succeed; the fan-out returns a collection that
	// matches the "holders_" prefix so the real processing chain reaches the query.
	// QueryCollection (no filters present) fails with a host-embedding error.
	queryErr := errors.New("connection() error occurred during connection handshake: dial tcp " + crmHostPort + ": connect: connection refused")
	mockCRM := portDS.NewMockCRMQueryable(ctrl)
	mockCRM.EXPECT().ListCollectionNames(gomock.Any()).Return([]string{"holders_org-1"}, nil)
	mockCRM.EXPECT().QueryCollection(gomock.Any(), "holders_org-1", gomock.Any(), gomock.Any()).Return(nil, queryErr)
	uc.SetDataSourceFactory(func(context.Context, *model.Connection, workerCrypto.Cryptor) (modelDatasource.DataSource, error) {
		return crmCompatDataSource{MockCRMQueryable: mockCRM}, nil
	})

	message := ExtractExternalDataMessage{
		JobID:        newTestJobID(),
		MappedFields: map[string]map[string][]string{"plugin_crm": {"holders": {"id"}}},
	}
	connections := []*model.Connection{{ConfigName: "plugin_crm", Type: model.TypeMongoDB, Host: "mongo-crm.internal", Port: 27017}}
	result := make(map[string]map[string][]map[string]any)

	err := uc.queryPluginCRMDatabase(ctx, message, connections, result)

	// The returned error keeps the verbatim driver detail for local diagnostics.
	require.Error(t, err)
	require.Contains(t, err.Error(), crmHostPort, "returned error should keep the verbatim driver detail for local diagnostics")

	spans := recorder.Ended()
	assertNoHostPortOnSpans(t, spans)

	// The STATIC host-free message must still be recorded on a span (redacted, not dropped).
	assert.True(t, spanStatusContains(spans, "Error processing plugin_crm collection"),
		"the static, host-free message must still be recorded on the span")
}
