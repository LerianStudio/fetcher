package command

import (
	"github.com/LerianStudio/fetcher/v2/pkg/engine"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
)

// extraction_request_mapper bridges the Manager's public FetcherRequest onto the
// embedded Engine's request contract for the job-creation path (T-009 ST-01,
// Option 2). It is a PURE host-side projection: NO validation, NO schema discovery,
// NO PlanExtraction. All Manager validation stays where it is and byte-identical.
//
// Owner decision B2 (locked): the Engine carries ONLY tenantId + requestId — NO
// organization, NO product. X-Product-Name / organization stay HOST concerns in
// the Manager. metadata.source (e.g. "plugin_crm") is preserved as OPAQUE
// engine.ExtractionRequest.Metadata that the Engine carries but never interprets.
//
// SCOPE NOTE: this subtask wires only the parts of the engine request that have a
// LIVE consumer on the create path — MappedFields and Metadata, from which the
// persisted job and the queue payload are built. Two projections are deliberately
// DEFERRED to their execution consumer (T-010), to avoid speculative/orphan code:
//   - engine.TenantContext (via connectioncompat.TenantContextFromRequest): nothing
//     on the create path consumes a TenantContext without an Engine planning/exec
//     call; it lands reusing the same T-008 bridge, preserving dual-tenant-truth.
//   - engine.ExtractionRequest.Filters: filters stay on the typed
//     request.DataRequest.Filters path (persisted + published unchanged); the engine
//     filter projection lands with the execution consumer, which pins it against the
//     planner's nested filter shape (engine.PlanStep.Filters) rather than guessing.

// mapToExtractionRequest projects a Manager FetcherRequest onto an
// engine.ExtractionRequest (MappedFields + opaque Metadata). Filters and Overrides
// are intentionally left nil — see the deferral note above.
func mapToExtractionRequest(request model.FetcherRequest) engine.ExtractionRequest {
	return engine.ExtractionRequest{
		MappedFields: mapMappedFields(request.DataRequest.MappedFields),
		Metadata:     request.Metadata,
	}
}

// mapMappedFields converts the Manager selection into map[string]engine.FieldSelection.
// FieldSelection is map[string][]string, so leaf slices are shared, not copied.
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

// mappedFieldsFromExtraction projects the engine selection back into the Manager's
// map[string]map[string][]string for job persistence (identity over the wire, leaf
// slices shared). Routing the job's mapped fields through the engine request is
// what makes the intermediate a live, non-orphan consumer.
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
