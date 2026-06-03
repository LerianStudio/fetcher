// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg/engine"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/connectioncompat"
	plugincrm "github.com/LerianStudio/fetcher/pkg/enginecompat/plugincrm"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/tablenorm"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
)

// extractInto fills result with the extracted data for the message. After the
// strangler completion (T-010) extraction flows ONLY through the embedded Engine
// runner (mandatory; see UseCase.Validate) for generic datasources, and through the
// explicit plugin_crm compatibility path for CRM.
//
// plugin_crm is the ONE datasource the generic Engine path does not cover: its
// extraction needs collection prefix fan-out, search-field filter hashing, and PII
// field decryption — product policy that the Engine core stays agnostic of. So the
// message is SPLIT by datasource: the plugin_crm portion (if any) extracts through
// the EXPLICIT Worker CRM compatibility path (queryPluginCRMDatabase -> QueryPluginCRM),
// and the remaining generic datasources extract through the Engine runner. Both merge
// into the same result. A job with no plugin_crm datasource never touches CRM code.
func (uc *UseCase) extractInto(
	ctx context.Context,
	message ExtractExternalDataMessage,
	connections []*model.Connection,
	result map[string]map[string][]map[string]any,
) error {
	crmMessage, genericMessage := splitCRMCompatibility(message, connections)

	// plugin_crm portion: explicit CRM compatibility path, Worker-owned, byte-identical
	// to the legacy extraction (reuses the unchanged QueryPluginCRM chain).
	if crmMessage != nil {
		if err := uc.queryPluginCRMDatabase(ctx, *crmMessage, connections, result); err != nil {
			return err
		}
	}

	// Generic portion: the Engine runner. Skipped entirely for a CRM-only job so a
	// plan-then-execute call is never made with an empty request.
	if genericMessage != nil {
		if err := uc.extractViaEngine(ctx, *genericMessage, connections, result); err != nil {
			return err
		}
	}

	return nil
}

// splitCRMCompatibility partitions a message into its plugin_crm portion (extracted
// through the Worker CRM compatibility path) and its generic portion (extracted
// through the Engine runner). It returns nil for a portion that has no datasources,
// so a CRM-only or generic-only job runs exactly one path.
//
// The CRM selection is EXACT and explicit (plugincrm.IsPluginCRM on the config name,
// AND a MongoDB connection type), mirroring the legacy MongoDB/plugin_crm branch. A
// generic datasource — including names that merely resemble "plugin_crm" — never
// triggers CRM behavior. Metadata is carried on BOTH portions unchanged so each path
// sees the same opaque metadata.source it does today.
func splitCRMCompatibility(
	message ExtractExternalDataMessage,
	connections []*model.Connection,
) (crm, generic *ExtractExternalDataMessage) {
	crmTypes := mongoConfigsByName(connections)

	crmFields := map[string]map[string][]string{}
	genericFields := map[string]map[string][]string{}

	for configName, tables := range message.MappedFields {
		if plugincrm.IsPluginCRM(configName) && crmTypes[configName] {
			crmFields[configName] = tables
		} else {
			genericFields[configName] = tables
		}
	}

	if len(crmFields) > 0 {
		crm = subMessage(message, crmFields)
	}

	if len(genericFields) > 0 {
		generic = subMessage(message, genericFields)
	}

	return crm, generic
}

// mongoConfigsByName indexes which config names resolve to a MongoDB connection, so
// the CRM split predicate matches the legacy "MongoDB AND plugin_crm" condition.
func mongoConfigsByName(connections []*model.Connection) map[string]bool {
	mongo := make(map[string]bool, len(connections))

	for _, conn := range connections {
		if conn != nil && conn.Type == model.TypeMongoDB {
			mongo[conn.ConfigName] = true
		}
	}

	return mongo
}

// subMessage builds a message carrying only the given datasources' mapped fields,
// their matching filters, and the original (opaque) metadata. The job id is
// preserved so both portions report against the same job.
func subMessage(message ExtractExternalDataMessage, fields map[string]map[string][]string) *ExtractExternalDataMessage {
	var filters map[string]map[string]map[string]modelJob.FilterCondition

	if len(message.Filters) > 0 {
		filters = make(map[string]map[string]map[string]modelJob.FilterCondition, len(fields))
		for configName := range fields {
			if f, ok := message.Filters[configName]; ok {
				filters[configName] = f
			}
		}
	}

	return &ExtractExternalDataMessage{
		JobID:        message.JobID,
		MappedFields: fields,
		Filters:      filters,
		Metadata:     message.Metadata,
	}
}

