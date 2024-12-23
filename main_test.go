package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/pogzyb/czdsdump/auth"
	"github.com/pogzyb/czdsdump/download/loader"
)

// Just a basic test to see memory usage:
// ICANN_USERNAME="" ICANN_PASSWORD="" && go test -run TestDownloadZone -memprofile mem && go tool pprof -http : mem

var (
	username = os.Getenv("ICANN_USERNAME")
	password = os.Getenv("ICANN_PASSWORD")
)

func TestDownloadZone(t *testing.T) {
	ctx := context.Background()
	zone := "art"
	zoneURL := fmt.Sprintf("https://czds-download-api.icann.org/czds/downloads/%s.zone", zone)
	accessToken, err := auth.GetAccessToken(ctx, username, password)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	fileLoader := loader.NewFileLoader(fmt.Sprintf("%s.txt.gz", zone), zoneURL, 2)
	err = fileLoader.DownloadZone(ctx, accessToken)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}
