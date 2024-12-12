package weatherapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"

	"weather/manager"
)

const apiName = "api.weatherapi.com"

func New(config map[string]interface{}) *weatherApi {
	apiKey := config[apiName].(map[string]interface{})["apiKey"].(string)
	return &weatherApi{
		apiKey: apiKey,
	}
}

type weatherApi struct {
	apiKey string
}

func (w weatherApi) Get(ctx context.Context, location manager.Location) (manager.Info, error) {
	resultChannel := make(chan info, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		close(resultChannel)
	}()

	go func() {
		params := url.Values{}

		params.Set("key", w.apiKey)
		params.Set("days", "2")

		if location.Longitude != "" && location.Latitude != "" {
			params.Add("q", fmt.Sprintf("%s,%s", location.Latitude, location.Longitude))
		} else {
			params.Add("q", location.City)
			params.Add("q", location.Country)
		}

		forecast, err := processRequest(ctx, "https://api.weatherapi.com/v1/forecast.json", params)
		if err != nil {
			forecast.err = err
		}

		select {
		case <-ctx.Done():
			return
		default:
			resultChannel <- forecast
		}
	}()

	go func() {
		params := url.Values{}

		params.Set("key", w.apiKey)
		params.Set("dt", location.Time.Format("2006-01-02"))
		params.Set("end_dt", location.Time.AddDate(0, 0, 1).Format("2006-01-02"))

		if location.Longitude != "" && location.Latitude != "" {
			params.Add("q", fmt.Sprintf("%s,%s", location.Latitude, location.Longitude))
		} else {
			params.Add("q", location.City)
			params.Add("q", location.Country)
		}

		history, err := processRequest(ctx, "https://api.weatherapi.com/v1/history.json", params)
		if err != nil {
			history.err = err
		}

		select {
		case <-ctx.Done():
			return
		default:
			history.Temperature = nil
			resultChannel <- history
		}
	}()

	result := map[int64]manager.Temperature{}

	var interrupt bool

	for {
		select {
		case <-ctx.Done():
			return manager.Info{}, ctx.Err()
		case info := <-resultChannel:
			if info.err != nil {
				return manager.Info{}, info.err
			}

			year, month, day := location.Time.Date()
			today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

			for _, temperature := range info.Temperature {
				t := time.Unix(temperature.Timestamp, 0).UTC()
				if t.Before(today) || t.After(today.Add(23*time.Hour)) {
					continue
				}
				result[temperature.Timestamp] = temperature
			}

			if interrupt {
				info.Temperature = make([]manager.Temperature, 0, len(result))

				for _, temperature := range result {
					info.Temperature = append(info.Temperature, temperature)
				}
				sort.Slice(info.Temperature, func(i, j int) bool {
					return info.Temperature[i].Timestamp < info.Temperature[j].Timestamp
				})

				info.Provider = apiName
				return info.Info, nil
			} else {
				interrupt = true
			}
		}
	}

}

func processRequest(ctx context.Context, path string, params url.Values) (info, error) {
	request := resty.New().R().SetContext(ctx)

	request.SetQueryParamsFromValues(params)

	response, err := request.Get(path)
	if err != nil {
		return info{}, err
	}

	if response.StatusCode() != 200 {
		buf := &bytes.Buffer{}

		err = json.Indent(buf, response.Body(), "", "  ")
		if err != nil {
			return info{}, err
		}

		return info{}, fmt.Errorf("resource code: %d\n%s", response.StatusCode(), buf.String())
	}

	info := info{}

	return info, info.unmarshal(response.Body())
}

type info struct {
	manager.Info
	err error
}

func (i *info) unmarshal(data []byte) error {
	type result struct {
		Location struct {
			Name    string `json:"name"`
			Region  string `json:"region"`
			Country string `json:"country"`
		} `json:"location"`
		Forecast struct {
			ForeCastday []struct {
				Hour []struct {
					TimeEpoch int64   `json:"time_epoch"`
					TempC     float64 `json:"temp_c"`
					Humidity  float64 `json:"humidity"`
				} `json:"hour"`
			} `json:"forecastday"`
		} `json:"forecast"`
	}

	var r result

	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	for _, foreCastday := range r.Forecast.ForeCastday {
		for _, hour := range foreCastday.Hour {
			i.Temperature = append(i.Temperature, manager.Temperature{
				Humidity:  hour.Humidity,
				TempC:     hour.TempC,
				Timestamp: hour.TimeEpoch,
			})
		}
	}
	i.Location.Country = r.Location.Country
	i.Location.Region = r.Location.Region
	i.Location.City = r.Location.Name

	return nil
}
