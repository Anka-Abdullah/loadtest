package main

import (
    "bytes"
    "context"
    "crypto/tls"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
    "sync"
    "sync/atomic"
    "time"
)

// Stats menyimpan statistik hasil load test
type Stats struct {
    TotalRequests      atomic.Int64
    SuccessfulRequests atomic.Int64
    FailedRequests     atomic.Int64
    TotalDuration      atomic.Int64 // Dalam nanoseconds
    MinDuration        atomic.Int64
    MaxDuration        atomic.Int64
    StatusCodes        sync.Map
}

// Config konfigurasi untuk load test
type Config struct {
    URL         string
    NumRequests int
    Concurrency int
    Timeout     int
    Method      string
    Body        string
    Headers     []string
    KeepAlive   bool
}

func main() {
    config := parseFlags()
    
    if config.URL == "" {
        fmt.Println("Error: URL harus diisi")
        flag.Usage()
        os.Exit(1)
    }

    fmt.Printf("🚀 Memulai load test...\n")
    fmt.Printf("   URL: %s\n", config.URL)
    fmt.Printf("   Requests: %d\n", config.NumRequests)
    fmt.Printf("   Concurrency: %d\n", config.Concurrency)
    fmt.Printf("   Method: %s\n\n", config.Method)

    stats := &Stats{}
    stats.MinDuration.Store(int64(time.Hour))

    startTime := time.Now()
    runLoadTest(config, stats)
    totalTime := time.Since(startTime)

    printResults(stats, totalTime, config)
}

func parseFlags() *Config {
    config := &Config{}

    flag.StringVar(&config.URL, "u", "", "URL target (required)")
    flag.IntVar(&config.NumRequests, "n", 100, "Jumlah request")
    flag.IntVar(&config.Concurrency, "c", 10, "Level konkurensi")
    flag.IntVar(&config.Timeout, "t", 30, "Timeout dalam detik")
    flag.StringVar(&config.Method, "m", "GET", "HTTP method")
    flag.StringVar(&config.Body, "d", "", "Request body")
    flag.BoolVar(&config.KeepAlive, "k", true, "Gunakan Keep-Alive connections")
    
    var headers string
    flag.StringVar(&headers, "H", "", "Headers (format: 'Header1:Value1;Header2:Value2')")

    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: loadtest [options] url\n\n")
        fmt.Fprintf(os.Stderr, "Options:\n")
        flag.PrintDefaults()
        fmt.Fprintf(os.Stderr, "\nContoh:\n")
        fmt.Fprintf(os.Stderr, "  loadtest -n 10000 -c 100 http://localhost:3000/api/users\n")
        fmt.Fprintf(os.Stderr, "  loadtest -n 5000 -c 50 -m POST -d '{\"name\":\"test\"}' http://localhost:3000/api/users\n")
        fmt.Fprintf(os.Stderr, "  loadtest -n 1000 -c 10 -H 'Authorization:Bearer token;Content-Type:application/json' https://api.example.com\n")
    }

    flag.Parse()

    // Parse headers
    if headers != "" {
        headerPairs := strings.Split(headers, ";")
        for _, pair := range headerPairs {
            if strings.TrimSpace(pair) != "" {
                config.Headers = append(config.Headers, pair)
            }
        }
    }

    // Jika URL diberikan sebagai argumen tanpa flag
    if flag.NArg() > 0 && config.URL == "" {
        config.URL = flag.Arg(0)
    }

    return config
}

