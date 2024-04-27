package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pogzyb/czdsdump/download/loader"
	"github.com/rs/zerolog/log"
)

var (
	BaseURL        string
	DefaultTimeout int
)

func init() {
	BaseURL = os.Getenv("ICANN_CZDS_BASE_URL")
	if BaseURL == "" {
		BaseURL = "https://czds-api.icann.org"
	}
}

type Loader interface {
	Download(ctx context.Context, accessToken string) (io.Reader, error)
	DownloadZone(ctx context.Context, accessToken string) error
	Save(ctx context.Context, r io.Reader) error
}

func NewLoader(outputFile, zoneURL string, numWorkers int) (Loader, error) {
	if strings.HasPrefix(outputFile, "s3://") || strings.HasPrefix(outputFile, "S3://") {
		log.Debug().Msg("Using S3 Loader!")
		return loader.NewS3Loader(outputFile, zoneURL, numWorkers)
	} else {
		return loader.NewFileLoader(outputFile, zoneURL, numWorkers), nil
	}
}

func GetTLDFromURL(zoneURL string) string {
	splits := strings.Split(zoneURL, "/")
	return splits[len(splits)-1]
}

func GetZoneURLs(ctx context.Context, accessToken string) ([]string, error) {
	client := http.Client{Timeout: time.Second * 30}
	url := BaseURL + "/czds/downloads/links"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []string{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := client.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()
	zones := []string{}
	raw, _ := io.ReadAll(resp.Body)
	if err = json.Unmarshal(raw, &zones); err != nil {
		return []string{}, err
	}
	return zones, nil
}
