package cli

import (
	"time"

	"github.com/spf13/cobra"

	"weather/manager"
)

func New(weather manager.Weather) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "weather",
		Args:  cobra.ExactArgs(2),
		Short: "CLI application for getting information of weather",
		RunE: func(cmd *cobra.Command, args []string) error {
			country := args[0]
			city := args[1]

			info, err := weather.Get(cmd.Context(), manager.Location{Country: country, City: city, Time: time.Now()})
			if err != nil {
				return err
			}

			cmd.Printf("PROVIDER\t %s\n", info.Provider)
			cmd.Printf("LOCATION\t %s %s %s\n",
				info.Location.Country,
				info.Location.Region,
				info.Location.City,
			)
			cmd.Printf("TIME\t\t")
			for _, temperature := range info.Temperature {
				cmd.Printf("%3s  ", time.Unix(temperature.Timestamp, 0).UTC().Format("15"))
			}

			cmd.Printf("\nTEMP\t\t")
			for _, temperature := range info.Temperature {
				cmd.Printf("%3.0f  ", temperature.TempC)
			}
			cmd.Printf("\nHUMIDITY\t")
			for _, temperature := range info.Temperature {
				cmd.Printf("%3.0f  ", temperature.Humidity)
			}
			cmd.Printf("\n")

			return nil
		},
	}

	return cmd, nil
}
