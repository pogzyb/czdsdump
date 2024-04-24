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

	"github.com/pogzyb/czdsdump/download/file"
	"github.com/pogzyb/czdsdump/download/s3"
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
	DefaultTimeout = 30
}

type Dumper interface {
	Init(ctx context.Context, path string, zoneLink string) error
	Copy(ctx context.Context, r io.Reader) error
}

func GetDumper(ctx context.Context, path, zoneLink string) (Dumper, error) {
	var d Dumper
	if strings.HasPrefix(path, "s3://") || strings.HasPrefix(path, "S3://") {
		d = new(s3.DumperS3)
	} else {
		d = new(file.DumperFile)
	}
	err := d.Init(ctx, path, zoneLink)
	return d, err
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: time.Second * time.Duration(DefaultTimeout)}
}

func GetZoneLinks(ctx context.Context, accessToken string) ([]string, error) {
	client := newHTTPClient()
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

func DumpZone(ctx context.Context, dumper Dumper, accessToken, zoneLink string) error {
	client := newHTTPClient()
	req, err := http.NewRequestWithContext(ctx, "GET", zoneLink, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = dumper.Copy(ctx, resp.Body)
	return err
}
