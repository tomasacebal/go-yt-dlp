package download

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrQueueFull indica que no hay espacio en la cola de jobs.
	ErrQueueFull = errors.New("cola llena")

	// ErrJobNotFound indica que no existe el job solicitado.
	ErrJobNotFound = errors.New("job no encontrado")
)

// Manager coordina cola, workers, estado de jobs y eventos de progreso.
type Manager struct {
	cfg Config

	mu          sync.RWMutex
	jobs        map[string]*jobRecord
	subscribers map[string]map[chan ProgressEvent]struct{}

	queue chan string
}

// NewManager crea un manager listo para recibir jobs.
//
// Args:
//
//	cfg: configuracion operativa del servicio.
//
// Returns:
//
//	*Manager: instancia inicializada.
//	error: error si no puede preparar el directorio de descargas.
func NewManager(cfg Config) (*Manager, error) {
	if err := os.MkdirAll(cfg.DownloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("no se pudo crear directorio de descargas: %w", err)
	}

	return &Manager{
		cfg:         cfg,
		jobs:        make(map[string]*jobRecord),
		subscribers: make(map[string]map[chan ProgressEvent]struct{}),
		queue:       make(chan string, cfg.QueueCapacity),
	}, nil
}

// Start inicia workers y limpieza periodica hasta que el contexto finalice.
//
// Args:
//
//	ctx: contexto de ciclo de vida del servicio.
func (m *Manager) Start(ctx context.Context) {
	for i := 0; i < m.cfg.WorkerCount; i++ {
		go m.workerLoop(ctx, i+1)
	}
	go m.cleanupLoop(ctx)
}

// CreateAndQueueJob valida y encola un nuevo job.
//
// Args:
//
//	req: datos de la descarga solicitada.
//
// Returns:
//
//	JobSnapshot: snapshot inicial en estado queued.
//	error: error de validacion o cola llena.
func (m *Manager) CreateAndQueueJob(req DownloadRequest) (JobSnapshot, error) {
	req.URL = normalizeDownloadURL(req.URL)
	if err := validateURL(req.URL); err != nil {
		return JobSnapshot{}, err
	}

	if err := validateAndNormalizeFlags(&req.Flags); err != nil {
		return JobSnapshot{}, err
	}

	jobID := uuid.NewString()
	now := time.Now()
	log.Printf("nuevo job %s url=%s audio_only=%t quality=%s", jobID, req.URL, req.Flags.AudioOnly, req.Flags.Quality)

	record := &jobRecord{
		id:        jobID,
		url:       req.URL,
		flags:     req.Flags,
		status:    JobStatusQueued,
		createdAt: now,
		updatedAt: now,
	}

	m.mu.Lock()
	m.jobs[jobID] = record
	m.mu.Unlock()

	select {
	case m.queue <- jobID:
	default:
		m.mu.Lock()
		delete(m.jobs, jobID)
		m.mu.Unlock()
		return JobSnapshot{}, ErrQueueFull
	}

	event := ProgressEvent{
		JobID:    jobID,
		Status:   JobStatusQueued,
		Message:  "job en cola",
		Progress: 0,
	}
	m.publishAndApply(jobID, event)

	return m.snapshot(record), nil
}

// GetSnapshot devuelve el estado actual de un job.
//
// Args:
//
//	jobID: identificador del job.
//
// Returns:
//
//	JobSnapshot: snapshot del job.
//	bool: true si el job existe.
func (m *Manager) GetSnapshot(jobID string) (JobSnapshot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, ok := m.jobs[jobID]
	if !ok {
		return JobSnapshot{}, false
	}
	return m.snapshot(record), true
}

