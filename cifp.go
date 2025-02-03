// cifp.go
// Copyright(c) Matt Pharr 2024
// SPDX: MIT-only

package main

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"net/http"

	"golang.org/x/net/html"
)

func DownloadCIFP() []byte {
	zipURL := GetCIFPZipURL()
	if zipURL == "" {
		log.Fatal("Unable to find URL for CIFP ZIP file")
	}
	log.Printf("CIFP is at %s", zipURL)

	resp, err := http.Get(zipURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var cifpZipBytes []byte
	if cifpZipBytes, err = io.ReadAll(resp.Body); err != nil {
		log.Fatal(err)
	}
	log.Printf("Received %d bytes", len(cifpZipBytes))

	cifpZip, err := zip.NewReader(bytes.NewReader(cifpZipBytes), int64(len(cifpZipBytes)))
	if err != nil {
		log.Fatal(err)
	}

	const cifpFilename = "FAACIFP18"
	var cifpFile *zip.File
	for _, f := range cifpZip.File {
		log.Printf("zip entry: %s (%d bytes)\n", f.Name, f.UncompressedSize64)
		if f.Name == cifpFilename {
			cifpFile = f
		}
	}
	if cifpFile == nil {
		log.Fatalf("Didn't find %q in CIFP zip file", cifpFilename)
	}

	r, err := cifpFile.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("CIFP is %d bytes after decompression", len(b))

	return b
}

// Scrape the FAA CIFP webpage to get the URL to the zip file with the latest CIFP.
func GetCIFPZipURL() string {
	url := "https://www.faa.gov/air_traffic/flight_info/aeronav/digital_products/cifp/download/"
	log.Print("Scraping CIFP page at " + url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	zipURL := ""

	var parse func(*html.Node)
	parse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "cfoutput" {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && child.Data == "a" {
					for _, attr := range child.Attr {
						if attr.Key == "href" && zipURL == "" {
							zipURL = attr.Val
						}
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			parse(child)
		}
	}

	for child := doc.FirstChild; child != nil; child = child.NextSibling {
		parse(child)
	}

	return zipURL
}
