package audit

import (
	"context"
	"sync"
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	logChanCapacity = 1024
	batchSize       = 100
	flushInterval   = 2 * time.Second
)

// AuditRecorder is a bounded, best-effort asynchronous audit sink. It is not a
// durable task queue: database failures and overload are counted and logged.
type AuditRecorder struct {
	db      *gorm.DB
	logger  *zap.Logger
	logChan chan model.SysOperationLog
	mu      sync.RWMutex
	closed  bool
	done    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	dropped metric.Int64Counter
}

func NewAuditRecorder(db *gorm.DB, logger *zap.Logger) *AuditRecorder {
	ctx, cancel := context.WithCancel(context.Background())
	if logger == nil {
		logger = zap.NewNop()
	}
	counter, err := otel.Meter("base-frame/audit").Int64Counter("cms.audit.dropped", metric.WithDescription("Audit records lost due to overload or database errors"))
	if err != nil {
		otel.Handle(err)
	}
	r := &AuditRecorder{db: db, logger: logger, logChan: make(chan model.SysOperationLog, logChanCapacity), done: make(chan struct{}), ctx: ctx, cancel: cancel, dropped: counter}
	go r.startWorker()
	return r
}

func (r *AuditRecorder) Push(record model.SysOperationLog) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return
	}
	select {
	case r.logChan <- record:
	default:
		r.recordDrop(1, "queue_full")
	}
}

func (r *AuditRecorder) recordDrop(count int, reason string) {
	if r.dropped != nil {
		r.dropped.Add(context.Background(), int64(count), metric.WithAttributes(attribute.String("reason", reason)))
	}
	r.logger.Warn("operation_logs_dropped", zap.Int("count", count), zap.String("reason", reason))
}

func (r *AuditRecorder) startWorker() {
	defer close(r.done)
	defer r.cancel()
	batch := make([]model.SysOperationLog, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			r.recordDrop(len(batch)+len(r.logChan), "shutdown_timeout")
			return
		case record, ok := <-r.logChan:
			if !ok {
				if len(batch) > 0 {
					r.flush(batch)
				}
				return
			}
			batch = append(batch, record)
			if len(batch) >= batchSize {
				r.flush(batch)
				clear(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				r.flush(batch)
				clear(batch)
				batch = batch[:0]
			}
		}
	}
}

func (r *AuditRecorder) flush(logs []model.SysOperationLog) {
	ctx, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()
	if err := r.db.WithContext(ctx).CreateInBatches(logs, len(logs)).Error; err != nil {
		r.recordDrop(len(logs), "database_error")
		r.logger.Error("flush_operation_logs_failed", zap.Error(err), zap.Int("count", len(logs)))
	}
}

// Close may run concurrently with Push and with other Close calls. The caller's
// deadline also cancels database work, so timeout cannot leave a draining worker.
func (r *AuditRecorder) Close(ctx context.Context) error {
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		close(r.logChan)
	}
	r.mu.Unlock()
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		r.cancel()
		return ctx.Err()
	}
}
