// main.go
// Copyright(c) Matt Pharr 2024
// SPDX: MIT-only

package main

import (
	"encoding/json"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"

	"golang.org/x/exp/constraints"
)

func main() {
	ScrapeCallsigns()

	cifp := DownloadCIFP()
	airports, navaids, fixes, airways := ParseARINC424(cifp)
	log.Printf("Got %d airports, %d navaids, %d fixes, %d airways from CIFP", len(airports), len(navaids), len(fixes), len(airways))

	WriteJSON(airports, "airports.json")
	WriteJSON(navaids, "navaids.json")
	WriteJSON(fixes, "fixes.json")
	WriteJSON(airways, "airways.json")
}

func WriteJSON[T any](data T, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(data); err != nil {
		log.Fatal(err)
	}

	log.Printf("Wrote %q", filename)
}

func FetchURL(url string) []byte {
	log.Printf("Fetching %s", url)

	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	var text []byte
	if text, err = io.ReadAll(response.Body); err != nil {
		log.Fatal(err)
	}

	log.Printf("Received %d bytes", len(text))

	return text
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
