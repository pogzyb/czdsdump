package loader

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog/log"
)

var (
	clientS3         *s3.Client
	overridePartSize = []string{"com.zone", "org.zone", "net.zone", "top.zone"}
)

func init() {
	conf, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("could not load aws configs: %v", err))
	}
	clientS3 = s3.NewFromConfig(conf, func(o *s3.Options) {
		if os.Getenv("AWS_ENDPOINT_URL") != "" {
			o.BaseEndpoint = aws.String(os.Getenv("AWS_ENDPOINT_URL"))
			o.UsePathStyle = true
		}
	})
}

func needsCustomPartSize(zoneLink string) bool {
	for _, zone := range overridePartSize {
		if strings.Contains(zoneLink, zone) {
			return true
		}
	}
	return false
}

type S3Loader struct {
	Bucket     string
	Key        string
	fileChunks chan *Chunk
	NumWorkers int
	Uploader   manager.Uploader
	ZoneURL    string
}

func NewS3Loader(outputFile, zoneURL string, numWorkers int) (*S3Loader, error) {
	uploader := manager.NewUploader(clientS3)
	if needsCustomPartSize(zoneURL) {
		uploader.PartSize = 10 * 1024 * 1024 // 10 Mib
	}
	pathParsed, err := url.Parse(outputFile)
	if err != nil {
		return nil, err
	}
	zoneParsed, err := url.Parse(zoneURL)
	if err != nil {
		return nil, err
	}
	split := strings.Split(zoneParsed.Path, "/")
	zoneName := split[len(split)-1]
	bucket := pathParsed.Host
	prefix := filepath.Join(pathParsed.Path, zoneName) + ".txt.gz"
	key := strings.TrimPrefix(prefix, "/")
	fc := make(chan *Chunk, numWorkers)
	return &S3Loader{
		Bucket:     bucket,
		Key:        key,
		fileChunks: fc,
		NumWorkers: numWorkers,
		ZoneURL:    zoneURL,
	}, nil
}

// Downloads the zone data concurrently and returns the resulting file as an `io.Reader`.
func (sl S3Loader) Download(ctx context.Context, accessToken string) (io.Reader, error) {
	return download(ctx, accessToken, sl.ZoneURL, sl.NumWorkers, sl.fileChunks)
}

// Simply combines the functionality of FileLoader's `Download` and `Save` functions.
func (sl S3Loader) DownloadZone(ctx context.Context, accessToken string) error {
	r, err := sl.Download(ctx, accessToken)
	if err != nil {
		return err
	}
	err = sl.Save(ctx, r)
	return err
}

// Saves the data in the `io.Reader` out to the FileLoader's File.
func (sl S3Loader) Save(ctx context.Context, r io.Reader) error {
	fn := func() error {
		_, err := sl.Uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(sl.Bucket),
			Key:    aws.String(sl.Key),
			Body:   r,
		})
		if err != nil {
			return err
		}
		return nil
	}
	boConst := backoff.NewConstantBackOff(time.Minute * 3)
	boContext := backoff.WithContext(boConst, ctx)
	boRetries := backoff.WithMaxRetries(boContext, 2)
	if err := backoff.Retry(fn, boRetries); err != nil {
		return err
	}
	return nil
}
