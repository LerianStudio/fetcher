package rabbitmq

import (
	"context"
	"maps"
	"strconv"
	"time"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	libConstants "github.com/LerianStudio/lib-commons/v5/commons/constants"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
)

// BuildSecurePublishing creates the canonical RabbitMQ publish envelope used by
// all Fetcher AMQP publish paths. It preserves caller headers, injects request
// and trace metadata, and signs the payload when a signer is configured.
func BuildSecurePublishing(ctx context.Context, reqID string, body []byte, headers map[string]any, signer crypto.Signer, enableSigning bool) amqp.Publishing {
	publishHeaders := amqp.Table{
		libConstants.HeaderID: reqID,
		"x-retry-count":       0,
	}

	maps.Copy(publishHeaders, headers)
	libOpentelemetry.InjectTraceHeadersIntoQueue(ctx, (*map[string]any)(&publishHeaders))

	if signer != nil && enableSigning {
		timestamp := time.Now().UTC().Unix()
		payload := crypto.BuildSignaturePayload(timestamp, body)

		publishHeaders[HeaderMessageSignature] = signer.Sign(payload)
		publishHeaders[HeaderSignatureTimestamp] = strconv.FormatInt(timestamp, 10)
		publishHeaders[HeaderSignatureVersion] = signer.SignatureVersion()
	}

	return amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers:      publishHeaders,
		Body:         body,
	}
}
