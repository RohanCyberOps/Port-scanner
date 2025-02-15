package main

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"text/template"
	"time"
)

// Struct to hold scan results
type ScanResult struct {
	Port    int
	Status  string
	Message string
}

// Struct to hold the scan configuration
type ScanConfig struct {
	IP             string
	Protocol       string
	StartPort      int
	EndPort        int
	Timeout        time.Duration
	MaxConcurrency int
}

// Global variables to store scan results and configuration
var (
	scanResults []ScanResult
	scanConfig  ScanConfig
)

// Function to scan a single port
func scanPort(protocol, ip string, port int, timeout time.Duration, wg *sync.WaitGroup, results chan ScanResult) {
	defer wg.Done()

	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout(protocol, address, timeout)
	if err != nil {
		results <- ScanResult{Port: port, Status: "Closed", Message: err.Error()}
		return
	}
	conn.Close()
	results <- ScanResult{Port: port, Status: "Open", Message: "Success"}
}

// Function to start the scan
func startScan(config ScanConfig) {
	var wg sync.WaitGroup
	results := make(chan ScanResult)
	concurrencyLimit := make(chan struct{}, config.MaxConcurrency)

	scanResults = []ScanResult{} // Reset previous results

	for port := config.StartPort; port <= config.EndPort; port++ {
		wg.Add(1)
		concurrencyLimit <- struct{}{} // Limit concurrency
		go func(port int) {
			defer func() { <-concurrencyLimit }()
			scanPort(config.Protocol, config.IP, port, config.Timeout, &wg, results)
		}(port)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		scanResults = append(scanResults, result)
	}
}

// Handler for the home page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, scanResults)
}

// Handler to start the scan
func scanHandler(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	ip := r.FormValue("ip")
	protocol := r.FormValue("protocol")
	startPort, _ := strconv.Atoi(r.FormValue("startPort"))
	endPort, _ := strconv.Atoi(r.FormValue("endPort"))
	timeout, _ := time.ParseDuration(r.FormValue("timeout"))
	maxConcurrency, _ := strconv.Atoi(r.FormValue("maxConcurrency"))

	// Validate input
	if ip == "" || protocol == "" || startPort < 1 || endPort > 65535 || startPort > endPort || timeout <= 0 || maxConcurrency < 1 {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Set scan configuration
	scanConfig = ScanConfig{
		IP:             ip,
		Protocol:       protocol,
		StartPort:      startPort,
		EndPort:        endPort,
		Timeout:        timeout,
		MaxConcurrency: maxConcurrency,
	}

	// Start the scan
	go startScan(scanConfig)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	// Serve static files (CSS, JS, etc.)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Define routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/scan", scanHandler)

	// Start the server
	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", nil)
}
