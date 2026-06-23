package resolver

import (
	"sort"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalDatasourceRegistry_IsInternal(t *testing.T) {
	registry := NewInternalDatasourceRegistry()

	tests := []struct {
		name       string
		configName string
		expected   bool
	}{
		{"midaz_onboarding is internal", "midaz_onboarding", true},
		{"midaz_transaction is internal", "midaz_transaction", true},
		{"plugin_crm is internal", "plugin_crm", true},
		{"random-external is not internal", "random-external", false},
		{"empty string is not internal", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, registry.IsInternal(tt.configName))
		})
	}
}

func TestInternalDatasourceRegistry_GetConfig(t *testing.T) {
	registry := NewInternalDatasourceRegistry()

	tests := []struct {
		name       string
		configName string
		wantOK     bool
		wantSvc    string
		wantModule string
		wantDBType model.DBType
	}{
		{"midaz_onboarding", "midaz_onboarding", true, "ledger", "onboarding", model.TypePostgreSQL},
		{"midaz_transaction", "midaz_transaction", true, "ledger", "transaction", model.TypePostgreSQL},
		{"plugin_crm", "plugin_crm", true, "ledger", "crm", model.TypeMongoDB},
		{"unknown returns false", "unknown", false, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, ok := registry.GetConfig(tt.configName)
			assert.Equal(t, tt.wantOK, ok)

			if ok {
				assert.Equal(t, tt.wantSvc, cfg.Service)
				assert.Equal(t, tt.wantModule, cfg.Module)
				assert.Equal(t, tt.wantDBType, cfg.DBType)
			}
		})
	}
}

func TestInternalDatasourceRegistry_ListInternal(t *testing.T) {
	registry := NewInternalDatasourceRegistry()

	names := registry.ListInternal()
	require.Len(t, names, 3)

	sort.Strings(names)
	assert.Equal(t, []string{"midaz_onboarding", "midaz_transaction", "plugin_crm"}, names)
}
