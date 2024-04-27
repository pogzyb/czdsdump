package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	downloadCmd.PersistentFlags().StringVarP(
		&outputDir, "output", "o", "./czds", "Where to write files (e.g. '/home/joe/czds/' or 's3://bucket/2024-01-01/').")
	downloadCmd.PersistentFlags().IntVarP(
		&workers, "workers", "w", 5, "Number of concurrent download workers.")

	rootCmd.AddCommand(downloadCmd)
}

var (
	outputDir string
	workers   int

	downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "...",
		Long:  `...`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("Use one of the download subcommands...")
		},
	}

	wg sync.WaitGroup
)

func checkEnv(k, v string) string {
	if v == "" {
		v = os.Getenv(k)
		if v == "" {
			log.Fatal().Msg(fmt.Sprintf("No value for icann username/password. Missing %s", k))
		}
	}
	return v
}

func createDir(dir string) {
	if strings.HasPrefix(dir, "s3://") || strings.HasPrefix(dir, "S3://") {
		return
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			log.Fatal().Msg(fmt.Sprintf("Could not create outputDir: %s", dir))
		}
	}
}
