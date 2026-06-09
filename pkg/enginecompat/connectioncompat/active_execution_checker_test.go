// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package connectioncompat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/connectioncompat"
	jobRepo "github.com/LerianStudio/fetcher/v2/pkg/mongodb/job"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestJobActiveExecutionChecker_ForwardsToJobRepo(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), "pg-main").
		Return(true, nil)

	checker := connectioncompat.NewJobActiveExecutionChecker(mockJobRepo)
	require.NotNil(t, checker)

	tenant, err := engine.NewTenantContext("tenant-a")
	require.NoError(t, err)

	active, err := checker.HasActiveExecutions(context.Background(), tenant, "pg-main")
	require.NoError(t, err)
	assert.True(t, active)
}

func TestJobActiveExecutionChecker_PropagatesRepoError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wantErr := errors.New("job store down")
	mockJobRepo := jobRepo.NewMockRepository(ctrl)
	mockJobRepo.EXPECT().
		ExistsRunningByMappedFieldKey(gomock.Any(), "pg-main").
		Return(false, wantErr)

	checker := connectioncompat.NewJobActiveExecutionChecker(mockJobRepo)

	tenant, _ := engine.NewTenantContext("tenant-a")
	_, err := checker.HasActiveExecutions(context.Background(), tenant, "pg-main")
	assert.ErrorIs(t, err, wantErr, "underlying repo error must propagate for host error mapping")
}

func TestNewJobActiveExecutionChecker_NilRepoYieldsNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, connectioncompat.NewJobActiveExecutionChecker(nil))
}
