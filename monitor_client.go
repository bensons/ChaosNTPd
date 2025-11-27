package main

import (
	"encoding/binary"
	"encoding/csv"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

// MonitorClient continuously polls an NTP server and records results
type MonitorClient struct {
	serverAddr   string
	pollInterval time.Duration
	maxRequests  int
	csvFile      *os.File
	csvWriter    *csv.Writer
	requestCount int
	startTime    time.Time
}

// NTPResponse holds parsed NTP response data
type NTPResponse struct {
	Stratum      uint8
	ReferenceID  string
	ServerTime   time.Time
	ActualTime   time.Time
	Offset       time.Duration
	RoundTrip    time.Duration
}

// NewMonitorClient creates a new monitoring client
func NewMonitorClient(serverAddr string, pollInterval time.Duration, maxRequests int, outputFile string) (*MonitorClient, error) {
	// Create CSV file
	csvFile, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("error creating CSV file: %w", err)
	}

	csvWriter := csv.NewWriter(csvFile)

	// Write CSV header
	header := []string{
		"timestamp",
		"request_number",
		"elapsed_seconds",
		"poll_interval_seconds",
		"server_time",
		"actual_time",
		"offset_seconds",
		"offset_minutes",
		"stratum",
		"reference_id",
		"round_trip_ms",
	}
	if err := csvWriter.Write(header); err != nil {
		csvFile.Close()
		return nil, fmt.Errorf("error writing CSV header: %w", err)
	}
	csvWriter.Flush()

	return &MonitorClient{
		serverAddr:   serverAddr,
		pollInterval: pollInterval,
		maxRequests:  maxRequests,
		csvFile:      csvFile,
		csvWriter:    csvWriter,
		requestCount: 0,
		startTime:    time.Now(),
	}, nil
}

// sendNTPRequest sends an NTP request and returns the response
func (m *MonitorClient) sendNTPRequest() (*NTPResponse, error) {
	// Connect to server
	conn, err := net.Dial("udp", m.serverAddr)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	// Create NTP request packet
	request := make([]byte, 48)
	request[0] = 0x1B // LI=0, VN=3, Mode=3 (client)

	// Set transmit timestamp to current time
	sendTime := time.Now()
	ntpTime := unixToNTP(sendTime)
	binary.BigEndian.PutUint64(request[40:48], ntpTime)

	// Send request
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write(request)
	if err != nil {
		return nil, fmt.Errorf("send error: %w", err)
	}

	// Receive response
	response := make([]byte, 48)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	receiveTime := time.Now()
	if err != nil {
		return nil, fmt.Errorf("receive error: %w", err)
	}

	if n != 48 {
		return nil, fmt.Errorf("invalid response size: %d", n)
	}

	// Parse response
	stratum := response[1]
	refID := string(response[12:16])

	// Parse transmit timestamp (server's send time)
	transmitTime := binary.BigEndian.Uint64(response[40:48])
	serverTime := ntpToUnix(transmitTime)

	// Calculate offset and round trip
	actualTime := time.Now()
	offset := serverTime.Sub(actualTime)
	roundTrip := receiveTime.Sub(sendTime)

	return &NTPResponse{
		Stratum:     stratum,
		ReferenceID: refID,
		ServerTime:  serverTime,
		ActualTime:  actualTime,
		Offset:      offset,
		RoundTrip:   roundTrip,
	}, nil
}

// recordResponse records an NTP response to CSV
func (m *MonitorClient) recordResponse(resp *NTPResponse) error {
	m.requestCount++
	elapsed := time.Since(m.startTime)

	record := []string{
		time.Now().UTC().Format(time.RFC3339Nano),
		fmt.Sprintf("%d", m.requestCount),
		fmt.Sprintf("%.3f", elapsed.Seconds()),
		fmt.Sprintf("%.0f", m.pollInterval.Seconds()),
		resp.ServerTime.UTC().Format(time.RFC3339Nano),
		resp.ActualTime.UTC().Format(time.RFC3339Nano),
		fmt.Sprintf("%.6f", resp.Offset.Seconds()),
		fmt.Sprintf("%.6f", resp.Offset.Minutes()),
		fmt.Sprintf("%d", resp.Stratum),
		resp.ReferenceID,
		fmt.Sprintf("%.3f", float64(resp.RoundTrip.Microseconds())/1000.0),
	}

	if err := m.csvWriter.Write(record); err != nil {
		return fmt.Errorf("error writing CSV record: %w", err)
	}
	m.csvWriter.Flush()
	return nil
}

