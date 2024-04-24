package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

var clientS3 *s3.Client

type DumperS3 struct {
	Uploader *manager.Uploader
	Bucket   string
	Key      string
}

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

func (ds *DumperS3) Init(ctx context.Context, path, zoneLink string) error {
	ds.Uploader = manager.NewUploader(clientS3)
	pathParsed, err := url.Parse(path)
	if err != nil {
		return err
	}
	zoneParsed, err := url.Parse(zoneLink)
	if err != nil {
		return err
	}
	split := strings.Split(zoneParsed.Path, "/")
	zoneName := split[len(split)-1]
	ds.Bucket = pathParsed.Host
	ds.Key = filepath.Join(pathParsed.Path, zoneName) + ".txt.gz"
	return nil
}

func (ds *DumperS3) Copy(ctx context.Context, r io.Reader) error {
	resp, err := ds.Uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(ds.Bucket),
		Key:    aws.String(ds.Key),
		Body:   r,
	})
	if err != nil {
		return err
	}
	log.Debug().Msg(fmt.Sprintf("Saved zone file: %s", *resp.Key))
	return nil
}
