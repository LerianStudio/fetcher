package rabbitmq

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/rabbitmq"
	"github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewPublisherRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	logger := log.NewNop()
	telemetry := &opentelemetry.Telemetry{}

	pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
	assert.NotNil(t, pr)
	assert.Equal(t, mockAdapter, pr.adapter)
}

func TestPublisherRoutes_Publish(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	logger := log.NewNop()
	telemetry := &opentelemetry.Telemetry{}
	ctx := context.Background()

	t.Run("successful publish", func(t *testing.T) {
		mockAdapter.EXPECT().
			ProducerDefault(gomock.Any(), "test-exchange", "test-key", gomock.Any(), nil).
			Return(nil)

		pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
		err := pr.Publish(ctx, "test-exchange", "test-key", []byte("test body"))
		assert.NoError(t, err)
	})

	t.Run("publish error", func(t *testing.T) {
		mockAdapter.EXPECT().
			ProducerDefault(gomock.Any(), "test-exchange", "test-key", gomock.Any(), nil).
			Return(errors.New("publish error"))

		pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
		err := pr.Publish(ctx, "test-exchange", "test-key", []byte("test body"))
		assert.Error(t, err)
	})
}

func TestPublisherRoutes_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAdapter := rabbitmq.NewMockAdapter(ctrl)
	logger := log.NewNop()
	telemetry := &opentelemetry.Telemetry{}
	ctx := context.Background()

	t.Run("successful shutdown", func(t *testing.T) {
		mockAdapter.EXPECT().Shutdown(gomock.Any()).Return(nil)

		pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
		err := pr.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("shutdown error", func(t *testing.T) {
		mockAdapter.EXPECT().Shutdown(gomock.Any()).Return(errors.New("shutdown error"))

		pr := NewPublisherRoutesWithAdapter(mockAdapter, logger, telemetry)
		err := pr.Shutdown(ctx)
		assert.Error(t, err)
	})
}
