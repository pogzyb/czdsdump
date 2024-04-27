package loader

import (
	"context"
	"io"
	"os"
)

type FileLoader struct {
	File       string
	fileChunks chan *Chunk
	NumWorkers int
	ZoneURL    string
}

func NewFileLoader(outputFile, zoneURL string, numWorkers int) FileLoader {
	fc := make(chan *Chunk, numWorkers)
	return FileLoader{
		File:       outputFile,
		fileChunks: fc,
		NumWorkers: numWorkers,
		ZoneURL:    zoneURL,
	}
}

// Downloads the zone data concurrently and returns the resulting file as an `io.Reader`.
func (fl FileLoader) Download(ctx context.Context, accessToken string) (io.Reader, error) {
	return download(ctx, accessToken, fl.ZoneURL, fl.NumWorkers, fl.fileChunks)
}

// Simply combines the functionality of FileLoader's `Download` and `Save` functions.
func (fl FileLoader) DownloadZone(ctx context.Context, accessToken string) error {
	r, err := fl.Download(ctx, accessToken)
	if err != nil {
		return err
	}
	err = fl.Save(ctx, r)
	return err
}

// Saves the data in the `io.Reader` out to the FileLoader's File.
func (fl FileLoader) Save(ctx context.Context, r io.Reader) error {
	f, err := os.Create(fl.File)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	return err
}
