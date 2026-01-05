package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSource     = "coaxial"
	scanTimeout       = 100 * time.Millisecond
	httpClientTimeout = 5 * time.Second
)

// Response structures for KEF API
type GetDataResponse struct {
	String            string `json:"string_"`
	KefPhysicalSource string `json:"kefPhysicalSource"`
	I32               int    `json:"i32_"`
	BoolString        string `json:"bool_"` // API returns "True" or "False" as string
}

// GetBool converts the string boolean to an actual boolean
func (g *GetDataResponse) GetBool() bool {
	return g.BoolString == "True"
}

type SetDataValue struct {
	Type              string `json:"type,omitempty"`
	String            string `json:"string_,omitempty"`
	KefPhysicalSource string `json:"kefPhysicalSource,omitempty"`
	I32               int    `json:"i32_,omitempty"`
	BoolString        string `json:"bool_,omitempty"` // API returns "True" or "False" as string
}

type SetDataResponse struct {
	Value SetDataValue `json:"value"`
}

func main() {
	// Get base URL from environment or use default
	baseURL := os.Getenv("KEF_URL")
	if baseURL == "" {
		fmt.Fprintf(os.Stderr, "Error: KEF_URL environment variable is not set\n")
		fmt.Fprintf(os.Stderr, "Please set it with: export KEF_URL=http://0.0.0.0\n")
		fmt.Fprintf(os.Stderr, "Or run 'kef scan' to find your speakers\n")
		os.Exit(1)
	}

	// Parse command line arguments
	flag.Parse()
	args := flag.Args()

	command := "show"
	if len(args) > 0 {
		command = args[0]
	}

	// Handle command aliases
	switch command {
	case "start", "on":
		command = defaultSource
	case "stop", "off":
		command = "standby"
	}

	// Execute command
	if err := executeCommand(command, baseURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func executeCommand(command, baseURL string) error {
	validSources := []string{"standby", "coaxial", "wifi", "bluetooth", "tv", "optic", "analog"}

	switch command {
	case "show":
		return showStatus(baseURL)

	case "up", "down":
		return adjustVolume(command, baseURL)

	case "mute", "unmute":
		return setMute(command, baseURL)

	case "browse":
		return browse(baseURL)

	case "scan":
		return scanNetwork()

	case "nmap":
		fmt.Println("Note: Run 'nmap -sn 192.168.0.0/24' manually")
		return nil

	default:
		// Check if it's a volume number (0-100)
		if volume, err := strconv.Atoi(command); err == nil {
			return setVolume(volume, baseURL)
		}

		// Check if it's a valid source
		for _, source := range validSources {
			if command == source {
				return setSource(command, baseURL)
			}
		}

		return fmt.Errorf("unknown command: %s", command)
	}
}

func getData(baseURL, path string) ([]GetDataResponse, error) {
	client := &http.Client{Timeout: httpClientTimeout}

	urlStr := fmt.Sprintf("%s/api/getData?path=%s&roles=value", baseURL, url.QueryEscape(path))
	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try to unmarshal as array first
	var arrayResult []GetDataResponse
	if err := json.Unmarshal(body, &arrayResult); err == nil && len(arrayResult) > 0 {
		return arrayResult, nil
	}

	// Try as single object
	var singleResult GetDataResponse
	if err := json.Unmarshal(body, &singleResult); err != nil {
		// If both fail, try to provide the actual response in error
		return nil, fmt.Errorf("failed to parse response: %w (response: %s)", err, string(body))
	}

	return []GetDataResponse{singleResult}, nil
}

func setData(baseURL, path, dataType, value string) (*SetDataResponse, error) {
	client := &http.Client{Timeout: httpClientTimeout}

	jsonValue := fmt.Sprintf(`{"type":"%s","%s":"%s"}`, dataType, dataType, value)
	urlStr := fmt.Sprintf("%s/api/setData?path=%s&roles=value&value=%s",
		baseURL, url.QueryEscape(path), url.QueryEscape(jsonValue))

	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to set data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try to unmarshal as SetDataResponse first
	var result SetDataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		// If it fails, check if it's just a boolean (success indicator)
		var boolResult bool
		if err := json.Unmarshal(body, &boolResult); err == nil {
			// API returned a boolean, return empty response to indicate success
			return &SetDataResponse{}, nil
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func setDataInt(baseURL, path string, value int) (*SetDataResponse, error) {
	client := &http.Client{Timeout: httpClientTimeout}

	jsonValue := fmt.Sprintf(`{"type":"i32_","i32_":"%d"}`, value)
	urlStr := fmt.Sprintf("%s/api/setData?path=%s&roles=value&value=%s",
		baseURL, url.QueryEscape(path), url.QueryEscape(jsonValue))

	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to set data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try to unmarshal as SetDataResponse first
	var result SetDataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		// If it fails, check if it's just a boolean (success indicator)
		var boolResult bool
		if err := json.Unmarshal(body, &boolResult); err == nil {
			// API returned a boolean, return empty response to indicate success
			return &SetDataResponse{}, nil
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func showStatus(baseURL string) error {
	deviceName, err := getData(baseURL, "settings:/deviceName")
	if err != nil {
		return fmt.Errorf("failed to get device name: %w", err)
	}
	fmt.Printf("Device: %s\n", deviceName[0].String)

	source, err := getData(baseURL, "settings:/kef/play/physicalSource")
	if err != nil {
		return fmt.Errorf("failed to get source: %w", err)
	}
	fmt.Printf("Source: %s\n", source[0].KefPhysicalSource)

	volume, err := getData(baseURL, "player:volume")
	if err != nil {
		return fmt.Errorf("failed to get volume: %w", err)
	}
	fmt.Printf("Volume: %d\n", volume[0].I32)

	muted, err := getData(baseURL, "settings:/mediaPlayer/mute")
	if err != nil {
		return fmt.Errorf("failed to get mute status: %w", err)
	}
	fmt.Printf("Muted : %t\n", muted[0].GetBool())

	return nil
}

func setSource(source, baseURL string) error {
	result, err := setData(baseURL, "settings:/kef/play/physicalSource", "kefPhysicalSource", source)
	if err != nil {
		return err
	}
	fmt.Printf("Source : %s\n", result.Value.KefPhysicalSource)
	return nil
}

func setVolume(volume int, baseURL string) error {
	if volume < 0 || volume > 100 {
		return fmt.Errorf("volume '%d' invalid, use value between 0 and 100", volume)
	}

	_, err := setDataInt(baseURL, "player:volume", volume)
	if err != nil {
		return err
	}
	fmt.Printf("Volume: %d\n", volume)
	return nil
}

func adjustVolume(direction, baseURL string) error {
	volumeData, err := getData(baseURL, "player:volume")
	if err != nil {
		return fmt.Errorf("failed to get current volume: %w", err)
	}

	volume := volumeData[0].I32

	switch direction {
	case "up":
		if volume <= 90 {
			volume += 10
		}
	case "down":
		if volume >= 10 {
			volume -= 10
		}
	}

	_, err = setDataInt(baseURL, "player:volume", volume)
	if err != nil {
		return err
	}
	fmt.Printf("Volume: %d\n", volume)
	return nil
}

func setMute(command, baseURL string) error {
	value := "False"
	if command == "mute" {
		value = "True"
	}

	result, err := setData(baseURL, "settings:/mediaPlayer/mute", "bool_", value)
	if err != nil {
		return err
	}

	// Convert string boolean to actual boolean for display
	muted := result.Value.BoolString == "True"
	fmt.Printf("Muted : %t\n", muted)
	return nil
}

func browse(baseURL string) error {
	fmt.Printf("Opening %s in browser...\n", baseURL)

	// Try different browser commands for Linux
	browsers := []string{
		"google-chrome",
		"chromium",
		"chromium-browser",
		"xdg-open",
		"sensible-browser",
		"firefox",
	}

	for _, browser := range browsers {
		cmd := fmt.Sprintf("%s %s >/dev/null 2>&1 &", browser, baseURL)
		if err := exec.Command("sh", "-c", cmd).Start(); err == nil {
			return nil
		}
	}

	fmt.Printf("Could not open browser automatically. Please open %s manually\n", baseURL)
	return nil
}

func scanNetwork() error {
	// Get default gateway
	gateway, err := getDefaultGateway()
	if err != nil {
		return fmt.Errorf("failed to get gateway: %w", err)
	}

	// Extract subnet
	parts := strings.Split(gateway, ".")
	if len(parts) != 4 {
		return fmt.Errorf("invalid gateway address: %s", gateway)
	}
	subnet := strings.Join(parts[:3], ".")

	fmt.Printf("Scanning %s.0/24 network subnet\n", subnet)

	// Scan IP range
	for i := 0; i <= 254; i++ {
		address := fmt.Sprintf("%s.%d", subnet, i)
		webAddress := fmt.Sprintf("http://%s", address)

		if i%10 == 0 {
			fmt.Printf("Scanning... %d/255\n", i)
		}

		if checkKEFSpeaker(address, webAddress) {
			fmt.Printf("Found KEF speakers at %s\n", webAddress)
			fmt.Printf("Set KEF_URL environment variable: export KEF_URL=%s\n", webAddress)
			return nil
		}
	}

	fmt.Printf("KEF speakers not found in %s.0/24 network range\n", subnet)
	return nil
}

func getDefaultGateway() (string, error) {
	// Simple approach: try common gateway addresses
	// For a more robust solution, you might want to parse /proc/net/route
	commonGateways := []string{"192.168.0.1", "192.168.1.1", "10.0.0.1", "192.168.2.1"}

	for _, gw := range commonGateways {
		conn, err := net.DialTimeout("tcp", gw+":80", 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return gw, nil
		}
	}

	// Fallback: assume 192.168.0.1
	return "192.168.0.1", nil
}

func checkKEFSpeaker(address, webAddress string) bool {
	// Try to connect to port 80
	conn, err := net.DialTimeout("tcp", address+":80", scanTimeout)
	if err != nil {
		return false
	}
	conn.Close()

	// Check if it's a KEF speaker
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(webAddress)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	return strings.Contains(string(body), "LS50 Wireless II")
}
