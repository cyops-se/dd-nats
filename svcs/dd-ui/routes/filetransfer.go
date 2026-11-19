package routes

import (
	"dd-nats/inner/dd-nats-file-inner/messages"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func registerFileTransferRoutes(api fiber.Router) {
	api.Post("/filetransfer/upload", UploadFilesToTransfer)
}

func UploadFilesToTransfer(c *fiber.Ctx) error {
	response, err := usvc.Request("usvc.filetransfer.listfolders", nil)
	if err != nil {
		e := usvc.Error("File transfer", "filetransfer usvc request failed: %s", err.Error())
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": e.Error()})
	}

	var msg messages.FolderInfo
	if err := json.Unmarshal(response, &msg); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": err.Error()})
	}

	if !msg.Success || msg.NewDir == "" {
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": "file transfer service not ready: " + msg.StatusMessage})
	}

	newdir := msg.NewDir
	file, err := c.FormFile("file")
	if err != nil {
		usvc.Error("File transfer", "No file provided: %s", err.Error())
		return c.Status(http.StatusBadRequest).JSON(&fiber.Map{"error": err.Error()})
	}

	// Only use the base name of the uploaded file, never a path
	name := filepath.Base(strings.ReplaceAll(file.Filename, "\\", "/"))
	if name == "." || name == ".." || name == "/" || name == "" {
		return c.Status(http.StatusBadRequest).JSON(&fiber.Map{"error": "invalid file name"})
	}

	usvc.Trace("File transfer", "Received file from upload: %s", name)

	// Make sure file transfer outgoing exists
	if err := os.MkdirAll(newdir, 0755); err != nil {
		e := usvc.Error("File transfer", "Failed to create directory %s: %s", newdir, err.Error())
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": e.Error()})
	}

	// Save to a temporary name first, so the file transfer service doesn't pick up a partial file
	filename := filepath.Join(newdir, name)
	tmpname := filepath.Join(filepath.Dir(newdir), "."+name+".upload")
	if err := c.SaveFile(file, tmpname); err != nil {
		e := usvc.Error("Upload of file to transfer failed", "failed to save file, name: '%s', size: %d, error: %s", name, file.Size, err.Error())
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": e.Error()})
	}

	if err := os.Rename(tmpname, filename); err != nil {
		os.Remove(tmpname)
		e := usvc.Error("Upload of file to transfer failed", "failed to move file, name: '%s', error: %s", name, err.Error())
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": e.Error()})
	}

	return c.Status(http.StatusOK).JSON(file)
}
