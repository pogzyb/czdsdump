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
	downloadCmd.AddCommand(downloadAllCmd)
}

var (
	downloadAllCmd = &cobra.Command{
		Use:   "all",
		Short: "Downloads ALL zone data from the Centralized Zone Database Service.",
		Long: `Downloads ALL zone data from ICANN's Centralized Zone Database Service to AWS S3 or a Local Directory.
Learn More: https://www.icann.org/resources/pages/czds-2014-03-03-en`,
		Run: func(cmd *cobra.Command, args []string) {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			username = checkEnv("ICANN_USERNAME", username)
			password = checkEnv("ICANN_PASSWORD", password)
			createDir(outputDir)
			DownloadAll(username, password, outputDir, workers)
		},
	}
)

func DownloadAll(username, password, outputDir string, workers int) {
	// Handle termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	// Authentication with ICANN
	accessToken, err := auth.GetAccessToken(ctx, username, password)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("could not get access token: %v", err))
	}
	// Channel size determines download concurrency
	zonesQueue := make(chan string, workers)
	for range workers {
		go func() {
			defer wg.Done()
			for zoneURL := range zonesQueue {
				if ctx.Err() != nil {
					// context cancelled; stop.
					return
				}
				// form the output filename
				tld := download.GetTLDFromURL(zoneURL)
				outputFile, err := download.GetOutputFile(outputDir, tld)
				if err != nil {
					log.Fatal().Msg(fmt.Sprintf("could not prepare output file: %s err: %v", outputDir, err))
				}
				// init the loader
				loader, err := download.NewLoader(outputFile, zoneURL, workers/2)
				if err != nil {
					log.Debug().Msg(fmt.Sprintf("could not get loader: %v", err))
					continue
				}
				log.Info().Msg(fmt.Sprintf("Downloading %s", zoneURL))
				// download and save
				err = loader.DownloadZone(ctx, accessToken)
				if err != nil {
					log.Debug().Msg(fmt.Sprintf("could not download: %s: %v", zoneURL, err))
					continue
				}
				log.Info().Msg(fmt.Sprintf("Saved %s", outputFile))
			}
		}()
		wg.Add(1)
	}
	// Submit the zone links to be downloaded to the workers
	done := make(chan struct{}, 1)
	go func() {
		defer func() { done <- struct{}{} }()
		zoneURLs, err := download.GetZoneURLs(ctx, accessToken)
		if err != nil {
			log.Debug().Msg(fmt.Sprintf("could not get zone links: %v", err))
			close(zonesQueue)

		} else {
			log.Info().Msg(fmt.Sprintf("Retrieving data from %d zones.", len(zoneURLs)))
			for _, zoneURL := range zoneURLs {
				zonesQueue <- zoneURL
			}
			// All zone links have been submitted;
			// close the channel and wait for workers to finish
			close(zonesQueue)
			wg.Wait()
		}
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
			cancel()
			return
		}
	}
}
