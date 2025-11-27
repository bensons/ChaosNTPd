package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// NTPServer represents the UDP NTP server
type NTPServer struct {
	config  *Config
	tracker *ClientTimeTracker
	conn    *net.UDPConn
}

// TransactionLog represents a transaction log entry
type TransactionLog struct {
	Timestamp   string  `json:"timestamp"`
	Event       string  `json:"event"`
	RequestType string  `json:"request_type"`
	Client      struct {
		IP     string `json:"ip"`
		Port   int    `json:"port"`
		IsNew  bool   `json:"is_new"`
	} `json:"client"`
	Request struct {
		Version          int    `json:"version"`
		Mode             int    `json:"mode"`
		TransmitTimestamp string `json:"transmit_timestamp"`
	} `json:"request"`
	Response struct {
		Stratum              int     `json:"stratum"`
		ReferenceID          string  `json:"reference_id"`
		ActualTime           string  `json:"actual_time"`
		OffsetSeconds        float64 `json:"offset_seconds"`
		OffsetMinutes        float64 `json:"offset_minutes"`
		ManipulatedTime      string  `json:"manipulated_time"`
		ElapsedSeconds       float64 `json:"elapsed_seconds,omitempty"`
		JitterApplied        float64 `json:"jitter_applied,omitempty"`
	} `json:"response"`
	Config struct {
		NMinutes int `json:"N_minutes"`
		XSeconds int `json:"X_seconds"`
		Stratum  int `json:"stratum"`
	} `json:"config"`
	ProcessingTimeMs float64 `json:"processing_time_ms"`
}

// NewNTPServer creates a new NTP server
func NewNTPServer(config *Config) *NTPServer {
	return &NTPServer{
		config:  config,
		tracker: NewClientTimeTracker(config),
	}
}

// Start starts the NTP server
func (s *NTPServer) Start() error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(s.config.Server.Host),
		Port: s.config.Server.Port,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind UDP socket: %w", err)
	}
	s.conn = conn

	LogInfo("ChaosNTPd listening on %s:%d", s.config.Server.Host, s.config.Server.Port)
	LogInfo("Stratum: %d, Initial Offset: ±%d min, Jitter: ±%d sec",
		s.config.NTP.Stratum,
		s.config.TimeManipulation.InitialOffsetMinutes,
		s.config.TimeManipulation.JitterSeconds)

	// Start statistics goroutine
	go s.statsLoop()

	// Handle requests
	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			LogError("Error reading UDP packet: %v", err)
			continue
		}

		// Handle in goroutine for concurrency
		go s.handleRequest(buffer[:n], clientAddr)
	}
}

// handleRequest handles a single NTP request
func (s *NTPServer) handleRequest(data []byte, clientAddr *net.UDPAddr) {
	startTime := time.Now()

	// Parse request
	request, err := ParseNTPPacket(data)
	if err != nil {
		LogError("Error parsing NTP packet from %s: %v", clientAddr.String(), err)
		return
	}

	// Validate it's a client request
	if request.Mode != 3 {
		LogWarning("Ignoring non-client request (mode %d) from %s", request.Mode, clientAddr.String())
		return
	}

	// Get manipulated time
	clientKey := clientAddr.IP.String()
	manipulatedTime, offset, isInitial := s.tracker.GetManipulatedTime(clientKey)

	// Create response
	response := CreateResponse(request, s.config, manipulatedTime)

	// Send response
	responseBytes := response.ToBytes()
	_, err = s.conn.WriteToUDP(responseBytes, clientAddr)
	if err != nil {
		LogError("Error sending response to %s: %v", clientAddr.String(), err)
		return
	}

	processingTime := time.Since(startTime)

	// Log transaction
	if s.config.Logging.LogTransactions {
		s.logTransaction(request, response, clientAddr, offset, isInitial, manipulatedTime, processingTime)
	}
}

// logTransaction logs a transaction
func (s *NTPServer) logTransaction(request, response *NTPPacket, clientAddr *net.UDPAddr,
	offset float64, isInitial bool, manipulatedTime time.Time, processingTime time.Duration) {

	log := TransactionLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Event:     "ntp_request",
	}

	if isInitial {
		log.RequestType = "initial"
	} else {
		log.RequestType = "subsequent"
	}

	log.Client.IP = clientAddr.IP.String()
	log.Client.Port = clientAddr.Port
	log.Client.IsNew = isInitial

	log.Request.Version = int(request.Version)
	log.Request.Mode = int(request.Mode)
	log.Request.TransmitTimestamp = NTPToUnix(request.TransmitTime).UTC().Format(time.RFC3339Nano)

	log.Response.Stratum = int(response.Stratum)
	log.Response.ReferenceID = string(response.ReferenceID[:])
	log.Response.ActualTime = time.Now().UTC().Format(time.RFC3339Nano)
	log.Response.OffsetSeconds = offset
	log.Response.OffsetMinutes = offset / 60.0
	log.Response.ManipulatedTime = manipulatedTime.UTC().Format(time.RFC3339Nano)

	log.Config.NMinutes = s.config.TimeManipulation.InitialOffsetMinutes
	log.Config.XSeconds = s.config.TimeManipulation.JitterSeconds
	log.Config.Stratum = s.config.NTP.Stratum

	log.ProcessingTimeMs = float64(processingTime.Microseconds()) / 1000.0

	// Output as JSON
	if s.config.Logging.Format == "json" {
		jsonData, _ := json.Marshal(log)
		fmt.Println(string(jsonData))
	} else {
		// Text format
		fmt.Printf("[%s] %s request from %s:%d - offset: %.1f sec (%.2f min)\n",
			log.Timestamp, log.RequestType, log.Client.IP, log.Client.Port,
			log.Response.OffsetSeconds, log.Response.OffsetMinutes)
	}
}

// statsLoop periodically logs statistics
func (s *NTPServer) statsLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		clients, requests := s.tracker.GetStats()
		LogInfo("Statistics: %d active clients, %d total requests served", clients, requests)
	}
}

// Stop stops the server
func (s *NTPServer) Stop() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
