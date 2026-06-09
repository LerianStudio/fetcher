package query

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/LerianStudio/fetcher/v2/pkg/mongodb/job"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/mock/gomock"
)

func TestGetJob_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jobID := uuid.New()
	now := time.Now().UTC()

	expectedJob := &model.Job{
		ID:     jobID,
		Status: model.JobStatusPending,
		MappedFields: map[string]map[string][]string{
			"datasource1": {
				"table1": {"field1", "field2"},
			},
		},
		Filters:     model.NestedFilters{},
		RequestHash: "abc123",
		CreatedAt:   now,
	}

	tests := []struct {
		name          string
		setupMock     func(mockRepo *job.MockRepository)
		jobID         uuid.UUID
		expectedJob   *model.Job
		expectedError error
	}{
		{
			name: "success - job found",
			setupMock: func(mockRepo *job.MockRepository) {
				mockRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(expectedJob, nil)
			},

			jobID:         jobID,
			expectedJob:   expectedJob,
			expectedError: nil,
		},
		{
			name: "error - repository returns mongo.ErrNoDocuments",
			setupMock: func(mockRepo *job.MockRepository) {
				mockRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(nil, mongo.ErrNoDocuments)
			},

			jobID:       jobID,
			expectedJob: nil,
			// Service passes through repository errors directly
			expectedError: mongo.ErrNoDocuments,
		},
		{
			name: "error - job not found (nil return)",
			setupMock: func(mockRepo *job.MockRepository) {
				mockRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(nil, nil)
			},

			jobID:         jobID,
			expectedJob:   nil,
			expectedError: pkg.ResponseErrorWithStatusCode{StatusCode: http.StatusNotFound},
		},
		{
			name: "error - repository error",
			setupMock: func(mockRepo *job.MockRepository) {
				mockRepo.EXPECT().
					FindByID(gomock.Any(), jobID).
					Return(nil, errors.New("database error"))
			},

			jobID:         jobID,
			expectedJob:   nil,
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := job.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			service := NewGetJob(mockRepo)

			ctx := testContext()

			result, err := service.Execute(ctx, tt.jobID)

			if tt.expectedError != nil {
				require.Error(t, err)
				if respErr, ok := tt.expectedError.(pkg.ResponseErrorWithStatusCode); ok {
					var actualRespErr pkg.ResponseErrorWithStatusCode
					if assert.True(t, errors.As(err, &actualRespErr)) {
						assert.Equal(t, respErr.StatusCode, actualRespErr.StatusCode)
					}
				} else if errors.Is(tt.expectedError, mongo.ErrNoDocuments) {
					// For mongo.ErrNoDocuments, check direct match
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					// For other generic errors, just check error occurred
					assert.Error(t, err)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedJob.ID, result.ID)
				assert.Equal(t, tt.expectedJob.Status, result.Status)
			}
		})
	}
}
