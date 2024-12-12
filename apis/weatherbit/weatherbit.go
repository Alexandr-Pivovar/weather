package weatherbit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"

	"weather/manager"
)

func New(config map[string]interface{}) *weatherApi {
	apiKey := config["api.weatherbit.io"].(map[string]interface{})["apiKey"].(string)
	return &weatherApi{
		apiKey: apiKey,
	}
}

type weatherApi struct {
	apiKey string
}

func (w weatherApi) Get(ctx context.Context, location manager.Location) (manager.Info, error) {
	resultChannel := make(chan *info, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		close(resultChannel)
	}()

	go func() {
		params := map[string]string{
			"key":        "d5ea8363334841babd9f3880490a3a8d",
			"start_date": fmt.Sprintf("%s:00", location.Time.Format("2006-1-02")),
			"end_date":   location.Time.Format("2006-1-02:15"),
		}

		if location.Latitude != "" && location.Longitude != "" {
			params["lat"] = location.Latitude
			params["lon"] = location.Longitude
		} else {
			params["city"] = location.City
		}

		historyHourly, err := processRequest(ctx, "https://api.weatherbit.io/v2.0/history/hourly", params)
		if err != nil {
			historyHourly.err = err
		}

		select {
		case <-ctx.Done():
			return
		default:
			resultChannel <- historyHourly
		}
	}()

	go func() {
		params := map[string]string{
			"key":   "d5ea8363334841babd9f3880490a3a8d",
			"hours": "24",
		}

		if location.Latitude != "" && location.Longitude != "" {
			params["lat"] = location.Latitude
			params["lon"] = location.Longitude
		} else {
			params["city"] = location.City
		}

		forecastHourly, err := processRequest(ctx, "https://api.weatherbit.io/v2.0/forecast/hourly", params)
		if err != nil {
			forecastHourly.err = err
		}

		select {
		case <-ctx.Done():
			return
		default:
			resultChannel <- forecastHourly
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

				info.Provider = "api.weatherbit.io"
				info.Location.Country = location.Country
				return info.Info, nil
			} else {
				interrupt = true
			}
		}
	}
}

type info struct {
	manager.Info
	err error
}

func (i *info) unmarshal(data []byte) error {
	type result struct {
		CityName string `json:"city_name"`
		Data     []struct {
			Temp float64 `json:"temp"`
			Rh   int64   `json:"rh"`
			Ts   int64   `json:"ts"`
		} `json:"data"`
	}

	var r result

	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	for _, data := range r.Data {
		i.Temperature = append(i.Temperature, manager.Temperature{
			TempC:     data.Temp,
			Humidity:  float64(data.Rh),
			Timestamp: data.Ts,
		})
	}
	i.Location.City = r.CityName

	return nil
}

func processRequest(ctx context.Context, path string, params map[string]string) (*info, error) {
	info := &info{}
	request := resty.New().R().SetContext(ctx)
	request.SetQueryParams(params)

	response, err := request.Get(path)
	if err != nil {
		return info, err
	}

	if response.StatusCode() != 200 {
		buf := &bytes.Buffer{}

		err = json.Indent(buf, response.Body(), "", "  ")
		if err != nil {
			return info, err
		}

		return info, fmt.Errorf("status code: %d\n%s", response.StatusCode(), buf.String())
	}

	return info, info.unmarshal(response.Body())
}
