package workers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/thenasky/go-framework/modules/email/models"
	"github.com/thenasky/go-framework/modules/email/providers"
	"github.com/thenasky/go-framework/modules/email/queue"
)

// EmailWorker processes email jobs from the queue
type EmailWorker struct {
	queue           *queue.MongoQueue
	providers       []providers.EmailProvider
	workerCount     int
	stopChan        chan struct{}
	wg              sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc
	processingDelay time.Duration
}

// WorkerConfig holds configuration for the email worker
type WorkerConfig struct {
	WorkerCount     int           `json:"worker_count"`     // Number of worker goroutines
	ProcessingDelay time.Duration `json:"processing_delay"` // Delay between job checks
	MaxRetries      int           `json:"max_retries"`      // Maximum retry attempts
	RetryDelay      time.Duration `json:"retry_delay"`      // Delay between retries
}

// DefaultWorkerConfig returns sensible default configuration
func DefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		WorkerCount:     2,                      // 2 workers by default
		ProcessingDelay: 100 * time.Millisecond, // Check every 100ms
		MaxRetries:      3,                      // Max 3 retries
		RetryDelay:      5 * time.Minute,        // Wait 5 minutes between retries
	}
}

// NewEmailWorker creates a new email worker
func NewEmailWorker(queue *queue.MongoQueue, providers []providers.EmailProvider, config *WorkerConfig) *EmailWorker {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EmailWorker{
		queue:           queue,
		providers:       providers,
		workerCount:     config.WorkerCount,
		stopChan:        make(chan struct{}),
		ctx:             ctx,
		cancel:          cancel,
		processingDelay: config.ProcessingDelay,
	}
}

// Start starts the email worker
func (w *EmailWorker) Start() {
	log.Printf("Starting email worker with %d workers", w.workerCount)

	// Start worker goroutines
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.workerRoutine(i)
	}

	// Start cleanup routine
	w.wg.Add(1)
	go w.cleanupRoutine()

	log.Println("Email worker started successfully")
}

// Stop stops the email worker gracefully
func (w *EmailWorker) Stop() {
	log.Println("Stopping email worker...")

	// Signal all workers to stop
	close(w.stopChan)

	// Cancel context
	w.cancel()

	// Wait for all workers to finish
	w.wg.Wait()

	log.Println("Email worker stopped successfully")
}

// workerRoutine is the main worker loop
func (w *EmailWorker) workerRoutine(workerID int) {
	defer w.wg.Done()

	log.Printf("Worker %d started", workerID)

	for {
		select {
		case <-w.stopChan:
			log.Printf("Worker %d stopping", workerID)
			return
		case <-w.ctx.Done():
			log.Printf("Worker %d context cancelled", workerID)
			return
		default:
			// Process next job
			if err := w.processNextJob(workerID); err != nil {
				log.Printf("Worker %d error: %v", workerID, err)
				// Small delay on error to prevent tight loop
				time.Sleep(1 * time.Second)
			}

			// Wait before checking for next job
			time.Sleep(w.processingDelay)

			// Add additional delay between workers to prevent rate limiting
			if workerID == 0 {
				time.Sleep(2 * time.Second)
			} else {
				time.Sleep(3 * time.Second)
			}
		}
	}
}

// processNextJob processes the next available job
func (w *EmailWorker) processNextJob(workerID int) error {
	// Get next job from queue
	job, err := w.queue.Dequeue()
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}

	// No jobs available
	if job == nil {
		return nil
	}

	log.Printf("Worker %d processing job %s (to: %s)", workerID, job.ID.Hex(), job.To)

	// Process the job
	if err := w.processJob(job); err != nil {
		log.Printf("Worker %d failed to process job %s: %v", workerID, job.ID.Hex(), err)

		// Check if this is a rate limiting error
		if strings.Contains(err.Error(), "Too many login attempts") ||
			strings.Contains(err.Error(), "rate limit") ||
			strings.Contains(err.Error(), "429") ||
			strings.Contains(err.Error(), "454") {

			// For rate limiting, add exponential backoff delay
			backoffDelay := time.Duration(job.Attempts) * 30 * time.Second
			if backoffDelay > 5*time.Minute {
				backoffDelay = 5 * time.Minute
			}

			log.Printf("Rate limiting detected, backing off for %v before retry", backoffDelay)
			time.Sleep(backoffDelay)

			// Don't mark as failed immediately, let it retry later
			return err
		}

		// Mark job as failed for non-rate-limiting errors
		if markErr := w.queue.MarkFailed(job.ID, err.Error()); markErr != nil {
			log.Printf("Worker %d failed to mark job %s as failed: %v", workerID, job.ID.Hex(), markErr)
		}

		return err
	}

	log.Printf("Worker %d successfully processed job %s", workerID, job.ID.Hex())
	return nil
}

// processJob sends an email using available providers
func (w *EmailWorker) processJob(job *models.EmailJob) error {
	var lastError error

	// Try each provider until one succeeds
	for _, provider := range w.providers {
		// Validate email before sending
		if err := provider.ValidateEmail(job.To); err != nil {
			lastError = fmt.Errorf("email validation failed: %w", err)
			continue
		}

		// Try to send email
		if err := provider.Send(job); err != nil {
			lastError = fmt.Errorf("provider %s failed: %w", provider.GetName(), err)
			continue
		}

		// Success! Mark job as complete
		providerName := provider.GetName()
		providerMsgID := fmt.Sprintf("msg_%d", time.Now().UnixNano()) // Generate unique ID

		if err := w.queue.MarkComplete(job.ID, providerName, providerMsgID); err != nil {
			return fmt.Errorf("failed to mark job complete: %w", err)
		}

		log.Printf("Email sent successfully via %s (job: %s)", providerName, job.ID.Hex())
		return nil
	}

	// All providers failed
	return fmt.Errorf("all providers failed to send email: %w", lastError)
}

// cleanupRoutine periodically cleans up old completed jobs
func (w *EmailWorker) cleanupRoutine() {
	defer w.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.queue.CleanupOldJobs(24 * time.Hour); err != nil {
				log.Printf("Cleanup routine error: %v", err)
			} else {
				log.Println("Cleanup routine completed successfully")
			}
		}
	}
}

// GetStats returns current worker statistics
func (w *EmailWorker) GetStats() (*models.EmailStats, error) {
	return w.queue.GetQueueStats()
}

// GetPendingCount returns the number of pending jobs
func (w *EmailWorker) GetPendingCount() (int64, error) {
	return w.queue.GetPendingJobsCount()
}

// IsRunning returns true if the worker is currently running
func (w *EmailWorker) IsRunning() bool {
	select {
	case <-w.stopChan:
		return false
	default:
		return true
	}
}
