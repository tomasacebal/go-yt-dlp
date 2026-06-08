package download

import (
	"testing"
	"time"
)

func TestCreateAndQueueJob(t *testing.T) {
	manager := mustManager(t)

	job, err := manager.CreateAndQueueJob(DownloadRequest{
		URL: "https://www.youtube.com/watch?v=abc123",
		Flags: DownloadFlags{
			Format:  "mp4",
			Quality: "best",
		},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if job.ID == "" {
		t.Fatalf("job id vacio")
	}
	if job.Status != JobStatusQueued {
		t.Fatalf("status inesperado: %s", job.Status)
	}
}

func TestCreateAndQueueJobQueueFull(t *testing.T) {
	cfg := LoadConfig()
	cfg.QueueCapacity = 1
	cfg.WorkerCount = 1
	cfg.DownloadDir = t.TempDir()

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	_, err = manager.CreateAndQueueJob(DownloadRequest{
		URL: "https://www.youtube.com/watch?v=one",
		Flags: DownloadFlags{
			Format:  "mp4",
			Quality: "best",
		},
	})
	if err != nil {
		t.Fatalf("no se esperaba error en primer job: %v", err)
	}

	_, err = manager.CreateAndQueueJob(DownloadRequest{
		URL: "https://www.youtube.com/watch?v=two",
		Flags: DownloadFlags{
			Format:  "mp4",
			Quality: "best",
		},
	})
	if err == nil {
		t.Fatalf("se esperaba error de cola llena")
	}
}

func TestSubscribeNotFound(t *testing.T) {
	manager := mustManager(t)
	_, _, err := manager.Subscribe("no-existe")
	if err == nil {
		t.Fatalf("se esperaba error para job inexistente")
	}
}

func mustManager(t *testing.T) *Manager {
	t.Helper()

	cfg := LoadConfig()
	cfg.WorkerCount = 1
	cfg.QueueCapacity = 2
	cfg.DownloadDir = t.TempDir()
	cfg.CleanupEvery = 10 * time.Minute
	cfg.FileTTL = time.Hour

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("error al crear manager: %v", err)
	}
	return manager
}
