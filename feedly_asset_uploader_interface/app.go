package main

import (
    "context"
    "encoding/csv"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
    "strings"
)

type App struct {
    ctx context.Context
}

func NewApp() *App {
    return &App{}
}

func (a *App) startup(ctx context.Context) {
    a.ctx = ctx
}

func (a *App) GetConfig() (Config, error) {
    return a.loadConfig()
}

func (a *App) UpdateConfig(config Config) error {
    file, err := os.Create("config.json")
    if err != nil {
        return fmt.Errorf("error creating config file: %v", err)
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "    ")
    if err := encoder.Encode(config); err != nil {
        return fmt.Errorf("error encoding config: %v", err)
    }

    return nil
}

func (a *App) ProcessCSVData(csvContent string) (string, error) {
    config, err := a.loadConfig()
    if err != nil {
        return "", fmt.Errorf("error loading config: %v", err)
    }

    if len(csvContent) == 0 {
        return "", fmt.Errorf("empty CSV content")
    }

    reader := csv.NewReader(strings.NewReader(csvContent))
    
    headers, err := reader.Read()
    if err != nil {
        return "", fmt.Errorf("error reading CSV headers: %v", err)
    }

    if len(headers) == 0 {
        return "", fmt.Errorf("no headers found in CSV")
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
            return "", fmt.Errorf("error reading CSV row: %v", err)
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

    if len(data) == 0 {
        return "", fmt.Errorf("no valid data found in CSV")
    }

    feedlyData, err := a.fetchFeedlyData(config)
    if err != nil {
        return "", fmt.Errorf("error fetching Feedly data: %v", err)
    }

    err = a.syncToFeedly(data, feedlyData, config)
    if err != nil {
        return "", fmt.Errorf("error syncing to Feedly: %v", err)
    }

    return "Sync completed successfully", nil
}