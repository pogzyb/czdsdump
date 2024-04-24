package file

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

type DumperFile struct {
	DstWriter io.Writer
	File      string
}

func (df *DumperFile) Init(ctx context.Context, path, zoneLink string) error {
	zoneParsed, err := url.Parse(zoneLink)
	if err != nil {
		return err
	}
	split := strings.Split(zoneParsed.Path, "/")
	zoneName := split[len(split)-1]
	file := filepath.Join(path, zoneName) + ".txt.gz"
	if _, err := os.Stat(file); os.IsNotExist(err) {
		os.MkdirAll(path, 0700)
	}
	writer, err := os.Create(file)
	df.File = file
	df.DstWriter = writer
	return err
}

func (df *DumperFile) Copy(ctx context.Context, r io.Reader) error {
	_, err := io.Copy(df.DstWriter, r)
	log.Debug().Msg(fmt.Sprintf("Saved zone file: %s", df.File))
	return err
}
