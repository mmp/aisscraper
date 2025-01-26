// arinc424.go
// Copyright(c) Matt Pharr 2024
// SPDX: MIT-only

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const ARINC424LineLength = 134 // 132 chars + \r + \n

func empty(s []byte) bool {
	for _, b := range s {
		if b != ' ' {
			return false
		}
	}
	return true
}

func parseInt(s []byte) int {
	if v, err := strconv.Atoi(string(s)); err != nil {
		panic(err)
	} else {
		return v
	}
}

func ParseARINC424(contents []byte) (map[string]Airport, map[string]Navaid, map[string]Fix, map[string][]Airway) {
	airports := make(map[string]Airport)
	navaids := make(map[string]Navaid)
	fixes := make(map[string]Fix)
	airways := make(map[string][]Airway)
	airwayWIP := make(map[string]AirwayFix)

	parseLLDigits := func(d, m, s []byte) float32 {
		deg, err := strconv.Atoi(string(d))
		if err != nil {
			panic(err)
		}
		min, err := strconv.Atoi(string(m))
		if err != nil {
			panic(err)
		}
		sec, err := strconv.Atoi(string(s))
		if err != nil {
			panic(err)
		}
		return float32(deg) + float32(min)/60 + float32(sec)/100/3600
	}
	parseLatLong := func(lat, long []byte) Point2LL {
		var p Point2LL

		p[1] = parseLLDigits(lat[1:3], lat[3:5], lat[5:])
		p[0] = parseLLDigits(long[1:4], long[4:6], long[6:])

		if lat[0] == 'S' {
			p[1] = -p[1]
		}
		if long[0] == 'W' {
			p[0] = -p[0]
		}
		return p
	}

	br := bufio.NewReader(bytes.NewReader(contents))
	var lines [][]byte

	getline := func() []byte {
		if n := len(lines); n > 0 {
			l := lines[n-1]
			lines = lines[:n-1]
			return l
		}

		b, err := br.ReadBytes('\n')
		if err == io.EOF {
			return nil
		}

		if len(b) != ARINC424LineLength {
			panic(fmt.Sprintf("unexpected line length: %d", len(b)))
		}
		return b
	}

	for {
		line := getline()
		if line == nil {
			break
		}

		recordType := line[0]
		if recordType != 'S' { // not a standard field
			continue
		}

		sectionCode := line[4]
		switch sectionCode {
		case 'D':
			subsectionCode := line[6]
			if subsectionCode == ' ' /* VOR */ || subsectionCode == 'B' /* NDB */ {
				id := strings.TrimSpace(string(line[13:17]))
				if len(id) < 3 {
					break
				}

				name := strings.TrimSpace(string(line[93:123]))
				if !empty(line[32:51]) {
					navaids[id] = Navaid{
						Type:     Select(subsectionCode == ' ', "VOR", "NDB"),
						Name:     name,
						Location: parseLatLong(line[32:41], line[41:51]),
					}
				} else {
					navaids[id] = Navaid{
						Type:     "DME",
						Name:     name,
						Location: parseLatLong(line[55:64], line[64:74]),
					}
				}
			}

		case 'E':
			subsection := line[5]
			switch subsection {
			case 'A': // enroute waypoint
				id := strings.TrimSpace(string(line[13:18]))
				fixes[id] = Fix{
					Location: parseLatLong(line[32:41], line[41:51]),
				}

			case 'R': // enroute airway
				route := strings.TrimSpace(string(line[13:18]))
				seq := string(line[25:29])

				level := func() AirwayLevel {
					switch line[45] {
					case 'B', ' ':
						return AirwayLevelAll
					case 'H':
						return AirwayLevelHigh
					case 'L':
						return AirwayLevelLow
					default:
						panic("unexpected airway level: " + string(line[45]))
					}
				}()
				direction := func() AirwayDirection {
					switch line[46] {
					case 'F':
						return AirwayDirectionForward
					case 'B':
						return AirwayDirectionBackward
					case ' ':
						return AirwayDirectionAny
					default:
						panic("unexpected airway direction")
					}
				}()

				fix := AirwayFix{
					Fix:       strings.TrimSpace(string(line[29:34])),
					Level:     level,
					Direction: direction,
				}
				airwayWIP[seq] = fix

				if line[40] == 'E' { // description code "end of airway"
					a := Airway{}
					for _, seq := range SortedMapKeys(airwayWIP) { // order by sequence number, just in case
						a.Fixes = append(a.Fixes, airwayWIP[seq])
					}
					airways[route] = append(airways[route], a)
					clear(airwayWIP)
				}
			}
			// TODO: holding patterns, etc...

		case 'H': // Heliports
			subsection := line[12]
			switch subsection {
			case 'C': // waypoint record
				id := string(line[13:18])
				location := parseLatLong(line[32:41], line[41:51])
				if _, ok := fixes[id]; ok {
					fmt.Printf("%s: repeats\n", id)
				}
				fixes[id] = Fix{Location: location}
			}

		case 'P': // Airports
			icao := string(line[6:10])
			subsection := line[12]
			switch subsection {
			case 'A': // primary airport records 4.1.7
				location := parseLatLong(line[32:41], line[41:51])
				elevation := parseInt(line[56:61])

				airports[icao] = Airport{
					Name:      strings.TrimSpace(string(line[93 : 92+30])),
					Elevation: elevation,
					Location:  location,
				}

			case 'C': // waypoint record 4.1.4
				id := string(line[13:18])
				location := parseLatLong(line[32:41], line[41:51])
				//if _, ok := fixes[id]; ok {
				// fmt.Printf("%s: repeats\n", id)
				//}
				fixes[id] = Fix{Location: location}

			case 'D': // SID 4.1.9

			case 'E': // STAR 4.1.9

			case 'F': // Approach 4.1.9

			case 'G': // runway records 4.1.10
				continuation := line[21]
				if continuation != '0' && continuation != '1' {
					continue
				}
				if string(line[27:31]) == "    " {
					// No heading available. This happens for e.g. seaports.
					continue
				}

				rwy := string(line[13:18])
				rwy = strings.TrimPrefix(rwy, "RW")
				rwy = strings.TrimPrefix(rwy, "0")
				rwy = strings.TrimSpace(rwy)

				ap := airports[icao]
				ap.Runways = append(ap.Runways, Runway{
					Id:        rwy,
					Heading:   float32(parseInt(line[27:31])) / 10,
					Threshold: parseLatLong(line[32:41], line[41:51]),
					Elevation: parseInt(line[66:71]),
				})
				airports[icao] = ap
			}
		}

	}

	return airports, navaids, fixes, airways
}
