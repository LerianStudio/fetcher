package rabbitmq

import (
	"context"
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
			headers:    map[string]any{libConstants.HeaderID: "evil", HeaderSignatureVersion: "evil"},
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

func TestVerifyMessageSignature_RejectsFutureTimestamp(t *testing.T) {
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

	adapter := &RabbitMQAdapter{options: AdapterOptions{Signer: signer, SignatureTimestampTolerance: time.Minute}}
	err = adapter.verifyMessageSignature(body, headers, "exchange", "route", libLogNop(), nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureFromFuture)
}

func libLogNop() libLog.Logger { return libLog.NewNop() }
