// av.go
// Copyright(c) Matt Pharr 2024
// SPDX: MIT-only

package main

type Point2LL [2]float32

type Airport struct {
	Name      string
	Elevation int
	Location  Point2LL
	Runways   []Runway
}

type AltitudeRestriction struct {
	// We treat 0 as "unset", which works naturally for the bottom but
	// requires occasional care at the top.
	Range [2]float32
}

type Navaid struct {
	Type     string
	Name     string
	Location Point2LL
}

type Fix struct {
	Location Point2LL
}

type Runway struct {
	Id        string
	Heading   float32
	Threshold Point2LL
	Elevation int
}

type AirwayLevel int

const (
	AirwayLevelAll = iota
	AirwayLevelLow
	AirwayLevelHigh
)

type AirwayDirection int

const (
	AirwayDirectionAny = iota
	AirwayDirectionForward
	AirwayDirectionBackward
)

type AirwayFix struct {
	Fix       string
	Level     AirwayLevel
	Direction AirwayDirection
}

type Airway struct {
	Fixes []AirwayFix
}

type Waypoint struct {
	Fix                 string
	Location            Point2LL
	AltitudeRestriction *AltitudeRestriction
	Speed               int
	Heading             int
	FlyOver             bool
	Delete              bool
	IAF, IF, FAF        bool
}
