package loader

import (
	"context"
	"io"
	"math"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog/log"
)

var (
	clientS3         *s3.Client
	overridePartSize = []string{"com.zone", "org.zone", "net.zone", "top.zone"}
)

func initAWS() {
	conf, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal().Msgf("could not load aws configs: %v", err)
	}
	// Verify credentials
	clientSTS := sts.NewFromConfig(conf)
	_, err = clientSTS.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatal().Msgf("aws credentials error: %v", err)
	}
	// Create s3 client
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
	Filename   string
	NumWorkers int
	Uploader   *manager.Uploader
	ZoneURL    string
	chunks     chan *FileChunk
}

func NewS3Loader(outputFile, zoneURL string, numWorkers int) (*S3Loader, error) {
	// Initialize the s3 client
	if clientS3 == nil {
		initAWS()
	}
	// Create an uploader with the client and custom options
	uploader := manager.NewUploader(clientS3, func(u *manager.Uploader) {
		if needsCustomPartSize(zoneURL) {
			u.PartSize = 10 * 1024 * 1024 // 10MB per part
		}
	})
	u, err := url.Parse(outputFile)
	if err != nil {
		return nil, err
	}
	return &S3Loader{
		Bucket:     u.Host,
		Key:        strings.TrimLeft(u.Path, "/"),
		NumWorkers: numWorkers,
		ZoneURL:    zoneURL,
		Uploader:   uploader,
		chunks:     make(chan *FileChunk, numWorkers),
	}, nil
}

func (sl S3Loader) DownloadZone(ctx context.Context, accessToken string) error {
	var wg sync.WaitGroup
	// Fetch the file size
	fs, err := getFileSize(ctx, sl.ZoneURL, accessToken)
	if err != nil {
		return err
	}
	// Create a temporary file
	f, err := os.CreateTemp("", "zonefile*")
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove(f.Name())
	// Start worker pool
	for range sl.NumWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range sl.chunks {
				err := downloadAndWriteChunk(ctx, sl.ZoneURL, accessToken, chunk.Start, chunk.End, f)
				if err != nil {
					log.Error().Msgf("could not download and write: %v", err)
				}
				log.Debug().Msgf("Finished start=%d end=%d zone=%s", chunk.Start, chunk.End, sl.ZoneURL)
			}
		}()
	}
	// Send chunks to the worker pool
	numChunks := int(max(math.Ceil(float64(fs/int(defaultChunkSize))), 1))
	for i := range numChunks {
		start := i * int(defaultChunkSize)
		if i > 0 {
			start += 1
		}
		end := min(start+int(defaultChunkSize), fs)
		sl.chunks <- &FileChunk{Start: int64(start), End: int64(end), File: f}
	}
	// Close worker pool
	close(sl.chunks)
	wg.Wait()
	// Use the S3 Uploader
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = sl.Uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sl.Bucket),
		Key:    aws.String(sl.Key),
		Body:   f,
	})
	return err
}
