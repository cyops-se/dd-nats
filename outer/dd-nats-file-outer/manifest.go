package main

import (
	"dd-nats/common/filetransfer"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

var manifestBuilder = filetransfer.NewManifestBuilder()

func createManifest(ctx *context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		manifest := manifestBuilder.Build(ctx.dir(ctx.donedir), func(err error) {
			ctx.svc.Error("File manifest", "Error building manifest: %s", err.Error())
		})

		filename := filepath.Join(ctx.basedir, "manifest.json")
		if content, err := json.Marshal(manifest); err == nil {
			if err = os.WriteFile(filename, content, 0644); err != nil {
				ctx.svc.Error("File manifest", "Failed to write manifest file %s, err: %s", filename, err.Error())
			}
		}

		if err := ctx.svc.Publish("file.manifest", manifest); err != nil {
			ctx.svc.Error("File manifest", "Failed to publish file manifest, err: %s", err.Error())
		}
	}
}
