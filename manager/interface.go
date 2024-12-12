package manager

import (
	"context"
	"time"
)

type Weather interface {
	Get(ctx context.Context, location Location) (Info, error)
}

type Geocoding interface {
	Get(ctx context.Context, location Location) (Location, error)
}

type Info struct {
	Temperature []Temperature
	Location    Location
	Provider    string
}

type Temperature struct {
	Timestamp int64
	TempC     float64
	Humidity  float64
}

type Location struct {
	City      string
	Region    string
	Country   string
	Latitude  string
	Longitude string
	Time      time.Time
}
