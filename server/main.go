package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"song-recognition/utils"
	"song-recognition/wav"

	"github.com/joho/godotenv"
	"github.com/mdobak/go-xerrors"
)

func main() {
	err := utils.CreateFolder("tmp")
	if err != nil {
		logger := utils.GetLogger()
		err := xerrors.New(err)
		ctx := context.Background()
		logger.ErrorContext(ctx, "Failed create tmp dir.", slog.Any("error", err))
	}

	if len(os.Args) < 2 {
		fmt.Println("Expected 'serve' subcommand")
		os.Exit(1)
	}
	_ = godotenv.Load()

	switch os.Args[1] {
	case "serve":
		// Check for FFmpeg availability before starting server
		if err := wav.CheckFFmpegAvailable(); err != nil {
			log.Printf("WARNING: %v\n", err)
			log.Println("The server will start but audio processing will fail until FFmpeg is installed.")
		} else {
			log.Println("FFmpeg is available")
		}

		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		protocol := serveCmd.String("proto", "http", "Protocol to use (http or https)")
		port := serveCmd.String("p", "5000", "Port to use")
		serveCmd.Parse(os.Args[2:])
		serve(*protocol, *port)
	default:
		fmt.Println("Expected 'serve' subcommand")
		os.Exit(1)
	}
}
