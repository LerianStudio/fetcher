package rabbitmq

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	libConstants "github.com/LerianStudio/lib-commons/v5/commons/constants"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	libLog "github.com/LerianStudio/lib-observability/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBuildSecurePublishing_Edges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		ctx           context.Context
		headers       map[string]any
		enableSigning bool
		signer        func(*gomock.Controller) crypto.Signer
		wantSigned    bool
		wantTenant    string
	}{
		{
			name: "nil headers signing disabled",
			ctx:  context.Background(),
		},
		{
			name:    "caller headers preserved",
			ctx:     context.Background(),
			headers: map[string]any{"caller": "value"},
		},
		{
			name:       "canonical header collision overwritten",
			ctx:        context.Background(),
			headers:    map[string]any{libConstants.HeaderID: "evil", "x-retry-count": 99, HeaderTenantID: "evil", HeaderMessageSignature: "evil", HeaderSignatureTimestamp: "123", HeaderSignatureVersion: "evil"},
			wantSigned: false,
		},
		{
			name:          "signing enabled with signer",
			ctx:           context.Background(),
			enableSigning: true,
			signer: func(ctrl *gomock.Controller) crypto.Signer {
				mockSigner := crypto.NewMockSigner(ctrl)
				mockSigner.EXPECT().SignatureVersion().Return("v1")
				mockSigner.EXPECT().Sign(gomock.Any()).DoAndReturn(func(payload []byte) string {
					assert.Contains(t, string(payload), "v1\n")
					assert.Contains(t, string(payload), "exchange\nroute\n")
					assert.Contains(t, string(payload), `{"jobId":"job-123"}`)

					return "signed"
				})
				return mockSigner
			},
			wantSigned: true,
		},
		{
			name:          "signing enabled nil signer skips signature",
			ctx:           context.Background(),
			enableSigning: true,
		},
		{
			name:          "signing enabled typed nil signer skips signature",
			ctx:           context.Background(),
			enableSigning: true,
			signer: func(*gomock.Controller) crypto.Signer {
				var signer *crypto.HMACSigner
				return signer
			},
		},
		{
			name:          "tenant context binds tenant header",
			ctx:           tmcore.ContextWithTenantID(context.Background(), "tenant-123"),
			enableSigning: true,
			signer: func(ctrl *gomock.Controller) crypto.Signer {
				mockSigner := crypto.NewMockSigner(ctrl)
				mockSigner.EXPECT().SignatureVersion().Return("v1")
				mockSigner.EXPECT().Sign(gomock.Any()).DoAndReturn(func(payload []byte) string {
					assert.Contains(t, string(payload), "tenant-123")
					return "tenant-signed"
				})
				return mockSigner
			},
			wantSigned: true,
			wantTenant: "tenant-123",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			var signer crypto.Signer
			if tt.signer != nil {
				signer = tt.signer(ctrl)
			}

			msg := BuildSecurePublishing(tt.ctx, "req-1", "exchange", "route", []byte(`{"jobId":"job-123"}`), tt.headers, signer, tt.enableSigning)

			assert.Equal(t, "req-1", msg.Headers[libConstants.HeaderID])
			assert.Equal(t, int64(0), int64(msg.Headers["x-retry-count"].(int)))
			assert.NotEqual(t, "evil", msg.Headers[HeaderTenantID])
			assert.NotEqual(t, "evil", msg.Headers[HeaderMessageSignature])
			assert.NotEqual(t, "123", msg.Headers[HeaderSignatureTimestamp])
			assert.NotEqual(t, "evil", msg.Headers[HeaderSignatureVersion])
			if tt.headers != nil && tt.headers["caller"] != nil {
				assert.Equal(t, "value", msg.Headers["caller"])
			}
			if tt.wantTenant != "" {
				assert.Equal(t, tt.wantTenant, msg.Headers[HeaderTenantID])
			}
			if tt.wantSigned {
				assert.NotEmpty(t, msg.Headers[HeaderMessageSignature])
				assert.NotEmpty(t, msg.Headers[HeaderSignatureTimestamp])
				assert.Equal(t, "v1", msg.Headers[HeaderSignatureVersion])
			} else {
				assert.Empty(t, msg.Headers[HeaderMessageSignature])
			}
		})
	}
}

