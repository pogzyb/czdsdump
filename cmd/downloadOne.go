package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pogzyb/czdsdump/auth"
	"github.com/pogzyb/czdsdump/download"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	downloadOneCmd.Flags().StringVarP(
		&zone, "zone", "z", "", "The zone database (e.g. 'com' or 'net').")
	downloadOneCmd.MarkFlagRequired("zone")
	downloadCmd.AddCommand(downloadOneCmd)
}

var (
	zone string

	downloadOneCmd = &cobra.Command{
		Use:   "one",
		Short: "Downloads a single zone from the Centralized Zone Database Service.",
		Long: `Downloads a single zone from ICANN's Centralized Zone Database Service to AWS S3 or a Local Directory.
Learn More: https://www.icann.org/resources/pages/czds-2014-03-03-en`,
		Run: func(cmd *cobra.Command, args []string) {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			username = checkEnv("ICANN_USERNAME", username)
			password = checkEnv("ICANN_PASSWORD", password)
			createDir(outputDir)
			DownloadOne(username, password, outputDir, zone, workers)
		},
	}
)

func DownloadOne(username, password, outputDir, zone string, workers int) {
	// handle termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	// authentication with ICANN
	accessToken, err := auth.GetAccessToken(ctx, username, password)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("could not get access token: %v", err))
	}
	done := make(chan struct{}, 1)
	go func() {
		defer func() { done <- struct{}{} }()
		zoneURL := fmt.Sprintf("https://czds-download-api.icann.org/czds/downloads/%s.zone", zone)
		log.Info().Msg(fmt.Sprintf("Downloading %s", zoneURL))
		if ctx.Err() != nil {
			return
		}
		outputFile, err := download.GetOutputFile(outputDir, zone)
		if err != nil {
			log.Fatal().Msg(fmt.Sprintf("could not prepare output file: %s err: %v", outputDir, err))
		}
		// init the loader
		loader, err := download.NewLoader(outputFile, zoneURL, workers)
		if err != nil {
			log.Debug().Msg(fmt.Sprintf("could not get loader: %v", err))
			return
		}
		// download and save
		err = loader.DownloadZone(ctx, accessToken)
		if err != nil {
			log.Debug().Msg(fmt.Sprintf("could not download: %s: %v", zoneURL, err))
			return
		}
		log.Info().Msg(fmt.Sprintf("Saved %s", outputFile))
	}()
	for {
		// Wait for completion
		select {
		case <-sigs:
			log.Info().Msg("Received Termination.")
			cancel()
			return
		case <-done:
			log.Info().Msg("Done.")
			close(done)
			cancel()
			return
		}
	}
}
