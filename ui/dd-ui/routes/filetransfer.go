package routes

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/logger"
	"dd-nats/inner/dd-nats-file-inner/messages"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gofiber/fiber/v2"
)

func RegisterFileTransferRoutes(api fiber.Router) {
	api.Post("/filetransfer/upload", UploadFilesToTransfer)
}

func UploadFilesToTransfer(c *fiber.Ctx) error {
	response, err := ddnats.Request("usvc.filetransfer.listfolders", nil)
	if err != nil {
		log.Printf("filetransfer usvc request failed: %s", err.Error())
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": err.Error()})
	}

	var msg messages.FolderInfo
	if err := json.Unmarshal(response.Data, &msg); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": err.Error()})
	}

	newdir := msg.NewDir
	log.Println("new folder:", newdir)

	file, err := c.FormFile("file")
	if err != nil {
		logger.Log("error", "No file provided", err.Error())
		return c.Status(503).SendString(err.Error())
	}

	logger.Trace("File transfer", "Received file from upload: %s", file.Filename)

	// Make sure file transfer outging exists
	// TODO: use proper interface to filetransfer for this (now hardcoded to ./outgoing/new)
	if _, err := os.Stat(newdir); os.IsNotExist(err) {
		os.MkdirAll(newdir, 0755)
	}

	filename := path.Join(newdir, file.Filename) //fmt.Sprintf("./outgoing/new/%s", file.Filename)
	if err := c.SaveFile(file, filename); err != nil {
		// msg := fmt.Sprintf("failed to save file, name: '%s', size: %d, error: %s", file.Filename, file.Size, err.Error())
		e := logger.Error("Upload of file to transfer failed", "failed to save file, name: '%s', size: %d, error: %s", file.Filename, file.Size, err.Error())
		return c.Status(http.StatusInternalServerError).JSON(&fiber.Map{"error": e.Error()})
	}

	return c.Status(http.StatusOK).JSON(file)
}
