package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"song-recognition/drone"
)

func main() {
	dir := flag.String("dir", filepath.Join("train_data"), "Directory containing labelled WAV template samples")
	out := flag.String("out", filepath.Join("drone", "templates.json"), "Output path for templates JSON")
	flag.Parse()

	templates, err := drone.BuildTemplatesFromDir(*dir)
	if err != nil {
		log.Fatalf("failed to build templates: %v", err)
	}

	if err := drone.SaveTemplates(*out, templates); err != nil {
		log.Fatalf("failed to save templates: %v", err)
	}

	fmt.Printf("Saved %d templates to %s\n", len(templates), *out)
}