// Subscribe registra un canal para recibir eventos de un job.
//
// Args:
//
//	jobID: identificador del job.
//
// Returns:
//
//	<-chan ProgressEvent: canal de solo lectura para eventos.
//	func(): funcion de desuscripcion.
//	error: error si no existe el job.
func (m *Manager) Subscribe(jobID string) (<-chan ProgressEvent, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.jobs[jobID]
	if !ok {
		return nil, nil, ErrJobNotFound
	}

	ch := make(chan ProgressEvent, 32)
	if _, exists := m.subscribers[jobID]; !exists {
		m.subscribers[jobID] = make(map[chan ProgressEvent]struct{})
	}
	m.subscribers[jobID][ch] = struct{}{}

	initial := ProgressEvent{
		JobID:    record.id,
		Status:   record.status,
		Progress: record.progress,
		Speed:    record.speed,
		ETA:      record.eta,
		Message:  record.lastError,
	}
	ch <- initial

	unsubscribe := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if subs, exists := m.subscribers[jobID]; exists {
			if _, present := subs[ch]; present {
				delete(subs, ch)
				close(ch)
			}
			if len(subs) == 0 {
				delete(m.subscribers, jobID)
			}
		}
	}

	return ch, unsubscribe, nil
}

func (m *Manager) workerLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case jobID := <-m.queue:
			if err := m.executeJob(ctx, jobID); err != nil {
				log.Printf("worker %d fallo job %s: %v", workerID, jobID, err)
			}
		}
	}
}

func (m *Manager) executeJob(ctx context.Context, jobID string) error {
	m.mu.RLock()
	record, ok := m.jobs[jobID]
	if !ok {
		m.mu.RUnlock()
		return ErrJobNotFound
	}
	req := DownloadRequest{
		URL:   record.url,
		Flags: record.flags,
	}
	m.mu.RUnlock()

	args, _, err := buildYTDLPArgs(jobID, req, m.cfg.DownloadDir)
	if err != nil {
		m.publishAndApply(jobID, ProgressEvent{
			JobID:   jobID,
			Status:  JobStatusError,
			Message: err.Error(),
		})
		return err
	}
	if err := unlockChromiumCookiesIfNeeded(m.cfg); err != nil {
		log.Printf("job %s aviso unlock cookies: %v", jobID, err)
	}
	args = appendRuntimeArgs(args, m.cfg)
	log.Printf("job %s ejecutando: %s %s", jobID, m.cfg.YTDLPBin, strings.Join(args, " "))

	m.publishAndApply(jobID, ProgressEvent{
		JobID:    jobID,
		Status:   JobStatusDownloading,
		Progress: 0,
		Message:  "iniciando descarga",
	})

	var finalPath string
	var lastErrLine string

	runErr := runYTDLP(ctx, m.cfg.YTDLPBin, args, func(line string) {
		line = sanitizeLogLine(line)
		if line == "" {
			return
		}
		log.Printf("job %s yt-dlp: %s", jobID, line)

		if event, ok := parseProgressLine(line); ok {
			event.JobID = jobID
			m.publishAndApply(jobID, event)
			return
		}

		if strings.HasPrefix(line, "__FILEPATH__:") {
			finalPath = strings.TrimSpace(strings.TrimPrefix(line, "__FILEPATH__:"))
			return
		}

		if strings.Contains(strings.ToLower(line), "error") || strings.Contains(strings.ToLower(line), "fatal") {
			lastErrLine = line
		}
	})

	if runErr != nil {
		message := "fallo yt-dlp"
		if lastErrLine != "" {
			message = lastErrLine
		} else if runErr != nil {
			message = runErr.Error()
		}
		lowerMessage := strings.ToLower(message)
		if strings.Contains(lowerMessage, "sign in to confirm") && m.cfg.CookiesBrowser == "" && m.cfg.CookiesFile == "" {
			message = message + " | configura YTDLP_COOKIES_FROM_BROWSER o YTDLP_COOKIES_FILE"
		}
		m.publishAndApply(jobID, ProgressEvent{
			JobID:    jobID,
			Status:   JobStatusError,
			Progress: 0,
			Message:  message,
		})
		return runErr
	}

	finalPath = resolveFinalPath(m.cfg.DownloadDir, jobID, finalPath)
	if finalPath == "" {
		err := fmt.Errorf("no se encontro archivo final para job %s", jobID)
		m.publishAndApply(jobID, ProgressEvent{
			JobID:    jobID,
			Status:   JobStatusError,
			Progress: 0,
			Message:  err.Error(),
		})
		return err
	}

	m.mu.Lock()
	if record, exists := m.jobs[jobID]; exists {
		record.filePath = finalPath
	}
	m.mu.Unlock()

	m.publishAndApply(jobID, ProgressEvent{
		JobID:    jobID,
		Status:   JobStatusCompleted,
		Progress: 100,
		Message:  "descarga completada",
	})
	log.Printf("job %s completado archivo=%s", jobID, finalPath)

	return nil
}