func runLoadTest(config *Config, stats *Stats) {
    // Worker pool pattern untuk Go 1.24
    jobs := make(chan int, config.NumRequests)
    results := make(chan bool, config.NumRequests)

    // Setup HTTP client
    client := createHTTPClient(config)

    // Buat request template
    baseReq, err := createBaseRequest(config)
    if err != nil {
        fmt.Printf("Error membuat request: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("📊 Menjalankan requests...")

    // Start workers
    var wg sync.WaitGroup
    for w := 0; w < config.Concurrency; w++ {
        wg.Add(1)
        go worker(w, client, baseReq, stats, jobs, results, &wg)
    }

    // Send jobs
    for i := 0; i < config.NumRequests; i++ {
        jobs <- i
    }
    close(jobs)

    // Wait for completion
    go func() {
        wg.Wait()
        close(results)
    }()

    // Progress monitoring
    completed := 0
    for range results {
        completed++
        if completed%100 == 0 {
            fmt.Printf("   Progress: %d/%d requests\n", completed, config.NumRequests)
        }
    }
}

func createHTTPClient(config *Config) *http.Client {
    return &http.Client{
        Timeout: time.Duration(config.Timeout) * time.Second,
        Transport: &http.Transport{
            TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
            MaxIdleConns:          config.Concurrency * 2,
            MaxIdleConnsPerHost:   config.Concurrency * 2,
            MaxConnsPerHost:       config.Concurrency * 2,
            IdleConnTimeout:       90 * time.Second,
            ResponseHeaderTimeout: time.Duration(config.Timeout) * time.Second,
            DisableKeepAlives:     !config.KeepAlive,
        },
    }
}

func createBaseRequest(config *Config) (*http.Request, error) {
    var body io.Reader
    if config.Body != "" {
        body = bytes.NewBufferString(config.Body)
    }

    req, err := http.NewRequestWithContext(context.Background(), config.Method, config.URL, body)
    if err != nil {
        return nil, err
    }

    // Set default headers
    req.Header.Set("User-Agent", "Go-Load-Tester/1.24")
    req.Header.Set("Accept", "*/*")
    req.Header.Set("Connection", "keep-alive")

    // Auto-detect content type
    if config.Body != "" {
        if strings.HasPrefix(config.Body, "{") || strings.HasPrefix(config.Body, "[") {
            req.Header.Set("Content-Type", "application/json")
        } else if strings.Contains(config.Body, "&") && strings.Contains(config.Body, "=") {
            req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        } else {
            req.Header.Set("Content-Type", "text/plain")
        }
    }

    // Add custom headers
    for _, header := range config.Headers {
        parts := strings.SplitN(header, ":", 2)
        if len(parts) == 2 {
            key := strings.TrimSpace(parts[0])
            value := strings.TrimSpace(parts[1])
            req.Header.Set(key, value)
        }
    }

    return req, nil
}

func worker(id int, client *http.Client, baseReq *http.Request, stats *Stats, 
           jobs <-chan int, results chan<- bool, wg *sync.WaitGroup) {
    defer wg.Done()
    
    for requestNum := range jobs {
        sendRequest(client, baseReq, stats, requestNum)
        results <- true
    }
}

func sendRequest(client *http.Client, baseReq *http.Request, stats *Stats, requestNum int) {
    // Clone request
    req := baseReq.Clone(baseReq.Context())
    
    start := time.Now()
    resp, err := client.Do(req)
    duration := time.Since(start)

    stats.TotalRequests.Add(1)
    stats.TotalDuration.Add(int64(duration))

    // Update min/max duration
    durationNs := int64(duration)
    for {
        currentMin := stats.MinDuration.Load()
        if durationNs < currentMin {
            if stats.MinDuration.CompareAndSwap(currentMin, durationNs) {
                break
            }
        } else {
            break
        }
    }

    for {
        currentMax := stats.MaxDuration.Load()
        if durationNs > currentMax {
            if stats.MaxDuration.CompareAndSwap(currentMax, durationNs) {
                break
            }
        } else {
            break
        }
    }

    if err != nil {
        stats.FailedRequests.Add(1)
        if requestNum < 3 { // Hanya tampilkan 3 error pertama
            fmt.Printf("❌ Request %d gagal: %v\n", requestNum+1, err)
        }
        return
    }

    defer resp.Body.Close()
    
    // Drain response body untuk reuse connection
    _, _ = io.Copy(io.Discard, resp.Body)

    stats.SuccessfulRequests.Add(1)
    
    // Update status codes dengan sync.Map
    if count, ok := stats.StatusCodes.Load(resp.StatusCode); ok {
        stats.StatusCodes.Store(resp.StatusCode, count.(int64)+1)
    } else {
        stats.StatusCodes.Store(resp.StatusCode, int64(1))
    }
}

func printResults(stats *Stats, totalTime time.Duration, config *Config) {
    fmt.Println("\n" + strings.Repeat("=", 60))
    fmt.Println("📈 HASIL LOAD TEST")
    fmt.Println(strings.Repeat("=", 60))

    totalRequests := stats.TotalRequests.Load()
    if totalRequests == 0 {
        fmt.Println("Tidak ada request yang berhasil dijalankan")
        return
    }

    avgDuration := time.Duration(stats.TotalDuration.Load() / totalRequests)
    rps := float64(totalRequests) / totalTime.Seconds()

    // Format output tabel
    fmt.Printf("%-25s %v\n", "Total waktu:", totalTime.Round(time.Millisecond))
    fmt.Printf("%-25s %d\n", "Total requests:", totalRequests)
    fmt.Printf("%-25s %d\n", "Requests sukses:", stats.SuccessfulRequests.Load())
    fmt.Printf("%-25s %d\n", "Requests gagal:", stats.FailedRequests.Load())
    fmt.Printf("%-25s %.2f\n", "Requests per detik:", rps)
    fmt.Printf("%-25s %v\n", "Rata-rata latency:", avgDuration.Round(time.Millisecond))
    fmt.Printf("%-25s %v\n", "Latency terendah:", time.Duration(stats.MinDuration.Load()).Round(time.Millisecond))
    fmt.Printf("%-25s %v\n", "Latency tertinggi:", time.Duration(stats.MaxDuration.Load()).Round(time.Millisecond))

    fmt.Println("\n📊 Distribusi Status Codes:")
    
    // Collect status codes for sorting
    var statusCodes []int
    stats.StatusCodes.Range(func(key, value interface{}) bool {
        statusCodes = append(statusCodes, key.(int))
        return true
    })

    // Simple sort
    for i := 0; i < len(statusCodes); i++ {
        for j := i + 1; j < len(statusCodes); j++ {
            if statusCodes[i] > statusCodes[j] {
                statusCodes[i], statusCodes[j] = statusCodes[j], statusCodes[i]
            }
        }
    }

    for _, code := range statusCodes {
        if count, ok := stats.StatusCodes.Load(code); ok {
            percentage := float64(count.(int64)) / float64(totalRequests) * 100
            fmt.Printf("  %-6d %6d requests  %6.1f%%\n", code, count.(int64), percentage)
        }
    }

    fmt.Println("\n" + strings.Repeat("=", 60))
    
    successRate := float64(stats.SuccessfulRequests.Load()) / float64(totalRequests) * 100
    fmt.Printf("Success Rate: %.1f%% - ", successRate)
    
    if successRate >= 99 {
        fmt.Println("🎉 EXCELLENT")
    } else if successRate >= 95 {
        fmt.Println("✅ VERY GOOD")
    } else if successRate >= 90 {
        fmt.Println("⚠️  GOOD")
    } else if successRate >= 80 {
        fmt.Println("⚠️  FAIR")
    } else {
        fmt.Println("❌ POOR")
    }
    
    // Additional metrics
    fmt.Printf("\n📊 Additional Metrics:\n")
    fmt.Printf("  Concurrency level:     %d\n", config.Concurrency)
    fmt.Printf("  Test duration:         %v\n", totalTime.Round(time.Second))
    fmt.Printf("  Avg. req/worker:       %.1f\n", float64(totalRequests)/float64(config.Concurrency))
    
    if config.KeepAlive {
        fmt.Println("  Connection reuse:      Enabled")
    } else {
        fmt.Println("  Connection reuse:      Disabled")
    }
    
    fmt.Println(strings.Repeat("=", 60))
}