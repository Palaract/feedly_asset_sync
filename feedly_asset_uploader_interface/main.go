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
	"embed"

    "github.com/wailsapp/wails/v2"
    "github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed frontend/dist
var assets embed.FS

type Config struct {
    UploadURL string `json:"upload_url"`
    APIKey    string `json:"api_key"`
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

func (a *App) loadConfig() (Config, error) {
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

func (a *App) readCSVData(filename string) (map[string][]string, error) {
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

func (a *App) fetchFeedlyData(config Config) ([]FeedlyList, error) {
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

func (a *App) syncToFeedly(csvData map[string][]string, feedlyData []FeedlyList, config Config) error {
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

func main() {
    app := NewApp()

    err := wails.Run(&options.App{
        Title:     "Feedly Sync",
        Width:     1024,
        Height:    768,
        Assets:    assets,
        OnStartup: app.startup,
        Bind: []interface{}{
            app,
        },
    })

    if err != nil {
        log.Fatalf("Error running Wails app: %v", err)
    }
}