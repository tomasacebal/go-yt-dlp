package download

import "time"

// JobStatus representa el estado del job de descarga.
type JobStatus string

const (
	JobStatusQueued      JobStatus = "queued"
	JobStatusDownloading JobStatus = "downloading"
	JobStatusCompleted   JobStatus = "completed"
	JobStatusError       JobStatus = "error"
)

// DownloadFlags contiene las opciones de descarga enviadas por el cliente.
type DownloadFlags struct {
	Format    string `json:"format"`
	AudioOnly bool   `json:"audioOnly"`
	Quality   string `json:"quality"`
	EmbedSubs bool   `json:"embedSubs"`
}

// DownloadRequest define el payload de inicio de descarga.
type DownloadRequest struct {
	URL   string        `json:"url"`
	Flags DownloadFlags `json:"flags"`
}

// StartResponse define la respuesta al iniciar un job.
type StartResponse struct {
	JobID  string    `json:"jobId"`
	Status JobStatus `json:"status"`
}

// ProgressEvent representa un evento emitido por WebSocket.
type ProgressEvent struct {
	JobID    string    `json:"jobId"`
	Status   JobStatus `json:"status"`
	Progress float64   `json:"progress,omitempty"`
	Speed    string    `json:"speed,omitempty"`
	ETA      string    `json:"eta,omitempty"`
	Message  string    `json:"message,omitempty"`
}

// JobSnapshot expone estado de un job para handlers.
type JobSnapshot struct {
	ID         string
	Status     JobStatus
	FilePath   string
	URL        string
	LastError  string
	Progress   float64
	Speed      string
	ETA        string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	FinishedAt *time.Time
}

type jobRecord struct {
	id         string
	url        string
	flags      DownloadFlags
	status     JobStatus
	filePath   string
	lastError  string
	progress   float64
	speed      string
	eta        string
	createdAt  time.Time
	updatedAt  time.Time
	finishedAt *time.Time
}
