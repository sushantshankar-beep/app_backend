package utils

import(
	"encoding/json"
	"fmt"

	"net/http"
	"time"
	"strings"
)

type OSMResponse struct {
	DisplayName string `json:"display_name"`
	Address     struct {
		City          string `json:"city"`
		Town          string `json:"town"`
		Village       string `json:"village"`
		County        string `json:"county"`
		State         string `json:"state"`
		StateDistrict string `json:"state_district"`
		Region        string `json:"region"`
		Country       string `json:"country"`
	} `json:"address"`
}


func ReverseGeocode(lat, lon float64) (string, string, error) {

	url := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?lat=%f&lon=%f&format=json&addressdetails=1",
		lat, lon,
	)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("User-Agent", "vahanwire-app")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("OSM API failed: %s", resp.Status)
	}

	var result OSMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	// CITY extraction
	city := result.Address.City
	if city == "" {
		city = result.Address.Town
	}
	if city == "" {
		city = result.Address.Village
	}
	if city == "" {
		city = result.Address.County
	}

	// STATE extraction
	state := result.Address.State
	if state == "" {
		state = result.Address.StateDistrict
	}
	if state == "" {
		state = result.Address.Region
	}
	if state == "" && result.DisplayName != "" {
		parts := strings.Split(result.DisplayName, ",")
		if len(parts) >= 3 {
			// Second last element usually state
			state = strings.TrimSpace(parts[len(parts)-3])
		}
	}

	fmt.Println("Extracted City:", city)
	fmt.Println("Extracted State:", state)

	return city, state, nil
}
