package query

import (
	"context"
	"golang-plugin-boilerplate/pkg/constant"
	"golang-plugin-boilerplate/pkg/model"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	example "golang-plugin-boilerplate/internal/adapters/postgres/example"
)

func TestCreateExample(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExampleRepo := example.NewMockRepository(ctrl)

	exampleCase := &ExampleCommand{
		ExampleRepo: mockExampleRepo,
	}

	createExampleInput := &model.CreateExampleInput{
		Name: "test",
		Age:  12,
	}

	tests := []struct {
		name           string
		exampleInput   *model.CreateExampleInput
		mockSetup      func()
		expectErr      bool
		expectedResult *model.ExampleOutput
	}{
		{
			name:         "Success - Create example",
			exampleInput: createExampleInput,
			mockSetup: func() {
				validUUID := uuid.New()
				mockExampleRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(&model.ExampleOutput{
						ID: validUUID.String(), Name: "test", Age: 12,
					}, nil)
			},
			expectErr: false,
			expectedResult: &model.ExampleOutput{
				ID: "valid-uuid", Name: "test", Age: 12,
			},
		},
		{
			name:         "Error - Create an example",
			exampleInput: createExampleInput,
			mockSetup: func() {
				mockExampleRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, constant.ErrBadRequest)
			},
			expectErr:      true,
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			ctx := context.Background()
			result, err := exampleCase.CreateExample(ctx, tt.exampleInput)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}
