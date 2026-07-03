// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package resolver

import (
	"fmt"
	"os"
	"strings"

	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
)

// serviceAPIKeyEnvPrefix is the env-var prefix for per-service tenant-manager
// API keys. The token suffix after this prefix is the normalized service token
// (e.g. MULTI_TENANT_SERVICE_API_KEY_PLUGIN_CRM → token "PLUGIN_CRM"). The bare
// prefix without a trailing token is the default key and is handled elsewhere.
const serviceAPIKeyEnvPrefix = "MULTI_TENANT_SERVICE_API_KEY_"

// normalizeServiceToken maps a tenant-manager service name to its env-token
// form: hyphens become underscores and the result is upper-cased. For example
// "plugin-crm" → "PLUGIN_CRM" and "ledger" → "LEDGER". This is the canonical
// key used both for the env-var suffix and for the client-map keys, so the
// loader and the adapter agree on a single normalization contract.
func normalizeServiceToken(service string) string {
	return strings.ToUpper(strings.ReplaceAll(service, "-", "_"))
}

// LoadServiceAPIKeysFromEnv scans os.Environ() for MULTI_TENANT_SERVICE_API_KEY_*
// vars and returns a map keyed by the normalized service token (the env-var name
// with the prefix stripped, then run through normalizeServiceToken) to its value.
// The bare MULTI_TENANT_SERVICE_API_KEY (no trailing token) is deliberately
// excluded — it is the default key, handled elsewhere. Entries with empty values
// are skipped.
//
// Normalizing on the loader side keeps the stored key in the same canonical form
// the adapter looks up (normalizeServiceToken(service)), so a hyphenated or
// lower-case env suffix cannot silently miss and fall back to the default key
// (CWE-178). Two distinct env vars that normalize to the same token are a
// configuration error: rather than silently last-writer-wins (which would route
// one service's credential to another, CWE-863), this returns an error naming
// the colliding token so the misconfiguration surfaces at startup. On success
// the returned map is always non-nil; on collision it is nil.
func LoadServiceAPIKeysFromEnv() (map[string]string, error) {
	keys := make(map[string]string)

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, serviceAPIKeyEnvPrefix) {
			continue
		}

		name, value, ok := strings.Cut(env, "=")
		if !ok {
			continue
		}

		// Strip the prefix to get the raw suffix; an empty suffix means the var
		// was exactly the prefix with no token, which we never expect here
		// because the bare default key has no trailing underscore.
		suffix := strings.TrimPrefix(name, serviceAPIKeyEnvPrefix)
		if suffix == "" || value == "" {
			continue
		}

		token := normalizeServiceToken(suffix)
		if _, exists := keys[token]; exists {
			return nil, fmt.Errorf("conflicting MULTI_TENANT_SERVICE_API_KEY_* env vars normalize to the same service token %q", token)
		}

		keys[token] = value
	}

	return keys, nil
}

// BuildServiceClients builds one tenant-manager client per entry in keys (the
// map keys are normalized service tokens) plus a default client from defaultKey,
// using the injected newClient closure. The closure carries all bootstrap and
// circuit-breaker concerns, keeping this function unit-testable and free of
// wiring details.
//
// When defaultKey is empty, defaultClient is nil and no error is returned. If
// any newClient call fails, the error is returned wrapped with the token (or
// "default") that failed, and both returned client values are nil. A newClient
// that returns (nil, nil) is also an error: a nil client would be a latent
// nil-deref at request time, so it is rejected at build time naming the service.
func BuildServiceClients(
	keys map[string]string,
	defaultKey string,
	newClient func(apiKey string) (*tmclient.Client, error),
) (clients map[string]*tmclient.Client, defaultClient *tmclient.Client, err error) {
	clients = make(map[string]*tmclient.Client, len(keys))

	for token, apiKey := range keys {
		client, clientErr := newClient(apiKey)
		if clientErr != nil {
			return nil, nil, fmt.Errorf("build tenant-manager client for service %q: %w", token, clientErr)
		}

		if client == nil {
			return nil, nil, fmt.Errorf("nil tenant-manager client for service %q", token)
		}

		clients[token] = client
	}

	if defaultKey != "" {
		client, clientErr := newClient(defaultKey)
		if clientErr != nil {
			return nil, nil, fmt.Errorf("build default tenant-manager client: %w", clientErr)
		}

		if client == nil {
			return nil, nil, fmt.Errorf("nil default tenant-manager client")
		}

		defaultClient = client
	}

	return clients, defaultClient, nil
}
