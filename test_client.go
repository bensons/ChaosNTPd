package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func main() {
	// Connect to the server
	conn, err := net.Dial("udp", "127.0.0.1:10123")
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		return
	}
	defer conn.Close()

	// Create NTP request packet
	request := make([]byte, 48)
	request[0] = 0x1B // LI=0, VN=3, Mode=3 (client)

	// Set transmit timestamp to current time
	now := time.Now()
	ntpTime := uint64(now.Unix()+2208988800)<<32 | uint64((now.UnixNano()%1e9)*(1<<32)/1e9)
	binary.BigEndian.PutUint64(request[40:48], ntpTime)

	// Send request
	fmt.Println("Sending NTP request to ChaosNTPd...")
	_, err = conn.Write(request)
	if err != nil {
		fmt.Printf("Error sending: %v\n", err)
		return
	}

	// Receive response
	response := make([]byte, 48)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		fmt.Printf("Error receiving: %v\n", err)
		return
	}

	if n != 48 {
		fmt.Printf("Invalid response size: %d\n", n)
		return
	}

	// Parse response
	stratum := response[1]
	refID := string(response[12:16])

	transmitTime := binary.BigEndian.Uint64(response[40:48])
	secs := (transmitTime >> 32) - 2208988800
	fraction := transmitTime & 0xFFFFFFFF
	nanos := (fraction * 1e9) >> 32
	serverTime := time.Unix(int64(secs), int64(nanos))

	actualTime := time.Now()
	offset := serverTime.Sub(actualTime)

	fmt.Printf("\n✓ Response received!\n")
	fmt.Printf("  Stratum:        %d\n", stratum)
	fmt.Printf("  Reference ID:   %s\n", refID)
	fmt.Printf("  Server Time:    %s\n", serverTime.Format("15:04:05.000"))
	fmt.Printf("  Actual Time:    %s\n", actualTime.Format("15:04:05.000"))
	fmt.Printf("  Offset:         %.2f seconds (%.2f minutes)\n", offset.Seconds(), offset.Minutes())
	fmt.Printf("\n✓ ChaosNTPd is working correctly!\n")
}
