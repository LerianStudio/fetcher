package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/lib-commons/v5/commons"
	tmclient "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
	obsRuntime "github.com/LerianStudio/lib-observability/runtime"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	defaultTerminalEventRepairInterval = 30 * time.Second
	defaultTerminalEventRepairBatch    = 25
)

type pendingTerminalEventLister interface {
	ListPendingTerminalEvents(ctx context.Context, limit int) ([]*model.Job, error)
}

type activeTenantLister interface {
	GetActiveTenantsByService(ctx context.Context, service string) ([]*tmclient.TenantSummary, error)
}

type tenantMongoDatabaseResolver interface {
	GetDatabaseForTenant(ctx context.Context, tenantID string) (*mongo.Database, error)
}

// TerminalEventRepairer periodically retries terminal job notifications that
// were persisted as pending after the original extraction message was DLQed.
type TerminalEventRepairer struct {
	useCase  *UseCase
	logger   libLog.Logger
	interval time.Duration
	batch    int

	tenantService string
	tenantLister  activeTenantLister
	mongoResolver tenantMongoDatabaseResolver
	requireTenant bool
}

func NewTerminalEventRepairer(useCase *UseCase, logger libLog.Logger) *TerminalEventRepairer {
	if logger == nil {
		logger = libLog.NewNop()
	}

	return &TerminalEventRepairer{
		useCase:  useCase,
		logger:   logger,
		interval: defaultTerminalEventRepairInterval,
		batch:    defaultTerminalEventRepairBatch,
	}
}

func NewTerminalEventRepairerWithTenantScope(useCase *UseCase, logger libLog.Logger, service string, tenantLister activeTenantLister, mongoResolver tenantMongoDatabaseResolver) *TerminalEventRepairer {
	repairer := NewTerminalEventRepairer(useCase, logger)
	repairer.tenantService = service
	repairer.tenantLister = tenantLister
	repairer.mongoResolver = mongoResolver
	repairer.requireTenant = true

	return repairer
}

func (r *TerminalEventRepairer) Run(launcher *commons.Launcher) error {
	if r == nil || r.useCase == nil {
		return nil
	}

	logger := r.logger
	if launcher != nil && launcher.Logger != nil {
		logger = launcher.Logger
	}

	ctx := observability.ContextWithLogger(context.Background(), logger)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)

	obsRuntime.SafeGoWithContext(ctx, logger, "terminal-event-repairer-signal-handler", obsRuntime.KeepRunning, func(ctx context.Context) {
		select {
		case <-sigs:
			cancel()
		case <-ctx.Done():
		}
	})

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		if err := r.RepairOnce(ctx); err != nil {
			r.logger.Log(ctx, libLog.LevelError, "failed to repair pending terminal job events", libLog.Err(err))
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *TerminalEventRepairer) RepairOnce(ctx context.Context) error {
	logger, tracer, _, _ := observability.NewTrackingFromContext(ctx)
	if logger == nil {
		logger = r.logger
	}

	ctx, span := tracer.Start(ctx, "service.terminal_event_repairer.repair_once")
	defer span.End()

	lister, ok := r.useCase.JobRepository.(pendingTerminalEventLister)
	if !ok {
		return nil
	}

	if r.requireTenant {
		if r.tenantLister == nil || r.mongoResolver == nil || r.tenantService == "" {
			return fmt.Errorf("multi-tenant terminal event repairer requires tenant lister, mongo resolver, and service name")
		}

		tenants, err := r.tenantLister.GetActiveTenantsByService(ctx, r.tenantService)
		if err != nil {
			return fmt.Errorf("list active tenants for terminal event repair: %w", err)
		}

		var tenantErrs []error

		for _, tenant := range tenants {
			if tenant == nil || tenant.ID == "" {
				continue
			}

			tenantCtx := tmcore.ContextWithTenantID(ctx, tenant.ID)

			tenantDB, err := r.mongoResolver.GetDatabaseForTenant(tenantCtx, tenant.ID)
			if err != nil {
				tenantErr := fmt.Errorf("resolve tenant mongo for terminal event repair tenant %s: %w", tenant.ID, err)
				tenantErrs = append(tenantErrs, tenantErr)

				logger.Log(ctx, libLog.LevelError, "failed to resolve tenant mongo for terminal event repair", libLog.String("tenant_id", tenant.ID), libLog.Err(err))

				continue
			}

			tenantCtx = tmcore.ContextWithMB(tenantCtx, tenantDB)
			if err := r.repairOnceInContext(tenantCtx, lister, logger); err != nil {
				tenantErr := fmt.Errorf("repair pending terminal events for tenant %s: %w", tenant.ID, err)
				tenantErrs = append(tenantErrs, tenantErr)

				logger.Log(ctx, libLog.LevelError, "failed to repair pending terminal events for tenant", libLog.String("tenant_id", tenant.ID), libLog.Err(err))
			}
		}

		return errors.Join(tenantErrs...)
	}

	return r.repairOnceInContext(ctx, lister, logger)
}

func (r *TerminalEventRepairer) repairOnceInContext(ctx context.Context, lister pendingTerminalEventLister, logger libLog.Logger) error {
	jobs, err := lister.ListPendingTerminalEvents(ctx, r.batch)
	if err != nil {
		return fmt.Errorf("list pending terminal events: %w", err)
	}

	var errs []error

	for _, job := range jobs {
		if job == nil {
			continue
		}

		if _, err := r.useCase.retryPendingTerminalEventForJob(ctx, job, logger); err != nil {
			repairErr := fmt.Errorf("repair pending terminal event for job %s: %w", job.ID, err)
			logger.Log(ctx, libLog.LevelError, "failed to repair pending terminal job event",
				libLog.String("job_id", job.ID.String()),
				libLog.Err(err),
			)

			errs = append(errs, repairErr)
		}
	}

	return errors.Join(errs...)
}
