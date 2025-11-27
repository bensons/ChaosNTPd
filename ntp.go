package main

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	// NTP epoch offset: seconds between 1900-01-01 and 1970-01-01
	ntpEpochOffset = 2208988800

	// NTP packet size
	ntpPacketSize = 48
)

// NTPPacket represents an NTP packet structure
type NTPPacket struct {
	LeapIndicator    uint8   // 2 bits
	Version          uint8   // 3 bits
	Mode             uint8   // 3 bits
	Stratum          uint8   // 8 bits
	Poll             int8    // 8 bits
	Precision        int8    // 8 bits
	RootDelay        uint32  // 32 bits
	RootDispersion   uint32  // 32 bits
	ReferenceID      [4]byte // 32 bits
	ReferenceTime    uint64  // 64 bits
	OriginTime       uint64  // 64 bits
	ReceiveTime      uint64  // 64 bits
	TransmitTime     uint64  // 64 bits
}

// ParseNTPPacket parses an NTP packet from bytes
func ParseNTPPacket(data []byte) (*NTPPacket, error) {
	if len(data) < ntpPacketSize {
		return nil, fmt.Errorf("invalid NTP packet: too short (%d bytes)", len(data))
	}

	packet := &NTPPacket{}

	// Parse first byte (LI, VN, Mode)
	firstByte := data[0]
	packet.LeapIndicator = (firstByte >> 6) & 0x03
	packet.Version = (firstByte >> 3) & 0x07
	packet.Mode = firstByte & 0x07

	// Parse remaining header fields
	packet.Stratum = data[1]
	packet.Poll = int8(data[2])
	packet.Precision = int8(data[3])

	packet.RootDelay = binary.BigEndian.Uint32(data[4:8])
	packet.RootDispersion = binary.BigEndian.Uint32(data[8:12])
	copy(packet.ReferenceID[:], data[12:16])

	// Parse timestamps
	packet.ReferenceTime = binary.BigEndian.Uint64(data[16:24])
	packet.OriginTime = binary.BigEndian.Uint64(data[24:32])
	packet.ReceiveTime = binary.BigEndian.Uint64(data[32:40])
	packet.TransmitTime = binary.BigEndian.Uint64(data[40:48])

	return packet, nil
}

// ToBytes converts the NTP packet to bytes
func (p *NTPPacket) ToBytes() []byte {
	data := make([]byte, ntpPacketSize)

	// First byte (LI, VN, Mode)
	data[0] = (p.LeapIndicator << 6) | (p.Version << 3) | p.Mode

	// Header fields
	data[1] = p.Stratum
	data[2] = uint8(p.Poll)
	data[3] = uint8(p.Precision)

	binary.BigEndian.PutUint32(data[4:8], p.RootDelay)
	binary.BigEndian.PutUint32(data[8:12], p.RootDispersion)
	copy(data[12:16], p.ReferenceID[:])

	// Timestamps
	binary.BigEndian.PutUint64(data[16:24], p.ReferenceTime)
	binary.BigEndian.PutUint64(data[24:32], p.OriginTime)
	binary.BigEndian.PutUint64(data[32:40], p.ReceiveTime)
	binary.BigEndian.PutUint64(data[40:48], p.TransmitTime)

	return data
}

// UnixToNTP converts Unix timestamp to NTP timestamp
func UnixToNTP(t time.Time) uint64 {
	unixSecs := t.Unix()
	unixNanos := t.UnixNano()

	// Seconds since NTP epoch
	ntpSecs := uint64(unixSecs + ntpEpochOffset)

	// Fractional part (nanoseconds to NTP fraction)
	fraction := uint64((unixNanos % 1e9) * (1 << 32) / 1e9)

	return (ntpSecs << 32) | fraction
}

// NTPToUnix converts NTP timestamp to Unix time
func NTPToUnix(ntp uint64) time.Time {
	secs := (ntp >> 32) - ntpEpochOffset
	fraction := ntp & 0xFFFFFFFF
	nanos := (fraction * 1e9) >> 32

	return time.Unix(int64(secs), int64(nanos))
}

// CreateResponse creates an NTP response packet
func CreateResponse(request *NTPPacket, config *Config, manipulatedTime time.Time) *NTPPacket {
	response := &NTPPacket{
		LeapIndicator:  0, // No warning
		Version:        request.Version,
		Mode:           4, // Server mode
		Stratum:        uint8(config.NTP.Stratum),
		Poll:           request.Poll,
		Precision:      int8(config.NTP.Precision),
		RootDelay:      0,
		RootDispersion: 0,
	}

	// Set reference ID (4 bytes ASCII)
	refID := config.NTP.ReferenceID
	if len(refID) > 4 {
		refID = refID[:4]
	}
	for i := 0; i < 4 && i < len(refID); i++ {
		response.ReferenceID[i] = refID[i]
	}

	// Set timestamps
	ntpTime := UnixToNTP(manipulatedTime)
	response.ReferenceTime = ntpTime - (1 << 32) // Reference was 1 second ago
	response.OriginTime = request.TransmitTime    // Echo client's transmit
	response.ReceiveTime = ntpTime                // When we "received" it
	response.TransmitTime = ntpTime               // When we're sending

	return response
}
