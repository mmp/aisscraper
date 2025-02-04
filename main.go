// main.go
// Copyright(c) Matt Pharr 2024
// SPDX: MIT-only

package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"maps"
	"os"
	"slices"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/exp/constraints"
)

func main() {
	runLocal := flag.Bool("runlocal", false, "run locally rather than in the AWS Lambda environment")
	flag.Parse()

	if *runLocal {
		handleRequest(context.TODO())
	} else {
		lambda.Start(handleRequest)
	}
}

func handleRequest(ctx context.Context) {
	callsigns := ScrapeCallsigns()
	StoreJSON(ctx, callsigns, "callsigns.json")

	cifp := DownloadCIFP()
	airports, navaids, fixes, airways := ParseARINC424(cifp)
	log.Printf("Got %d airports, %d navaids, %d fixes, %d airways from CIFP", len(airports), len(navaids), len(fixes), len(airways))

	StoreJSON(ctx, airports, "airports.json")
	StoreJSON(ctx, navaids, "navaids.json")
	StoreJSON(ctx, fixes, "fixes.json")
	StoreJSON(ctx, airways, "airways.json")
}

var s3Client *s3.Client

func init() {
	if cfg, err := config.LoadDefaultConfig(context.TODO()); err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	} else {
		s3Client = s3.NewFromConfig(cfg)
	}
}

func StoreJSON[T any](ctx context.Context, data T, filename string) {
	// This is arguably gratuitous versus just encoding to a []byte
	// since these things are just a few megabytes, but here we go.
	pr, pw := io.Pipe()
	go func() {
		if err := json.NewEncoder(pw).Encode(data); err != nil {
			pw.CloseWithError(err)
		} else {
			pw.Close()
		}
	}()

	// If S3BUCKET is set, write to that bucket, else save files locally.
	if bucket := os.Getenv("S3BUCKET"); bucket != "" {
		_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &filename,
			Body:   pr,
		})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Uploaded %q to S3 bucket %q", filename, bucket)
	} else {
		f, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		if n, err := io.Copy(f, pr); err != nil {
			log.Fatal(err)
		} else {
			log.Printf("Wrote %d bytes to %q", n, filename)
		}
	}
}

func Select[T any](sel bool, a, b T) T {
	if sel {
		return a
	} else {
		return b
	}
}

// SortedMapKeys returns the keys of the given map, sorted from low to high.
func SortedMapKeys[K constraints.Ordered, V any](m map[K]V) []K {
	return slices.Sorted(maps.Keys(m))
}
