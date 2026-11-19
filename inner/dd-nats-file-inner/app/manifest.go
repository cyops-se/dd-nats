package app

import (
	"dd-nats/common/filetransfer"
	"dd-nats/common/types"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

var manifestBuilder = filetransfer.NewManifestBuilder()

// GetManifest returns the latest manifest (empty until the first one is built)
func GetManifest() types.FileManifest {
	return manifestBuilder.Manifest()
}

func createManifest(ctx *context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		manifest := manifestBuilder.Build(ctx.Dir(ctx.DoneDir), func(err error) {
			ctx.svc.Error("File manifest", "Error building manifest: %s", err.Error())
		})

		filename := filepath.Join(ctx.BaseDir, "manifest.json")
		if content, err := json.Marshal(manifest); err == nil {
			if err = os.WriteFile(filename, content, 0644); err != nil {
				ctx.svc.Error("File manifest", "Failed to write manifest file %s, err: %s", filename, err.Error())
			}
		}

		if err := ctx.svc.Publish("file.manifest", manifest); err != nil {
			ctx.svc.Error("File manifest", "Failed to publish file manifest, err: %s", err.Error())
		}

		<-ticker.C
	}
}
