package main

import (
	"github.com/pogzyb/czdsdump/cmd"
	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

func main() { cmd.Execute() }
