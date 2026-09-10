package loader

import (
	"context"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

type FileLoader struct {
	Filename   string
	NumWorkers int
	ZoneURL    string
	chunks     chan *FileChunk
}

func NewFileLoader(outputFile, zoneURL string, numWorkers int) FileLoader {
	return FileLoader{
		Filename:   outputFile,
		NumWorkers: numWorkers,
		ZoneURL:    zoneURL,
		chunks:     make(chan *FileChunk, numWorkers),
	}
}

func (fl FileLoader) DownloadZone(ctx context.Context, accessToken string) error {
	var wg sync.WaitGroup
	// Fetch the file size
	fs, err := getFileSize(ctx, fl.ZoneURL, accessToken)
	if err != nil {
		return err
	}
	// Open the output file
	f, err := os.Create(fl.Filename)
	if err != nil {
		return err
	}
	// Start worker pool
	for range fl.NumWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range fl.chunks {
				err := downloadAndWriteChunk(ctx, fl.ZoneURL, accessToken, chunk.Start, chunk.End, f)
				if err != nil {
					log.Error().Msgf("could not download and write: %v", err)
				}
				log.Debug().Msgf("Finished start=%d end=%d zone=%s", chunk.Start, chunk.End, fl.ZoneURL)
			}
		}()
	}
	// Send chunks to the worker pool
	for start := int64(0); start < fs; start += defaultChunkSize {
		end := min(start+defaultChunkSize-1, fs-1)
		fl.chunks <- &FileChunk{Start: start, End: end, File: f}
	}
	// Close worker pool
	close(fl.chunks)
	wg.Wait()
	return nil
}
