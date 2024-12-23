package loader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"
)

type Chunk struct {
	Index  int
	Buffer *bytes.Buffer
}

func calcChunkSize(totalSize, workers int) int {
	v := float64(totalSize) / float64(workers)
	return int(math.Ceil(v))
}

func downloadChunk(ctx context.Context, index, size, workers int, accessToken, zoneURL string, chunks chan *Chunk) {
	client := http.Client{Timeout: time.Second * 120}
	buffer := &bytes.Buffer{}
	chunk := &Chunk{Index: index, Buffer: buffer}
	startBytes := index * size
	reqRange := fmt.Sprintf("bytes=%d-%d", startBytes, startBytes+size-1)
	if workers-1 == index {
		reqRange = fmt.Sprintf("bytes=%d-", startBytes)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", zoneURL, nil)
	if err != nil {
		return
	}
	req.Header.Add("Range", reqRange)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	io.Copy(chunk.Buffer, resp.Body)
	chunks <- chunk
}

func download(ctx context.Context, accessToken, zoneURL string, numWorkers int, fileChunks chan *Chunk) (io.Reader, error) {
	client := http.Client{Timeout: time.Second * 120}
	req, err := http.NewRequestWithContext(ctx, "HEAD", zoneURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		return nil, fmt.Errorf("could not get Content-Length header")
	}
	fileSize, err := strconv.Atoi(contentLength)
	if err != nil {
		return nil, err
	}
	chunkSize := calcChunkSize(fileSize, numWorkers)
	for i := range numWorkers {
		go downloadChunk(
			ctx, i, chunkSize, numWorkers, accessToken, zoneURL, fileChunks)
	}
	chunkCount := 0
	chunkBuffers := make([]io.Reader, numWorkers)
	for chunk := range fileChunks {
		chunkBuffers[chunk.Index] = chunk.Buffer
		chunkCount++
		if chunkCount == numWorkers {
			break
		}
	}
	return io.MultiReader(chunkBuffers[:]...), nil
}
