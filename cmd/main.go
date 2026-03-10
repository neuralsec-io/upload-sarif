package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})

	fs := afero.NewOsFs()

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer stop()

	flags, err := parseFlags(os.Args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if flags.Version {
		fmt.Printf("%s (commit %s, built at %s)\n", version, commit, date)
		os.Exit(0)
	}

	zerolog.SetGlobalLevel(flags.LogLevel)

	log.Debug().
		Str("version", version).
		Str("commit", commit).
		Str("date", date).
		Any("flags", flags).
		Msg("parsed flags")

	sarifFiles, err := detectSarifFiles(fs, flags.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to detect SARIF files")
	}

	if len(sarifFiles) == 0 {
		log.Info().Msg("No SARIF files found — nothing to upload.")
		os.Exit(0)
	}

	log.Info().
		Int("count", len(sarifFiles)).
		Msg("Detected SARIF files")

	uploader := NewSarifUploader(
		"https://api.neuralsec.io/v1/sarif/github",
		flags.APIKey,
		flags.GitHubRepoOwner,
		flags.GitHubRepoName,
		flags.Revision,
		fs,
	)

	for _, sarifPath := range sarifFiles {
		select {
		case <-ctx.Done():
			log.Warn().Msg("Upload cancelled by user or signal")
			os.Exit(1)
		default:
			log.Info().Str("file", sarifPath).Msg("Uploading SARIF file...")

			if err := uploader.Upload(ctx, sarifPath); err != nil {
				log.Error().Err(err).Str("file", sarifPath).Msg("Failed to upload SARIF file")
				continue
			}

			log.Info().Str("file", sarifPath).Msg("Successfully uploaded SARIF file ✅")
		}
	}

	log.Info().Msg("All uploads completed 🚀")
}
