// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package resolver

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/model"

	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
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

		port, _ := strconv.Atoi(getEnv("PORT"))

		dbType := model.DBType(strings.ToUpper(getEnv("TYPE")))

		conn := &model.Connection{
			ConfigName:   configName,
			Type:         dbType,
			Host:         getEnv("HOST"),
			Port:         port,
			DatabaseName: getEnv("DATABASE"),
			Username:     getEnv("USER"),
		}

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

		envConnections[configName] = conn

		logger.Log(context.Background(), libLog.LevelInfo, "Loaded internal datasource from env vars",
			libLog.String("config_name", configName),
			libLog.String("type", string(dbType)),
			libLog.String("host", conn.Host),
		)
	}

	return envConnections
}
