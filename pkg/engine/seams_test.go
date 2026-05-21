package engine

import (
	"strings"
	"testing"
)

func TestRequiredSeams_Characterization_IsComplete(t *testing.T) {
	t.Parallel()

	required := []Seam{
		{
			Name:       "datasource_factory_side_effects",
			SourcePath: "pkg/datasource/datasource_factory.go",
			Reason:     "datasource construction currently decrypts credentials, validates SSL options, tests live connections, and creates adapter repositories as side effects",
		},
		{
			Name:       "manager_create_fetcher_job_orchestration",
			SourcePath: "components/manager/internal/services/command/create_fetcher_job.go",
			Reason:     "manager job creation owns deduplication, connection resolution, ownership validation, connection testing, persistence, and queue publishing orchestration",
		},
		{
			Name:       "manager_validate_schema_orchestration",
			SourcePath: "components/manager/internal/services/query/validate_schema.go",
			Reason:     "schema validation resolves configured datasources, fetches schema metadata, applies datasource-specific defaults, and preserves plugin_crm table compatibility",
		},
		{
			Name:       "worker_extract_external_data_orchestration",
			SourcePath: "components/worker/internal/services/extract_data.go",
			Reason:     "worker extraction owns message parsing, idempotent job state transitions, connection resolution, datasource querying, storage persistence, and completion flow",
		},
		{
			Name:       "connection_resolver_behavior",
			SourcePath: "pkg/resolver/resolver.go",
			Reason:     "connection resolution abstracts internal tenant-managed datasources and external configured connections behind a stable ResolveConnections seam",
		},
		{
			Name:       "storage_result_handling",
			SourcePath: "components/worker/internal/services/extract_data.go",
			Reason:     "extracted results are marshaled, encrypted, written to configured object storage, and converted into result path and HMAC metadata for job updates",
		},
		{
			Name:       "worker_plugin_crm_compatibility",
			SourcePath: "components/worker/internal/services/extract_crm_data.go",
			Reason:     "plugin_crm extraction depends on physical collection prefix matching, filter transformation, field decryption, and merged logical collection output",
		},
		{
			Name:       "plugin_crm_adapter_compatibility",
			SourcePath: "components/manager/internal/services/query/validate_schema.go; components/worker/internal/services/extract_crm_data.go",
			Reason:     "plugin_crm remains Manager/Worker compatibility behavior for the first Engine release and must not become a generic Engine datasource extension",
		},
		{
			Name:       "notification_publishing",
			SourcePath: "components/worker/internal/services/job_notification.go",
			Reason:     "job status notifications are serialized with source metadata, routed by status and product, and published to the RabbitMQ job events exchange",
		},
	}

	actual := RequiredSeams()
	seamsByName := make(map[string]Seam, len(actual))
	for _, seam := range actual {
		if seam.Name == "" {
			t.Fatalf("runtime seam characterization contains an empty seam name: %#v", seam)
		}
		if seam.SourcePath == "" {
			t.Fatalf("runtime seam characterization %q has an empty source path", seam.Name)
		}
		if seam.Reason == "" {
			t.Fatalf("runtime seam characterization %q has an empty reason", seam.Name)
		}

		seamsByName[seam.Name] = seam
	}

	for _, tt := range required {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			got, ok := seamsByName[tt.Name]
			if !ok {
				t.Fatalf("missing runtime seam characterization %q", tt.Name)
			}
			if got.SourcePath != tt.SourcePath {
				t.Fatalf("runtime seam characterization %q source path = %q, want %q", tt.Name, got.SourcePath, tt.SourcePath)
			}
			if got.Reason == "" {
				t.Fatalf("runtime seam characterization %q reason is empty", tt.Name)
			}
			if got.Reason != tt.Reason {
				t.Fatalf("runtime seam characterization %q reason = %q, want %q", tt.Name, got.Reason, tt.Reason)
			}
		})
	}
}

func TestRequiredSeams_PluginCRMAdapterCompatibility_IsAdapterScoped(t *testing.T) {
	t.Parallel()

	const seamName = "plugin_crm_adapter_compatibility"

	actual := RequiredSeams()
	seamsByName := make(map[string]Seam, len(actual))
	for _, seam := range actual {
		seamsByName[seam.Name] = seam
	}

	seam, ok := seamsByName[seamName]
	if !ok {
		t.Fatalf("missing runtime seam characterization %q", seamName)
	}

	anchors := []struct {
		name string
		path string
	}{
		{
			name: "manager validation anchor",
			path: "components/manager/internal/services/query/validate_schema.go",
		},
		{
			name: "worker extraction anchor",
			path: "components/worker/internal/services/extract_crm_data.go",
		},
	}

	for _, tt := range anchors {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(seam.SourcePath, tt.path) {
				t.Fatalf("runtime seam characterization %q source path = %q, want anchor %q", seamName, seam.SourcePath, tt.path)
			}
		})
	}

	if !strings.Contains(seam.Reason, "Manager/Worker compatibility behavior") {
		t.Fatalf("runtime seam characterization %q reason = %q, want Manager/Worker compatibility behavior", seamName, seam.Reason)
	}

	for _, genericName := range []string{"plugin_crm_extension", "crm_extension", "datasource_crm_extension"} {
		genericName := genericName
		t.Run("no generic core seam "+genericName, func(t *testing.T) {
			t.Parallel()

			if _, exists := seamsByName[genericName]; exists {
				t.Fatalf("generic Engine core CRM seam %q must not exist; plugin_crm remains adapter compatibility behavior", genericName)
			}
		})
	}
}
