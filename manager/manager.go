package manager

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

func New() *weather {
	return &weather{}
}

type weather struct {
	apis      []Weather
	geocoding Geocoding
}

func (w *weather) Get(ctx context.Context, location Location) (Info, error) {
	if len(w.apis) == 0 {
		return Info{}, fmt.Errorf("apis not found")
	}

	type result struct {
		Info Info
		err  error
	}

	var (
		info Info
		err  error
	)
	if w.geocoding != nil {
		location, err = w.geocoding.Get(ctx, location)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return Info{}, err
		}
	}

	resultChannel := make(chan result)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, api := range w.apis {
		go func(api Weather) {
			info, err = api.Get(ctx, location)
			resultChannel <- result{Info: info, err: err}
		}(api)
	}

	select {
	case res := <-resultChannel:
		return res.Info, res.err
	}
}

func (w *weather) RegisterAPI(apis ...Weather) {
	w.apis = append(w.apis, apis...)
}

func (w *weather) SetGeocoding(geocoding Geocoding) {
	w.geocoding = geocoding
}
