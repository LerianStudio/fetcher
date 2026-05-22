// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package resolver

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/datasource/sslmode"
	"github.com/LerianStudio/fetcher/pkg/model"

	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
)

// LoadInternalConnectionsFromEnv scans DATASOURCE_{NAME}_* env vars and builds
// Connection objects for each internal datasource found in the registry.
// This allows internal datasources (e.g. plugin_crm, midaz_onboarding) to be
// configured via environment variables at deploy time.
func LoadInternalConnectionsFromEnv(registry *InternalDatasourceRegistry, logger libLog.Logger) map[string]*model.Connection {
	envConnections := make(map[string]*model.Connection)

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "DATASOURCE_") || !strings.Contains(env, "_CONFIG_NAME=") {
			continue
		}

		configName := strings.SplitN(env, "=", 2)[1]
		if configName == "" || !registry.IsInternal(configName) {
			continue
		}

		// Extract the NAME segment: DATASOURCE_{NAME}_CONFIG_NAME → NAME
		prefix := strings.TrimSuffix(strings.SplitN(env, "=", 2)[0], "_CONFIG_NAME")

		getEnv := func(field string) string {
			return os.Getenv(prefix + "_" + field)
		}

		portStr := getEnv("PORT")

		port, portErr := strconv.Atoi(portStr)
		if portErr != nil || port == 0 {
			logger.Log(context.Background(), libLog.LevelWarn, "Invalid or missing PORT for internal datasource, connection may fail",
				libLog.String("config_name", configName),
				libLog.String("port_value", portStr),
			)
		}

		dbType, typeErr := model.NewTypeFromString(getEnv("TYPE"))
		if typeErr != nil {
			logger.Log(context.Background(), libLog.LevelWarn, "Invalid TYPE for internal datasource, skipping",
				libLog.String("config_name", configName),
				libLog.String("type_value", getEnv("TYPE")),
			)

			continue
		}

		host := getEnv("HOST")
		database := getEnv("DATABASE")

		if host == "" || database == "" {
			logger.Log(context.Background(), libLog.LevelWarn, "Missing HOST or DATABASE for internal datasource, skipping",
				libLog.String("config_name", configName),
				libLog.String("host", host),
				libLog.String("database", database),
			)

			continue
		}

		conn := &model.Connection{
			ConfigName:   configName,
			Type:         dbType,
			Host:         host,
			Port:         port,
			DatabaseName: database,
			Username:     getEnv("USER"),
		}

		// Parse SSL env vars (SSLMODE is the gate; CA/Cert/Key are optional).
		// Behavior parity with TYPE invalid-skip (lines ~53-60): on invalid SSL
		// mode we log WARN and skip the connection entirely so misconfiguration
		// surfaces loudly instead of silently downgrading to plaintext.
		ssl, sslErr := parseSSLFromEnv(getEnv, dbType)
		if sslErr != nil {
			logger.Log(context.Background(), libLog.LevelWarn, "Invalid SSLMODE for internal datasource, skipping",
				libLog.String("config_name", configName),
				libLog.String("type", string(dbType)),
				libLog.String("sslmode_value", getEnv("SSLMODE")),
				libLog.String("error", sslErr.Error()),
			)

			continue
		}

		conn.SSL = ssl

		// Parse OPTIONS env var (query-string format: authSource=admin&directConnection=true)
		// into conn.Metadata so buildMongoDBOptions can use them.
		if opts := getEnv("OPTIONS"); opts != "" {
			metadata := make(map[string]any)

			for _, pair := range strings.Split(opts, "&") {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					metadata[kv[0]] = kv[1]
				}
			}

			conn.Metadata = &metadata
		}

		// Internal connections use plaintext password (no encryption needed, in-memory only).
		conn.SetPlaintextPassword(getEnv("PASSWORD"))

		if existing, dup := envConnections[configName]; dup {
			logger.Log(context.Background(), libLog.LevelWarn, "Duplicate configName in DATASOURCE_* env vars, keeping first entry",
				libLog.String("config_name", configName),
				libLog.String("kept_host", existing.Host),
				libLog.String("skipped_host", conn.Host),
			)

			continue
		}

		envConnections[configName] = conn

		logger.Log(context.Background(), libLog.LevelInfo, "Loaded internal datasource from env vars",
			libLog.String("config_name", configName),
			libLog.String("type", string(dbType)),
			libLog.String("host", conn.Host),
		)
	}

	return envConnections
}

// parseSSLFromEnv reads SSLMODE/SSL_CA/SSL_CERT/SSL_KEY via getEnv and returns
// the populated SSLConfig. SSLMODE is the gate: when unset, returns (nil, nil)
// so the connection keeps SSL nil and the driver default applies. When SSLMODE
// is set but invalid for dbType, returns the validation error so callers can
// WARN and skip the connection (parity with TYPE invalid-skip).
func parseSSLFromEnv(getEnv func(string) string, dbType model.DBType) (*model.SSLConfig, error) {
	sslModeRaw := getEnv("SSLMODE")
	if sslModeRaw == "" {
		return nil, nil
	}

	if err := validateSSLModeForType(dbType, sslModeRaw); err != nil {
		return nil, err
	}

	return &model.SSLConfig{
		Mode: sslModeRaw,
		CA:   getEnv("SSL_CA"),
		Cert: getEnv("SSL_CERT"),
		Key:  getEnv("SSL_KEY"),
	}, nil
}

// validateSSLModeForType validates that the SSL mode is valid for the given
// database type by dispatching to the per-driver allowlist functions in
// pkg/datasource/sslmode. Returns nil for unknown types (no validation).
func validateSSLModeForType(dbType model.DBType, mode string) error {
	switch dbType {
	case model.TypePostgreSQL:
		return sslmode.ValidatePostgreSQLMode(mode)
	case model.TypeMySQL:
		return sslmode.ValidateMySQLMode(mode)
	case model.TypeOracle:
		return sslmode.ValidateOracleMode(mode)
	case model.TypeMongoDB:
		return sslmode.ValidateMongoDBMode(mode)
	case model.TypeSQLServer:
		return sslmode.ValidateSQLServerMode(mode)
	default:
		return nil
	}
}