// extractViaEngine maps the queued job into an engine.ExtractionRequest, bridges
// the tenant (tenantId + requestId ONLY — owner decision B2), seeds the resolved
// connections into the context (so the Engine never re-resolves), and invokes the
// Engine runner in DIRECT mode. The returned inline bytes are decoded back into the
// Worker's result map, which the UNCHANGED save path then encrypts, signs, and
// stores (ST-02 stays Worker-owned).
func (uc *UseCase) extractViaEngine(
	ctx context.Context,
	message ExtractExternalDataMessage,
	connections []*model.Connection,
	result map[string]map[string][]map[string]any,
) error {
	logger, tracer, _, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data.extract_via_engine")
	defer span.End()

	tenant, err := connectioncompat.TenantContextFromRequest(ctx)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to bridge tenant context for engine execution", err)
		return fmt.Errorf("bridge tenant context: %w", err)
	}

	request := mapJobToExtractionRequest(message, connections)

	// Seed the connections the Worker already resolved so the Engine's request-scoped
	// ConnectionStore returns them without re-resolving (tenant-manager stays out of
	// the Engine core). This reuses the SAME context-seed contract the Manager schema
	// path uses (schemacompat.WithResolvedConnections), so the worker runner can wire
	// schemacompat.NewConnectionStore() without a second divergent seed.
	ctx = schemacompat.WithResolvedConnections(ctx, connections)

	extractionResult, err := uc.EngineRunner.RunExtraction(ctx, tenant, message.JobID.String(), request)
	if err != nil {
		libOtel.HandleSpanError(span, "engine extraction failed", err)
		return fmt.Errorf("engine extraction: %w", err)
	}

	return decodeDirectResult(ctx, extractionResult, result, logger)
}

// decodeDirectResult unpacks a DIRECT-mode ExtractionResult into the Worker's
// result map. It enforces the result-shape invariant established at this
// integration: exactly one of Direct / Reference is non-nil. ST-01 wires DIRECT
// mode only (the Worker owns encrypt/store/HMAC, ST-02), so a Reference result —
// or a result with neither arm — is a contract violation here rather than a silent
// empty extraction.
func decodeDirectResult(
	ctx context.Context,
	res engine.ExtractionResult,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
) error {
	if res.Direct == nil && res.Reference != nil {
		return fmt.Errorf("engine returned a store-mode reference result; worker expects direct mode")
	}

	if res.Direct == nil {
		return fmt.Errorf("engine returned no direct result payload")
	}

	if len(res.Direct.Data) == 0 {
		// An empty payload is a valid empty extraction; leave result untouched.
		return nil
	}

	decoded := make(map[string]map[string][]map[string]any)
	if err := json.Unmarshal(res.Direct.Data, &decoded); err != nil {
		logger.Log(ctx, libLog.LevelError, "failed to decode direct engine result", libLog.Err(err))
		return fmt.Errorf("decode direct engine result: %w", err)
	}

	for db, tables := range decoded {
		result[db] = tables
	}

	return nil
}

// mapJobToExtractionRequest projects the queued job message onto an
// engine.ExtractionRequest. It is the host-side mapping half of owner decision
// Option 2 (T-010): mapped-field and filter TABLE KEYS are canonicalized per
// datasource type via tablenorm — the SAME schemautil rule the legacy adapters and
// the snapshot side use — so the Engine's literal table match behaves byte-
// identically to the legacy Worker's per-adapter normalization. metadata.source
// (e.g. "plugin_crm") is carried as OPAQUE Metadata the Engine never interprets.
func mapJobToExtractionRequest(message ExtractExternalDataMessage, connections []*model.Connection) engine.ExtractionRequest {
	typesByConfig := datasourceTypesByConfig(connections)

	return engine.ExtractionRequest{
		MappedFields: mapMappedFields(message.MappedFields, typesByConfig),
		Filters:      mapFilters(message.Filters, typesByConfig),
		Metadata:     message.Metadata,
	}
}

