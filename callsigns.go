// callsigns.go
// Copyright(c) Matt Pharr 2024
// SPDX: MIT-only

package main

import (
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type Callsign struct {
	Telephony string
	Airline   string
	Country   string
}

func ScrapeCallsigns() {
	url := "https://www.faa.gov/air_traffic/publications/atpubs/cnt_html/chap3_section_3.html"
	log.Printf("Fetching %s", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	callsigns := make(map[string]Callsign)

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var extract func(n *html.Node) string
	extract = func(n *html.Node) string {
		if n.Type == html.TextNode {
			return n.Data
		}
		var r string
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r += extract(c)
		}
		return r
	}

	var parse func(*html.Node)
	parse = func(node *html.Node) {
		var row []string
		if node.Type == html.ElementNode && node.Data == "tr" {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
					row = append(row, strings.TrimSpace(extract(child)))
				}
			}
			if len(row) > 0 && row[0] != "3Ltr" && row[3] != "" {
				if _, ok := callsigns[row[0]]; ok {
					log.Printf("Warning: %q is repeated\n", row[0])
				}
				callsigns[row[0]] = Callsign{Telephony: row[3], Airline: row[1], Country: row[2]}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			parse(child)
		}
	}

	for child := doc.FirstChild; child != nil; child = child.NextSibling {
		parse(child)
	}

	WriteJSON(callsigns, "callsigns.json")
}