// printStats prints current statistics
func (m *MonitorClient) printStats(resp *NTPResponse) {
	elapsed := time.Since(m.startTime)
	fmt.Printf("[%s] Request #%d (elapsed: %s)\n",
		time.Now().Format("15:04:05"),
		m.requestCount,
		elapsed.Round(time.Second))
	fmt.Printf("  Server Time:   %s\n", resp.ServerTime.Format("15:04:05.000"))
	fmt.Printf("  Actual Time:   %s\n", resp.ActualTime.Format("15:04:05.000"))
	fmt.Printf("  Offset:        %.3f sec (%.3f min)\n", resp.Offset.Seconds(), resp.Offset.Minutes())
	fmt.Printf("  Stratum:       %d\n", resp.Stratum)
	fmt.Printf("  Reference ID:  %s\n", resp.ReferenceID)
	fmt.Printf("  Round Trip:    %.2f ms\n", float64(resp.RoundTrip.Microseconds())/1000.0)
	fmt.Println()
}

// Run starts the monitoring loop
func (m *MonitorClient) Run() error {
	defer m.csvFile.Close()

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ChaosNTPd Monitoring Client                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Server:         %s\n", m.serverAddr)
	fmt.Printf("Poll Interval:  %s\n", m.pollInterval)
	fmt.Printf("Max Requests:   %d\n", m.maxRequests)
	fmt.Printf("Output File:    %s\n", m.csvFile.Name())
	fmt.Println()
	fmt.Println("Starting monitoring... (Press Ctrl+C to stop)")
	fmt.Println()

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	// Send first request immediately
	resp, err := m.sendNTPRequest()
	if err != nil {
		return fmt.Errorf("error on first request: %w", err)
	}
	m.recordResponse(resp)
	m.printStats(resp)

	// Continue polling at interval
	for m.requestCount < m.maxRequests {
		<-ticker.C

		resp, err := m.sendNTPRequest()
		if err != nil {
			fmt.Printf("Error on request #%d: %v\n", m.requestCount+1, err)
			continue
		}

		m.recordResponse(resp)
		m.printStats(resp)
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("Monitoring complete! %d requests recorded.\n", m.requestCount)
	fmt.Printf("Results saved to: %s\n", m.csvFile.Name())
	fmt.Println("═══════════════════════════════════════════════════════════════")

	return nil
}

// Helper functions for NTP timestamp conversion
func unixToNTP(t time.Time) uint64 {
	unixSecs := t.Unix()
	unixNanos := t.UnixNano()
	ntpSecs := uint64(unixSecs + 2208988800)
	fraction := uint64((unixNanos % 1e9) * (1 << 32) / 1e9)
	return (ntpSecs << 32) | fraction
}

func ntpToUnix(ntp uint64) time.Time {
	secs := (ntp >> 32) - 2208988800
	fraction := ntp & 0xFFFFFFFF
	nanos := (fraction * 1e9) >> 32
	return time.Unix(int64(secs), int64(nanos))
}

func main() {
	// Command-line flags
	serverAddr := flag.String("server", "127.0.0.1:10123", "NTP server address (host:port)")
	pollInterval := flag.Int("interval", 64, "Poll interval in seconds (typical NTP: 64-1024)")
	maxRequests := flag.Int("requests", 20, "Maximum number of requests to send")
	outputFile := flag.String("output", "ntp_monitor.csv", "Output CSV file")

	flag.Parse()

	// Create and run monitoring client
	client, err := NewMonitorClient(
		*serverAddr,
		time.Duration(*pollInterval)*time.Second,
		*maxRequests,
		*outputFile,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	if err := client.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running client: %v\n", err)
		os.Exit(1)
	}
}
