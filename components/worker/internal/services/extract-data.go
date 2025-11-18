package services

import (
	"github.com/LerianStudio/fetcher/pkg/model"
)

// GenerateReportMessage contains the information needed to generate a report.
type GenerateReportMessage struct {
	// DataQueries maps database names to tables and their fields.
	// Format: map[databaseName]map[tableName][]fieldName.
	// Example: {"onboarding": {"organization": ["name"], "ledger": ["id"]}}.
	MappedFields map[string]map[string][]string `json:"mappedFields"`

	// Filters specify advanced filtering criteria using FilterCondition for complex queries.
	// Format: map[databaseName]map[tableName]map[fieldName]model.FilterCondition
	// Example: {"db": {"table": {"created_at": {"gte": ["2025-06-01"], "lte": ["2025-06-30"]}}}}
	Filters map[string]map[string]map[string]model.FilterCondition `json:"filters"`

	// Metadata contains additional metadata for the report.
	Metadata map[string]any `json:"metadata"`
}

// GenerateReport handles a report generation request by loading a template file,
// processing it, and storing the final report in the report repository.
func (uc *UseCase) GenerateReport() error {
	//logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)
	//
	//ctx, span := tracer.Start(ctx, "service.generate_report")
	//defer span.End()
	//
	//message, err := uc.parseMessage(ctx, body, &span, logger)
	//if err != nil {
	//	return err
	//}
	//
	//if skip := uc.shouldSkipProcessing(ctx, message.ReportID, logger); skip {
	//	return nil
	//}
	//
	//templateBytes, err := uc.loadTemplate(ctx, tracer, message, &span, logger)
	//if err != nil {
	//	return err
	//}
	//
	//result := make(map[string]map[string][]map[string]any)
	//
	//if err := uc.queryExternalData(ctx, message, result); err != nil {
	//	return uc.handleErrorWithUpdate(ctx, message.ReportID, &span, "Error querying external data", err, logger)
	//}
	//
	//renderedOutput, err := uc.renderTemplate(ctx, tracer, templateBytes, result, message, &span, logger)
	//if err != nil {
	//	return err
	//}
	//
	//finalOutput, err := uc.convertToPDFIfNeeded(ctx, tracer, message, renderedOutput, &span, logger)
	//if err != nil {
	//	return err
	//}
	//
	//if err := uc.saveReport(ctx, tracer, message, finalOutput, logger); err != nil {
	//	return uc.handleErrorWithUpdate(ctx, message.ReportID, &span, "Error saving report", err, logger)
	//}
	//
	//if err := uc.markReportAsFinished(ctx, message.ReportID, &span, logger); err != nil {
	//	return err
	//}

	return nil
}
