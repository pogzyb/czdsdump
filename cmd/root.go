package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	username string
	password string
	verbose  bool

	rootCmd = &cobra.Command{
		Use:   "czdsdump",
		Short: "...",
		Long:  `Utility for exporting data from the Central Zone Database Service.`,
		Run:   func(cmd *cobra.Command, args []string) {},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&username, "username", "u", "", "ICANN account username. If empty, value is read from env var: ICANN_USERNAME).")
	rootCmd.PersistentFlags().StringVarP(
		&password, "password", "p", "", "ICANN account password. If empty, value is read from env var: ICANN_PASSWORD).")
	rootCmd.PersistentFlags().BoolVarP(
		&verbose, "verbose", "v", false, "Enable verbose debug logging.")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
