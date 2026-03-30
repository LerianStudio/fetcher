package datasource

import (
	"context"
	"testing"

	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func testConnectionV2(dbType model.DBType) *model.Connection {
	return &model.Connection{
		ID: uuid.New(),

		ConfigName:           "test-conn",
		Type:                 dbType,
		Host:                 "localhost",
		Port:                 5432,
		DatabaseName:         "testdb",
		Username:             "testuser",
		PasswordEncrypted:    "encrypted",
		EncryptionKeyVersion: "v1",
	}
}

// ============================================================================
// Nil parameter tests
// ============================================================================

func TestNewDataSourceFromConnection_NilConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	ds, err := NewDataSourceFromConnection(context.Background(), nil, mockCryptor, logger)
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection cannot be nil")
}

func TestNewDataSourceFromConnection_NilCryptor(t *testing.T) {
	conn := testConnectionV2(model.TypePostgreSQL)
	logger := &log.GoLogger{Level: log.LevelDebug}

	ds, err := NewDataSourceFromConnection(context.Background(), conn, nil, logger)
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cryptor cannot be nil for encrypted connections")
}

// ============================================================================
// Unsupported database type
// ============================================================================

func TestNewDataSourceFromConnection_UnsupportedType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.DBType("CASSANDRA"))

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database type: CASSANDRA")
}

// ============================================================================
// Base config construction
// ============================================================================

func TestNewDataSourceConfigFromConnection(t *testing.T) {
	conn := testConnectionV2(model.TypePostgreSQL)
	conn.Port = 5432

	config := newDataSourceConfigFromConnection(conn)

	assert.Equal(t, conn.ID.String(), config.ID)
	assert.Equal(t, "test-conn", config.ConfigName)
	assert.Equal(t, "POSTGRESQL", config.Type)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "5432", config.Port)
	assert.Equal(t, "testdb", config.DatabaseName)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "encrypted", config.PasswordEncrypted)
}

func TestNewDataSourceConfigFromConnection_WithSSL(t *testing.T) {
	conn := testConnectionV2(model.TypePostgreSQL)
	conn.SSL = &model.SSLConfig{
		Mode: "require",
	}

	config := newDataSourceConfigFromConnection(conn)

	assert.Equal(t, "require", config.SSL.Mode)
}

func TestNewDataSourceConfigFromConnection_NilSSL(t *testing.T) {
	conn := testConnectionV2(model.TypePostgreSQL)
	conn.SSL = nil

	config := newDataSourceConfigFromConnection(conn)

	assert.Equal(t, model.SSLConfig{}, config.SSL)
}

// ============================================================================
// Type dispatching (verifies that each DB type enters the correct path)
// Each type will fail at the connection step, but the error message reveals
// which code path was taken.
// ============================================================================

func TestNewDataSourceFromConnection_DispatchesMongoDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypeMongoDB)
	conn.Port = 27017

	// Decrypt returns a password so the code proceeds past decryption to the connection attempt
	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	// Will fail at MongoDB connection attempt
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MongoDB")
}

func TestNewDataSourceFromConnection_DispatchesPostgreSQL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypePostgreSQL)

	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	// Will fail at PostgreSQL connection attempt
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PostgreSQL")
}

func TestNewDataSourceFromConnection_DispatchesMySQL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypeMySQL)
	conn.Port = 3306

	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	// Will fail at MySQL connection attempt
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MySQL")
}

func TestNewDataSourceFromConnection_DispatchesOracle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypeOracle)
	conn.Port = 1521
	meta := map[string]any{"serviceName": "ORCL"}
	conn.Metadata = &meta

	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	// Will fail at Oracle connection attempt
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Oracle")
}

func TestNewDataSourceFromConnection_DispatchesSQLServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypeSQLServer)
	conn.Port = 1433

	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	// Will fail at SQL Server connection attempt
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SQL Server")
}

// ============================================================================
// Oracle requires serviceName in metadata
// ============================================================================

func TestNewDataSourceFromConnection_OracleMissingServiceName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypeOracle)
	conn.Port = 1521
	conn.Metadata = nil // No metadata

	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "serviceName is required")
}

// ============================================================================
// SSL mode validation (these fail before attempting connection)
// ============================================================================

func TestNewDataSourceFromConnection_PostgreSQLInvalidSSLMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypePostgreSQL)
	conn.SSL = &model.SSLConfig{
		Mode: "invalid-mode; DROP TABLE users;--",
	}

	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	assert.Nil(t, ds)
	require.Error(t, err)
}

func TestNewDataSourceFromConnection_MySQLInvalidSSLMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypeMySQL)
	conn.Port = 3306
	conn.SSL = &model.SSLConfig{
		Mode: "malicious-mode",
	}

	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	assert.Nil(t, ds)
	require.Error(t, err)
}

func TestNewDataSourceFromConnection_MongoDBInvalidSSLMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCryptor := crypto.NewMockCryptor(ctrl)
	logger := &log.GoLogger{Level: log.LevelDebug}

	conn := testConnectionV2(model.TypeMongoDB)
	conn.Port = 27017
	conn.SSL = &model.SSLConfig{
		Mode: "inject; evil",
	}

	mockCryptor.EXPECT().Decrypt(gomock.Any(), "encrypted", "v1").Return("password123", nil)

	ds, err := NewDataSourceFromConnection(context.Background(), conn, mockCryptor, logger)
	assert.Nil(t, ds)
	require.Error(t, err)
}

// ============================================================================
// Factory with logger
// ============================================================================

func TestNewDataSourceFromConnectionWithLogger(t *testing.T) {
	logger := &log.GoLogger{Level: log.LevelDebug}

	factory := NewDataSourceFromConnectionWithLogger(logger)
	require.NotNil(t, factory)

	// Verify it returns error for nil connection
	ds, err := factory(context.Background(), nil, nil)
	assert.Nil(t, ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection cannot be nil")
}

// ============================================================================
// DataSourceFactory type check
// ============================================================================

func TestDataSourceFactory_TypeSignature(t *testing.T) {
	logger := &log.GoLogger{Level: log.LevelDebug}

	// Verify NewDataSourceFromConnectionWithLogger returns a compatible function
	var factory DataSourceFactory = NewDataSourceFromConnectionWithLogger(logger)
	require.NotNil(t, factory)

	// Verify it can be called
	ds, err := factory(context.Background(), nil, nil)
	assert.Nil(t, ds)
	require.Error(t, err)
}

// ============================================================================
// Verify DataSource interface compliance
// ============================================================================

func TestDataSourceConfig_HasRequiredFields(t *testing.T) {
	// Verify that the base config struct has all expected fields
	config := datasource.DataSourceConfig{
		ID:                "test-id",
		ConfigName:        "test-config",
		Type:              "POSTGRESQL",
		Host:              "localhost",
		Port:              "5432",
		DatabaseName:      "testdb",
		Username:          "testuser",
		PasswordEncrypted: "encrypted",
	}

	assert.Equal(t, "test-id", config.ID)
	assert.Equal(t, "test-config", config.ConfigName)
	assert.Equal(t, "POSTGRESQL", config.Type)
}
