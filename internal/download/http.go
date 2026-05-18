package download

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registra endpoints REST, websocket y archivos estaticos.
//
// Args:
//
//	app: instancia Fiber.
//	manager: manager de jobs de descarga.
//	wsCfg: configuracion websocket de Fiber.
func RegisterRoutes(app *fiber.App, manager *Manager, wsCfg websocket.Config) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("web/index.html")
	})
	app.Get("/styles.css", func(c *fiber.Ctx) error {
		return c.SendFile("web/styles.css")
	})
	app.Get("/app.js", func(c *fiber.Ctx) error {
		return c.SendFile("web/app.js")
	})
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("assets/icon/icon.ico")
	})
	app.Get("/icon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("assets/icon/icon.ico")
	})

	api := app.Group("/api/download")
	api.Post("/start", func(c *fiber.Ctx) error {
		var req DownloadRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "payload invalido",
			})
		}

		job, err := manager.CreateAndQueueJob(req)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrQueueFull) {
				status = http.StatusTooManyRequests
			}
			return c.Status(status).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(http.StatusAccepted).JSON(StartResponse{
			JobID:  job.ID,
			Status: job.Status,
		})
	})

	api.Get("/file/:jobId", func(c *fiber.Ctx) error {
		jobID := c.Params("jobId")
		job, ok := manager.GetSnapshot(jobID)
		if !ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "job no encontrado"})
		}
		if job.Status != JobStatusCompleted {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "archivo no disponible"})
		}

		filePath := resolveFinalPath(manager.cfg.DownloadDir, jobID, job.FilePath)
		if filePath == "" {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "ruta de archivo vacia"})
		}
		if _, err := os.Stat(filePath); err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "archivo no encontrado"})
		}

		fileName := filepath.Base(filePath)
		quotedFileName := strings.ReplaceAll(fileName, `"`, `\"`)
		encodedFileName := url.PathEscape(fileName)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", quotedFileName, encodedFileName))
		return c.SendFile(filePath, true)
	})

	app.Use("/ws/progress/:jobId", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws/progress/:jobId", websocket.New(func(conn *websocket.Conn) {
		jobID := conn.Params("jobId")
		events, unsubscribe, err := manager.Subscribe(jobID)
		if err != nil {
			_ = conn.WriteJSON(ProgressEvent{
				JobID:   jobID,
				Status:  JobStatusError,
				Message: err.Error(),
			})
			_ = conn.Close()
			return
		}
		defer unsubscribe()
		defer conn.Close()

		for event := range events {
			_ = conn.SetWriteDeadline(time.Now().Add(manager.cfg.WSWriteTimeout))
			if err := conn.WriteJSON(event); err != nil {
				return
			}
			if event.Status == JobStatusCompleted || event.Status == JobStatusError {
				return
			}
		}
	}, wsCfg))
}

func validateURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return errors.New("url invalida")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("solo se acepta http o https")
	}
	if parsed.Host == "" {
		return errors.New("url invalida: host vacio")
	}
	return nil
}
