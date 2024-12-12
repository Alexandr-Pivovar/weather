package geocoding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"

	"weather/manager"
)

func New(config map[string]interface{}) *geocoding {
	apiKey := config["geocoding"].(map[string]interface{})["apiKey"].(string)
	return &geocoding{
		apiKey: apiKey,
	}
}

type geocoding struct {
	apiKey string
}

func (g geocoding) Get(ctx context.Context, location manager.Location) (manager.Location, error) {
	params := map[string]string{
		"api_key": "67595f25254ad894267079ozb396c37",
		"q":       fmt.Sprintf("%s,%s", location.Country, location.City),
	}

	result, err := processRequest(ctx, "https://geocode.maps.co/search", params)
	if err != nil {
		return location, err
	}

	location.Latitude = result.Latitude
	location.Longitude = result.Longitude

	return location, nil
}

func processRequest(ctx context.Context, path string, params map[string]string) (manager.Location, error) {
	type responseStruct struct {
		Lat   string `json:"lat"`
		Lon   string `json:"lon"`
		Class string `json:"class"`
		Type  string `json:"type"`
	}

	request := resty.New().R().SetContext(ctx)
	request.SetQueryParams(params)

	response, err := request.Get(path)
	if err != nil {
		return manager.Location{}, err
	}

	if response.StatusCode() != 200 {
		buf := &bytes.Buffer{}

		err = json.Indent(buf, response.Body(), "", "  ")
		if err != nil {
			return manager.Location{}, err
		}

		return manager.Location{}, fmt.Errorf("status code: %d\n%s", response.StatusCode(), buf.String())
	}

	responseStr := make([]responseStruct, 0, 8)
	err = json.Unmarshal(response.Body(), &responseStr)
	if err != nil {
		return manager.Location{}, err
	}

	for i := range responseStr {
		if responseStr[i].Type == "city" && responseStr[i].Class == "place" ||
			responseStr[i].Type == "administrative" && responseStr[i].Class == "boundary" {
			return manager.Location{
				Longitude: responseStr[i].Lon,
				Latitude:  responseStr[i].Lat,
			}, nil
		}
	}

	return manager.Location{}, manager.ErrNotFound
}