func (m *Manager) publishAndApply(jobID string, event ProgressEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, exists := m.jobs[jobID]
	if !exists {
		return
	}

	record.status = event.Status
	record.progress = event.Progress
	record.speed = event.Speed
	record.eta = event.ETA
	record.updatedAt = time.Now()
	if event.Status == JobStatusError {
		record.lastError = event.Message
	}
	if event.Status == JobStatusCompleted || event.Status == JobStatusError {
		t := time.Now()
		record.finishedAt = &t
	}
	log.Printf("job %s evento status=%s progress=%.1f speed=%s eta=%s msg=%s", jobID, event.Status, event.Progress, event.Speed, event.ETA, event.Message)

	if subs, ok := m.subscribers[jobID]; ok {
		for ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

func (m *Manager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.CleanupEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanupExpired()
		}
	}
}

func (m *Manager) cleanupExpired() {
	cutoff := time.Now().Add(-m.cfg.FileTTL)

	var removeIDs []string
	var removeFiles []string

	m.mu.RLock()
	for id, job := range m.jobs {
		if job.finishedAt == nil || job.finishedAt.After(cutoff) {
			continue
		}
		removeIDs = append(removeIDs, id)
		if job.filePath != "" {
			removeFiles = append(removeFiles, job.filePath)
		}
	}
	m.mu.RUnlock()

	for _, filePath := range removeFiles {
		clean := filepath.Clean(filePath)
		baseAbs, baseErr := filepath.Abs(m.cfg.DownloadDir)
		fileAbs, fileErr := filepath.Abs(clean)
		if baseErr == nil && fileErr == nil && strings.HasPrefix(fileAbs, baseAbs) {
			_ = os.Remove(fileAbs)
			_ = os.Remove(filepath.Dir(fileAbs))
		}
	}

	if len(removeIDs) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range removeIDs {
		delete(m.jobs, id)
		if subs, ok := m.subscribers[id]; ok {
			for ch := range subs {
				close(ch)
			}
		}
		delete(m.subscribers, id)
	}
}

func (m *Manager) snapshot(job *jobRecord) JobSnapshot {
	return JobSnapshot{
		ID:         job.id,
		Status:     job.status,
		FilePath:   job.filePath,
		URL:        job.url,
		LastError:  job.lastError,
		Progress:   job.progress,
		Speed:      job.speed,
		ETA:        job.eta,
		CreatedAt:  job.createdAt,
		UpdatedAt:  job.updatedAt,
		FinishedAt: job.finishedAt,
	}
}

func inferFinalPathFallback(dir string, jobID string) string {
	// Formato actual: data/downloads/<jobID>/<titulo> [id].ext
	matches, _ := filepath.Glob(filepath.Join(dir, jobID, "*"))
	picked := pickNewestUsableFile(matches)
	if picked != "" {
		return picked
	}

	// Formato legacy: data/downloads/<jobID>.ext
	legacy, _ := filepath.Glob(filepath.Join(dir, jobID+".*"))
	return pickNewestUsableFile(legacy)
}

func resolveFinalPath(dir, jobID, hinted string) string {
	if hinted != "" {
		if info, err := os.Stat(hinted); err == nil && !info.IsDir() {
			return hinted
		}
	}
	return inferFinalPathFallback(dir, jobID)
}

func pickNewestUsableFile(candidates []string) string {
	var picked string
	var pickedTime time.Time
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		lower := strings.ToLower(candidate)
		if strings.HasSuffix(lower, ".part") || strings.HasSuffix(lower, ".tmp") {
			continue
		}
		if picked == "" || info.ModTime().After(pickedTime) {
			picked = candidate
			pickedTime = info.ModTime()
		}
	}
	return picked
}