// datasourceTypesByConfig indexes the resolved connections by config name so the
// mapper can normalize each datasource's table keys with its own default-schema
// rule. A config absent from the map normalizes with an empty default schema
// (no prefix stripping), which is the safe identity for unknown types.
func datasourceTypesByConfig(connections []*model.Connection) map[string]model.DBType {
	byConfig := make(map[string]model.DBType, len(connections))

	for _, conn := range connections {
		if conn == nil {
			continue
		}

		if _, exists := byConfig[conn.ConfigName]; !exists {
			byConfig[conn.ConfigName] = conn.Type
		}
	}

	return byConfig
}

// mapMappedFields converts the Worker selection into map[string]engine.FieldSelection,
// canonicalizing each table key AND each field name for its datasource type. Table
// keys use tablenorm.NormalizeTable; field names use tablenorm.NormalizeField (a
// no-op for PG/MySQL/SQLServer, which were case-sensitive; UPPERCASE for Oracle,
// which stores identifiers uppercased and matched case-insensitively in the legacy
// path). A datasource with two table names canonicalizing to the same key keeps the
// first (a benign dedupe that matches the legacy lookup, which would resolve both to
// one table).
func mapMappedFields(
	mappedFields map[string]map[string][]string,
	typesByConfig map[string]model.DBType,
) map[string]engine.FieldSelection {
	if mappedFields == nil {
		return nil
	}

	out := make(map[string]engine.FieldSelection, len(mappedFields))

	for configName, tables := range mappedFields {
		dbType := typesByConfig[configName]
		selection := make(engine.FieldSelection, len(tables))

		for table, fields := range tables {
			selection[tablenorm.NormalizeTable(dbType, table)] = normalizeFieldsForType(dbType, fields)
		}

		out[configName] = selection
	}

	return out
}

// normalizeFieldsForType canonicalizes a field-name slice for the datasource type.
// It is a no-op for types whose identifiers are case-sensitive (PG/MySQL/SQLServer),
// and folds to UPPERCASE for Oracle, restoring the legacy Oracle EqualFold/ToUpper
// matching at the host seam so the Engine's literal field membership succeeds. A nil
// input yields nil (shared, no allocation) so the no-op path stays allocation-free.
func normalizeFieldsForType(dbType model.DBType, fields []string) []string {
	if !tablenorm.FoldsFieldCase(dbType) || fields == nil {
		return fields
	}

	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = tablenorm.NormalizeField(dbType, field)
	}

	return out
}

// mapFilters projects the Worker's typed nested filters into the Engine's opaque
// Filters payload. It emits FULLY-NESTED map[string]any (config -> table -> field ->
// value) because that is the shape the planner's datasourceFilters reads
// (planner.go: it asserts map[string]any at the config AND table levels and copies
// each field's value as an opaque any). Emitting the typed
// map[string]map[string]job.FilterCondition at the config level — as a prior
// revision did — fails the planner's map[string]any assertion, so step.Filters is
// never set and the generic datasource extracts UNFILTERED (a data-scoping defect).
// The leaf VALUE is the modelJob.FilterCondition carried as any: it survives the
// plan->execute round-trip untouched and the adapter's filtersForConfig reconstructs
// the typed map at the connector boundary.
//
// Table keys are canonicalized per datasource type (tablenorm) so they align with
// the normalized mapped-field and snapshot table keys; field keys are canonicalized
// per type too (Oracle folds to UPPERCASE — see normalizeFieldsForType) so an Oracle
// filter field matches the uppercased schema identity, restoring legacy parity.
func mapFilters(
	filters map[string]map[string]map[string]modelJob.FilterCondition,
	typesByConfig map[string]model.DBType,
) map[string]any {
	if len(filters) == 0 {
		return nil
	}

	out := make(map[string]any, len(filters))

	for configName, tables := range filters {
		dbType := typesByConfig[configName]
		tableMap := make(map[string]any, len(tables))

		for table, conditions := range tables {
			fieldMap := make(map[string]any, len(conditions))
			for field, condition := range conditions {
				fieldMap[tablenorm.NormalizeField(dbType, field)] = condition
			}

			tableMap[tablenorm.NormalizeTable(dbType, table)] = fieldMap
		}

		out[configName] = tableMap
	}

	return out
}
