package command

import (
	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/model"
)

// extraction_request_mapper bridges the Manager's public FetcherRequest onto the
// embedded Engine's request/tenant contracts for the job-creation path (T-009
// ST-01). It is a PURE host-side projection: it performs NO validation, NO schema
// discovery, and NO PlanExtraction call. All current Manager validation
// (IsValid, metadata.source, filter references, connection resolution, product
// ownership, live connection test / FET-0414) stays where it is and BYTE-IDENTICAL.
//
// Owner decision B2 (locked): the Engine carries ONLY tenantId + requestId. There
// is NO organization and NO product in any engine.* type. X-Product-Name and the
// organization id stay HOST concerns in the Manager, exactly as today. metadata.source
// (e.g. "plugin_crm") is preserved as OPAQUE engine.ExtractionRequest.Metadata that
// the Engine carries but never interprets — mirroring the T-006 planner.
//
// SCOPE NOTE (T-009 ST-01): only the request projection is wired here, because the
// engine.ExtractionRequest has a live consumer (job persistence + queue payload).
// The engine.TenantContext bridge (connectioncompat.TenantContextFromRequest) is
// deliberately NOT introduced in this subtask: nothing on the create path consumes
// a TenantContext without an Engine planning/execution call, so adding it now would
// be speculative orphan code. The tenant bridge lands with the execution/planning
// consumer (ST-03 / T-010), reusing the SAME connectioncompat bridge the T-008
// connection/schema services already use — preserving the dual-tenant-truth
// invariant at the point a consumer exists.
//
// The mapped engine.ExtractionRequest is a LIVE intermediate, not orphan code:
// model.NewJob's MappedFields/Metadata are sourced from it (see buildJobFromRequest
// in create_fetcher_job.go), so the persisted job AND the emitted queue payload are
// built from the engine-shaped request while remaining byte-identical to the legacy
// shape. T-010 (worker execution) will mirror/share this same request shape.

// mapToExtractionRequest projects a Manager FetcherRequest onto an
// engine.ExtractionRequest. MappedFields convert to engine.FieldSelection (the
// same underlying type), filters are carried per datasource as the Engine's
// opaque map[string]any contract, and metadata (including metadata.source) is
// carried opaque. No limit overrides are introduced: this subtask maps, it does
// not bound.
func mapToExtractionRequest(request model.FetcherRequest) engine.ExtractionRequest {
	return engine.ExtractionRequest{
		MappedFields: mapMappedFields(request.DataRequest.MappedFields),
		Filters:      mapFilters(request.DataRequest.Filters),
		Metadata:     request.Metadata,
	}
}

// mapMappedFields converts the Manager's map[string]map[string][]string selection
// into the Engine's map[string]engine.FieldSelection. FieldSelection is defined as
// map[string][]string, so the per-datasource value is reinterpreted without copying
// the leaf slices — the projection is identity over the wire.
func mapMappedFields(mappedFields map[string]map[string][]string) map[string]engine.FieldSelection {
	if mappedFields == nil {
		return nil
	}

	out := make(map[string]engine.FieldSelection, len(mappedFields))
	for datasource, tables := range mappedFields {
		out[datasource] = engine.FieldSelection(tables)
	}

	return out
}

// mapFilters carries the Manager's typed NestedFilters into the Engine's opaque
// per-datasource filter map. The Engine contract is map[string]any keyed by
// datasource; the Engine never interprets the value. The queue payload does NOT
// flow through this representation — it emits the typed filters directly (see
// buildQueueMessage) to keep the published bytes byte-identical — so this mapping
// exists for the engine request contract, not as the wire source for filters.
func mapFilters(filters model.NestedFilters) map[string]any {
	if len(filters) == 0 {
		return nil
	}

	out := make(map[string]any, len(filters))
	for datasource, tables := range filters {
		out[datasource] = tables
	}

	return out
}

// mappedFieldsFromExtraction projects the Engine's map[string]engine.FieldSelection
// back into the Manager's map[string]map[string][]string for job persistence. Since
// engine.FieldSelection IS map[string][]string, the conversion is identity over the
// wire (the leaf slices are shared, not copied), so the persisted job's mapped
// fields are byte-identical to sourcing them directly from the request. Routing
// them through the engine.ExtractionRequest is what makes the intermediate a live,
// non-orphan consumer.
func mappedFieldsFromExtraction(request engine.ExtractionRequest) map[string]map[string][]string {
	if request.MappedFields == nil {
		return nil
	}

	out := make(map[string]map[string][]string, len(request.MappedFields))
	for datasource, tables := range request.MappedFields {
		out[datasource] = tables
	}

	return out
}
