package app

import (
	"dd-nats/common/ddnats"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"encoding/json"
	"io/ioutil"
	"path"
	"time"
)

var Manifest *types.FileManifest

func createManifest(ctx *context) {
	ticker := time.NewTicker(10 * time.Second)

	for {
		<-ticker.C
		Manifest = new(types.FileManifest)
		Manifest.LastUpdate = time.Now().UTC()
		directoryManifest(ctx, "/", Manifest)

		filename := path.Join(ctx.BaseDir, "manifest.json")
		content, _ := json.Marshal(Manifest)
		ioutil.WriteFile(filename, content, 0777)

		if err := ddnats.Publish("file.manifest", Manifest); err != nil {
			logger.Error("File manifest", "Failed to publish file manifest to NATS, err: %s", err.Error())
		}
	}
}

func directoryManifest(ctx *context, dirname string, manifest *types.FileManifest) {
	readdir := path.Join(ctx.BaseDir, ctx.DoneDir, dirname)
	infos, err := ioutil.ReadDir(readdir)
	if err != nil {
		logger.Error("file manifest", "error reading directory: %s, error: %s", readdir, err.Error())
		return
	}

	for _, fi := range infos {
		if !fi.IsDir() {
			filename := path.Join(readdir, fi.Name())
			hash := calcHash(filename)
			hashvalue := hash.Sum(nil)

			info := types.FileInfo{Name: fi.Name(), Path: dirname, Size: int(fi.Size()), Date: fi.ModTime(), Hash: hashvalue}
			manifest.Files = append(manifest.Files, info)
		} else {
			directoryManifest(ctx, path.Join(dirname, fi.Name()), manifest)
		}
	}
}
