package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pogzyb/czdsdump/auth"
	"github.com/pogzyb/czdsdump/download"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	dumpCmd.Flags().StringVarP(
		&username, "username", "u", "", "ICANN account username. If empty, value is read from env var: CZDS_ICANN_USERNAME).")
	dumpCmd.Flags().StringVarP(
		&password, "password", "p", "", "ICANN account password. If empty, value is read from env var: CZDS_ICANN_PASSWORD).")
	dumpCmd.Flags().StringVarP(
		&outputDir, "output", "o", "./czds", "Where to write files (e.g. '/home/joe/czds/' or 's3://bucket/2024-01-01/').")
	dumpCmd.Flags().IntVarP(
		&workers, "workers", "w", 20, "Number of concurrent download workers.")
	dumpCmd.Flags().BoolVarP(
		&verbose, "verbose", "v", false, "Enable verbose debug logging.")

	rootCmd.AddCommand(dumpCmd)
}

var (
	username  string
	password  string
	outputDir string
	workers   int
	verbose   bool

	dumpCmd = &cobra.Command{
		Use:   "all",
		Short: "Export ALL zone data from the Centralized Zone Database Service.",
		Long: `Export ALL zone data from ICANN's Centralized Zone Database Service to AWS S3 or a Local Directory.
Learn More: https://www.icann.org/resources/pages/czds-2014-03-03-en`,
		Run: func(cmd *cobra.Command, args []string) {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			if username == "" {
				if username = os.Getenv("CZDS_ICANN_USERNAME"); username == "" {
					log.Fatal().Msg("One of --username or CZDS_ICANN_USERNAME must be specified.")
				}
			}
			if password == "" {
				if password = os.Getenv("CZDS_ICANN_PASSWORD"); password == "" {
					log.Fatal().Msg("One of --password or CZDS_ICANN_PASSWORD must be specified.")
				}
			}
			Dump(username, password, outputDir, workers)
		},
	}

	wg sync.WaitGroup
)

func Dump(username, password, outputDir string, workers int) {
	// Handle termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	// Authentication with ICANN
	accessToken, err := auth.GetAccessToken(ctx, username, password)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("could not get access token: %v", err))
	}

	// Create worker pool of size `workers` that listens for zone links.
	zoneURLs := make(chan string, workers)
	for range workers {
		go func() {
			defer wg.Done()
			for zoneLink := range zoneURLs {
				if ctx.Err() != nil {
					return
				}
				dumper, err := download.GetDumper(ctx, outputDir, zoneLink)
				if err != nil {
					log.Debug().Msg(fmt.Sprintf("could not get file dumper: %v", err))
					return
				}
				err = download.DumpZone(ctx, dumper, accessToken, zoneLink)
				if err != nil {
					log.Debug().Msg(fmt.Sprintf("could not dump: %s: %v", zoneLink, err))
					return
				}
			}
		}()
		wg.Add(1)
	}

	// Submit the zone links to be downloaded to the workers
	done := make(chan struct{}, 1)
	go func() {
		defer func() { done <- struct{}{} }()
		zoneLinks, err := download.GetZoneLinks(ctx, accessToken)
		if err != nil {
			log.Debug().Msg(fmt.Sprintf("could not get zone links: %v", err))
			close(zoneURLs)

		} else {
			for _, zoneLink := range zoneLinks {
				zoneURLs <- zoneLink
			}
			// All zone links have been submitted, so now
			// wait for all workers to finish
			wg.Wait()
			close(zoneURLs)
		}
	}()

wait:
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
			break wait
		}
	}
}
