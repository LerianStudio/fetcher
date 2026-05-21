package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"time"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	libConstants "github.com/LerianStudio/lib-commons/v5/commons/constants"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
)

const HeaderTenantID = "X-Tenant-ID"

// BuildSecurePublishing creates the canonical RabbitMQ publish envelope used by
// all Fetcher AMQP publish paths. It preserves caller headers, injects request
// and trace metadata, and signs the payload when a signer is configured.
//
// lib-commons currently exposes webhook HMAC helpers, but those bind HTTP
// webhook wire fields (X-Webhook-Signature plus timestamp/body). Fetcher's AMQP
// envelope must bind tenant ID, job ID, exchange, and routing key so cross-tenant
// or cross-route replay fails. Until lib-commons grows a queue-envelope signing
// primitive, this file is the canonical Fetcher-specific envelope; keep signing
// changes here and covered by security_envelope_test.go.
func BuildSecurePublishing(ctx context.Context, reqID, exchange, routingKey string, body []byte, headers map[string]any, signer crypto.Signer, enableSigning bool) amqp.Publishing {
	publishHeaders := amqp.Table{}
	maps.Copy(publishHeaders, headers)

	// Canonical/security-critical headers are owned by the envelope builder.
	// Caller-provided collisions are deliberately overwritten before signing.
	publishHeaders[libConstants.HeaderID] = reqID
	publishHeaders["x-retry-count"] = 0

	if tenantID := tmcore.GetTenantIDContext(ctx); tenantID != "" {
		publishHeaders[HeaderTenantID] = tenantID
	}

	libOpentelemetry.InjectTraceHeadersIntoQueue(ctx, (*map[string]any)(&publishHeaders))

	if !isNilSigner(signer) && enableSigning {
		timestamp := time.Now().UTC().Unix()
		version := signer.SignatureVersion()
		jobID := extractJobID(body)
		tenantID, _ := publishHeaders[HeaderTenantID].(string)
		payload := BuildMessageSignaturePayload(timestamp, version, tenantID, jobID, exchange, routingKey, body)

		publishHeaders[HeaderMessageSignature] = signer.Sign(payload)
		publishHeaders[HeaderSignatureTimestamp] = strconv.FormatInt(timestamp, 10)
		publishHeaders[HeaderSignatureVersion] = version
	}

	return amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers:      publishHeaders,
		Body:         body,
	}
}

func isNilSigner(signer crypto.Signer) bool {
	if signer == nil {
		return true
	}

	value := reflect.ValueOf(signer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// BuildMessageSignaturePayload constructs the canonical payload bound by the
// RabbitMQ HMAC signature. It binds the message body plus routing and tenant
// metadata so replaying the same body under another tenant/exchange/key fails.
func BuildMessageSignaturePayload(timestamp int64, version, tenantID, jobID, exchange, routingKey string, body []byte) []byte {
	payload := make([]byte, 0, len(body)+len(version)+len(tenantID)+len(jobID)+len(exchange)+len(routingKey)+64)
	payload = fmt.Appendf(payload, "%d\n%s\n%s\n%s\n%s\n%s\n", timestamp, version, tenantID, jobID, exchange, routingKey)
	payload = append(payload, body...)

	return payload
}

func extractJobID(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	if jobID, ok := payload["jobId"].(string); ok {
		return jobID
	}

	return ""
}
