package loader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type FileChunk struct {
	Start int64
	End   int64
	File  *os.File
}

var defaultChunkSize int64 = 5e6 // 5MB

func downloadAndWriteChunk(ctx context.Context, url, token string, start, end int64, file *os.File) error {
	client := http.Client{Timeout: time.Second * 120}
	reqRange := fmt.Sprintf("bytes=%d-%d", start, end)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Range", reqRange)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	var resp *http.Response
	do := func() error {
		resp, err = client.Do(req)
		return err
	}
	backOffCtx := backoff.WithContext(backoff.NewConstantBackOff(time.Minute*1), ctx)
	retry := backoff.WithMaxRetries(backOffCtx, 2)
	if err := backoff.Retry(do, retry); err != nil {
		return err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_, err = file.WriteAt(bodyBytes, start)
	return err
}

func getFileSize(ctx context.Context, zoneURL, token string) (int, error) {
	client := http.Client{Timeout: time.Second * 120}
	req, err := http.NewRequestWithContext(ctx, "HEAD", zoneURL, nil)
	if err != nil {
		return -1, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		return -1, fmt.Errorf("could not get Content-Length header")
	}
	return strconv.Atoi(contentLength)
}