func TestVerifyMessageSignature_TamperAndLegacyCompatibility(t *testing.T) {
	t.Parallel()

	signer, err := crypto.NewHMACSigner([]byte("0123456789abcdef0123456789abcdef"), crypto.SignatureVersion)
	require.NoError(t, err)

	timestamp := time.Now().Add(-time.Second).Unix()
	body := []byte(`{"jobId":"job-123"}`)
	signature := signer.Sign(BuildMessageSignaturePayload(timestamp, signer.SignatureVersion(), "tenant-123", "job-123", "exchange", "route", body))
	headers := map[string]any{
		HeaderTenantID:           "tenant-123",
		HeaderMessageSignature:   signature,
		HeaderSignatureTimestamp: strconv.FormatInt(timestamp, 10),
		HeaderSignatureVersion:   signer.SignatureVersion(),
	}

	require.NoError(t, VerifyMessageSignature(body, headers, "exchange", "route", signer, time.Minute, libLogNop(), nil))

	tamperedHeaders := map[string]any{
		HeaderTenantID:           "tenant-other",
		HeaderMessageSignature:   signature,
		HeaderSignatureTimestamp: strconv.FormatInt(timestamp, 10),
		HeaderSignatureVersion:   signer.SignatureVersion(),
	}
	err = VerifyMessageSignature(body, tamperedHeaders, "exchange", "route", signer, time.Minute, libLogNop(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureVerificationFailed)

	legacySignature := signer.Sign(crypto.BuildSignaturePayload(timestamp, body))
	legacyHeaders := map[string]any{
		HeaderMessageSignature:   legacySignature,
		HeaderSignatureTimestamp: strconv.FormatInt(timestamp, 10),
		HeaderSignatureVersion:   signer.SignatureVersion(),
	}
	err = VerifyMessageSignature(body, legacyHeaders, "exchange", "route", signer, time.Minute, libLogNop(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureVerificationFailed)
	require.NoError(t, VerifyMessageSignature(body, legacyHeaders, "exchange", "route", signer, time.Minute, libLogNop(), nil, true))

	legacyTenantHeaders := map[string]any{
		HeaderTenantID:           "tenant-123",
		HeaderMessageSignature:   legacySignature,
		HeaderSignatureTimestamp: strconv.FormatInt(timestamp, 10),
		HeaderSignatureVersion:   signer.SignatureVersion(),
	}
	require.NoError(t, VerifyMessageSignature(body, legacyTenantHeaders, "exchange", "route", signer, time.Minute, libLogNop(), nil, true))
}

func TestVerifyMessageSignature_RejectsRouteAndBodyTamper(t *testing.T) {
	t.Parallel()

	signer, err := crypto.NewHMACSigner([]byte("0123456789abcdef0123456789abcdef"), crypto.SignatureVersion)
	require.NoError(t, err)

	timestamp := time.Now().Add(-time.Second).Unix()
	body := []byte(`{"jobId":"job-123"}`)
	signature := signer.Sign(BuildMessageSignaturePayload(timestamp, signer.SignatureVersion(), "tenant-123", "job-123", "exchange", "route", body))
	headers := map[string]any{
		HeaderTenantID:           "tenant-123",
		HeaderMessageSignature:   signature,
		HeaderSignatureTimestamp: strconv.FormatInt(timestamp, 10),
		HeaderSignatureVersion:   signer.SignatureVersion(),
	}

	for _, tt := range []struct {
		name       string
		body       []byte
		exchange   string
		routingKey string
	}{
		{name: "exchange tamper", body: body, exchange: "other-exchange", routingKey: "route"},
		{name: "routing key tamper", body: body, exchange: "exchange", routingKey: "other-route"},
		{name: "body tamper", body: []byte(`{"jobId":"job-456"}`), exchange: "exchange", routingKey: "route"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := VerifyMessageSignature(tt.body, headers, tt.exchange, tt.routingKey, signer, time.Minute, libLogNop(), nil)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrSignatureVerificationFailed)
		})
	}
}

func TestVerifyMessageSignature_TypedNilSigner(t *testing.T) {
	t.Parallel()

	var signer *crypto.HMACSigner
	err := VerifyMessageSignature([]byte(`{}`), map[string]any{}, "exchange", "route", signer, time.Minute, libLogNop(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureVerifierNotConfigured)
}

func TestCompatibilityWaivers_DocumentMultiTenantConsumerAndAMQPSigningGaps(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	waiverPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "compatibility-waivers.md")
	content, err := os.ReadFile(waiverPath)
	require.NoError(t, err)

	waivers := string(content)
	require.Contains(t, waivers, "Resolved: Worker no longer depends on `tmconsumer.MultiTenantConsumer` hidden Tenant Manager client")
	require.Contains(t, waivers, "RabbitMQ AMQP security envelope uses temporary local HMAC adapter")
	require.Contains(t, waivers, "client.WithCircuitBreaker")
	require.Contains(t, waivers, "No longer blocks runtime")
	require.Contains(t, waivers, "queue-envelope signing/verification APIs")
}

func TestVerifyMessageSignature_AllowsBoundedFutureSkew(t *testing.T) {
	t.Parallel()

	signer, err := crypto.NewHMACSigner([]byte("0123456789abcdef0123456789abcdef"), crypto.SignatureVersion)
	require.NoError(t, err)

	future := time.Now().Add(10 * time.Second).Unix()
	body := []byte(`{"jobId":"job-123"}`)
	payload := BuildMessageSignaturePayload(future, signer.SignatureVersion(), "tenant-123", "job-123", "exchange", "route", body)
	headers := map[string]any{
		HeaderTenantID:           "tenant-123",
		HeaderMessageSignature:   signer.Sign(payload),
		HeaderSignatureTimestamp: strconv.FormatInt(future, 10),
		HeaderSignatureVersion:   signer.SignatureVersion(),
	}

	require.NoError(t, VerifyMessageSignature(body, headers, "exchange", "route", signer, time.Minute, libLogNop(), nil))
}

func TestVerifyMessageSignature_RejectsFutureTimestampBeyondSkew(t *testing.T) {
	t.Parallel()

	signer, err := crypto.NewHMACSigner([]byte("0123456789abcdef0123456789abcdef"), crypto.SignatureVersion)
	require.NoError(t, err)

	future := time.Now().Add(10 * time.Minute).Unix()
	body := []byte(`{"jobId":"job-123"}`)
	payload := BuildMessageSignaturePayload(future, signer.SignatureVersion(), "tenant-123", "job-123", "exchange", "route", body)
	headers := map[string]any{
		HeaderTenantID:           "tenant-123",
		HeaderMessageSignature:   signer.Sign(payload),
		HeaderSignatureTimestamp: strconv.FormatInt(future, 10),
		HeaderSignatureVersion:   signer.SignatureVersion(),
	}

	err = VerifyMessageSignature(body, headers, "exchange", "route", signer, time.Minute, libLogNop(), nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureFromFuture)
}

func libLogNop() libLog.Logger { return libLog.NewNop() }
