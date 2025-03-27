package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	UploadURL string `json:"upload_url"`
	APIKey    string `json:"api_key"` 
	CSVPath   string `json:"csv_path"`
}

type FeedlyEntity struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type FeedlyList struct {
	ID       string         `json:"id,omitempty"`
	Label    string         `json:"label"`
	Type     string         `json:"type"`
	Entities []FeedlyEntity `json:"entities"`
}

func loadConfig() (Config, error) {
	var config Config
	file, err := os.Open("config.json")
	if err != nil {
		return config, fmt.Errorf("error opening config: %v", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return config, fmt.Errorf("error decoding config: %v", err)
	}
	return config, nil
}

func readCSVData(filename string) (map[string][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV headers: %v", err)
	}

	data := make(map[string][]string)
	for _, header := range headers {
		data[header] = []string{}
	}

	rowCount := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV row: %v", err)
		}

		rowCount++
		if rowCount > 51 {
			log.Printf("Warning: CSV has more than 51 rows (including header). Truncating excess rows.")
			break
		}

		for i, value := range record {
			if i < len(headers) && value != "" {
				data[headers[i]] = append(data[headers[i]], value)
			}
		}
	}

	return data, nil
}

func fetchFeedlyData(config Config) ([]FeedlyList, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?details=true", config.UploadURL), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching Feedly data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var feedlyData []FeedlyList
	if err := json.NewDecoder(resp.Body).Decode(&feedlyData); err != nil {
		return nil, fmt.Errorf("error decoding Feedly response: %v", err)
	}

	return feedlyData, nil
}

func syncToFeedly(csvData map[string][]string, feedlyData []FeedlyList, config Config) error {
	client := &http.Client{}

	for listName, entries := range csvData {
		if len(entries) == 0 {
			continue
		}

		var existingLists []FeedlyList
		for _, list := range feedlyData {
			if strings.HasPrefix(list.Label, listName) {
				existingLists = append(existingLists, list)
			}
		}

		var entities []FeedlyEntity
		for _, entry := range entries {
			entities = append(entities, FeedlyEntity{
				Type: "customKeyword",
				Text: entry,
			})
		}

		if len(existingLists) == 0 {
			newList := FeedlyList{
				Label:    listName,
				Type:     "customTopic",
				Entities: entities,
			}

			payload, err := json.Marshal(newList)
			if err != nil {
				return fmt.Errorf("error marshaling new list: %v", err)
			}

			req, err := http.NewRequest("POST", config.UploadURL, strings.NewReader(string(payload)))
			if err != nil {
				return fmt.Errorf("error creating request: %v", err)
			}

			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("error creating list: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return fmt.Errorf("unexpected status code creating list: %d", resp.StatusCode)
			}

			time.Sleep(time.Second)
		} else {
			for _, list := range existingLists {
				if len(list.Entities) >= 50 {
					continue
				}

				list.Entities = entities[:min(50-len(list.Entities), len(entities))]
				
				payload, err := json.Marshal(list)
				if err != nil {
					return fmt.Errorf("error marshaling updated list: %v", err)
				}

				req, err := http.NewRequest("PUT", config.UploadURL, strings.NewReader(string(payload)))
				if err != nil {
					return fmt.Errorf("error creating request: %v", err)
				}

				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))

				resp, err := client.Do(req)
				if err != nil {
					return fmt.Errorf("error updating list: %v", err)
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusNoContent {
					return fmt.Errorf("unexpected status code updating list: %d", resp.StatusCode)
				}

				time.Sleep(time.Second)
			}
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	csvData, err := readCSVData(config.CSVPath)
	if err != nil {
		log.Fatalf("Failed to read CSV data: %v", err)
	}

	feedlyData, err := fetchFeedlyData(config)
	if err != nil {
		log.Fatalf("Failed to fetch Feedly data: %v", err)
	}

	if err := syncToFeedly(csvData, feedlyData, config); err != nil {
		log.Fatalf("Failed to sync data to Feedly: %v", err)
	}

	log.Println("Successfully synced data to Feedly")
}
