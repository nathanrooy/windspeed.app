package weather

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
)

var windArrows = [...]string{"↓", "↙", "←", "↖", "↑", "↗", "→", "↘", "↓"}

type WeatherResponse struct {
	Current	struct {
		Clouds 		uint16
		Dew_point 	float32
		Feels_like 	float32
		Humidity 	uint8
		Pressure 	float32
		Temp 		float32
		Uvi			float32
		Wind_speed 	float32
		Wind_gust 	float32
		Wind_deg 	float64
	}
}

func CreateStamp(lat float64, lng float64, dt int64, units string) string {
	// Construct weather API request.
	var url string = "http://api.openweathermap.org/data/2.5/onecall/timemachine?"
	url += fmt.Sprintf("lat=%f&lon=%f", lat, lng)
	url += fmt.Sprintf("&dt=%d&units=%s", dt, units)
	url += fmt.Sprintf("&appid=%s", os.Getenv("WEATHER_API_KEY"))
	url += "&only_current={true}"

	// Call the weather API.
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Call to weather API failed...")
	}
	defer resp.Body.Close()

	// Parse API response.
	var wResp WeatherResponse
	err = json.NewDecoder(resp.Body).Decode(&wResp)

	// Construct weather stamp.
	var wStamp string = fmt.Sprintf("%0.1f°", wResp.Current.Temp)
	switch units {
	case "imperial":
		wStamp += "F"
	case "metric":
		wStamp += "C"
	}
	wStamp += fmt.Sprintf(", clouds: %d%%", wResp.Current.Clouds)
	wStamp += fmt.Sprintf(", humidity: %d%%", wResp.Current.Humidity)
	wStamp += fmt.Sprintf(", wind: %0.1f", wResp.Current.Wind_speed)
	if wResp.Current.Wind_gust > 0 {
		wStamp += fmt.Sprintf(" (%0.1f gust)", wResp.Current.Wind_gust)
	}
	switch units {
	case "imperial":
		wStamp += " mph "
	case "metric":
		wStamp += " km/h "
	}
	wStamp += windArrows[int(math.Round(wResp.Current.Wind_deg / 45))]
	return wStamp
}
