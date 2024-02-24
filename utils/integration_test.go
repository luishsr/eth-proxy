package utils

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

var iterations int

// This variable is used to control the duration of the load test
var duration int

var addresses = []string{
	"0x5E447e8ecAAaaF0a2fe87fd0B6CF3C02DfBC336f",
	"0x94cdB19C6c3B22a6eEa7160636F2426E02Bc3b58",
	"0x00E298504792F69FEbf5c6B4660974301b4FE1Bd",
	"0x47716d0BB008cE109Ddb5f1F55d3807fB88013a9",
	"0xaf5e49b16E5Ac01dd8c6db64dF8496952420bABa",
	"0x39a473D2f33E74f9e530c7938C84862Bd2693c53",
	"0x7Be8564F406bCf7C2Fed09572B9906bB59102A25",
	"0x7920464135C2b6a4Fe88BE93825f3ec496ec7517",
	"0xDc1E64B55cD30DE4aFfAD73C6A9a52730cb41235",
	"0x8d713c7b82d2631039A9f148d31aE150EFFe892f",
	"0x00a3Ac5E156B4B291ceB59D019121beB6508d93D",
}

func init() {
	// Define the command-line flag. Here, "iterations" is the name of the flag, 1 is the default value,
	// and the string is a description of the flag.
	flag.IntVar(&iterations, "iterations", 1, "Number of times to run the test")
	flag.IntVar(&duration, "duration", 10, "Duration (in seconds) to run the response time test")
}

func TestFetchBalances(t *testing.T) {
	flag.Parse() // Parse the command-line flags

	for i := 0; i < iterations; i++ {
		t.Logf("Iteration %d/%d", i+1, iterations)
		throttle := make(chan struct{}, 1)
		var wg sync.WaitGroup

		for _, address := range addresses {
			wg.Add(1)
			go func(addr string) {
				defer wg.Done()
				throttle <- struct{}{}
				defer func() { <-throttle }()

				url := fmt.Sprintf("http://localhost:8080/eth/balance/%s", addr)
				resp, err := http.Get(url)
				if err != nil {
					t.Errorf("Error fetching balance for address %s: %s", addr, err)
					return
				}
				defer func(Body io.ReadCloser) {
					err := Body.Close()
					if err != nil {
						return
					}
				}(resp.Body)

				if resp.StatusCode != http.StatusOK {
					t.Errorf("Unexpected status code for address %s: %d", addr, resp.StatusCode)
					return
				}

			}(address)

			// Sleep N milliseconds between launching each goroutine
			time.Sleep(100 * time.Millisecond)
		}

		wg.Wait()
	}
}

func TestAPIResponseTime(t *testing.T) {
	flag.Parse() // Parse the command-line flags

	// Using a ticker to continuously make requests
	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()

	endTime := time.Now().Add(time.Duration(duration) * time.Second)

	var totalResponseTime time.Duration
	var requestCount int64

	for range ticker.C {
		if time.Now().After(endTime) {
			break // Stop the test after the specified duration
		}

		var wg sync.WaitGroup
		for _, address := range addresses {
			wg.Add(1)
			go func(addr string) {
				defer wg.Done()

				startTime := time.Now()
				url := fmt.Sprintf("http://localhost:8080/eth/balance/%s", addr)
				_, err := http.Get(url)
				if err != nil {
					t.Logf("Error fetching balance for address %s: %s", addr, err)
					return
				}

				responseTime := time.Since(startTime)
				totalResponseTime += responseTime
				requestCount++
			}(address)
		}
		wg.Wait()

		// Calculate and log the average response time after each batch of requests
		avgResponseTime := totalResponseTime.Seconds() * 1000 / float64(requestCount)  // Convert to milliseconds
		fmt.Printf("\r\033[32mAverage response time: %.2f ms\033[0m", avgResponseTime) // Use \r to overwrite the line and ANSI codes for green color
	}

	fmt.Println()
}
