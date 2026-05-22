package services

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LerianStudio/fetcher/pkg/model"
	"github.com/LerianStudio/lib-commons/v5/commons"
	observability "github.com/LerianStudio/lib-observability"
	libLog "github.com/LerianStudio/lib-observability/log"
)

const (
	defaultTerminalEventRepairInterval = 30 * time.Second
	defaultTerminalEventRepairBatch    = 25
)

type pendingTerminalEventLister interface {
	ListPendingTerminalEvents(ctx context.Context, limit int) ([]*model.Job, error)
}

// TerminalEventRepairer periodically retries terminal job notifications that
// were persisted as pending after the original extraction message was DLQed.
type TerminalEventRepairer struct {
	useCase  *UseCase
	logger   libLog.Logger
	interval time.Duration
	batch    int
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

	go func() {
		select {
		case <-sigs:
			cancel()
		case <-ctx.Done():
		}
	}()

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

	jobs, err := lister.ListPendingTerminalEvents(ctx, r.batch)
	if err != nil {
		return fmt.Errorf("list pending terminal events: %w", err)
	}

	for _, job := range jobs {
		if job == nil {
			continue
		}

		if _, err := r.useCase.retryPendingTerminalEventForJob(ctx, job, logger); err != nil {
			return err
		}
	}

	return nil
}
