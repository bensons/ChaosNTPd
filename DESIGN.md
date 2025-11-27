# ChaosNTPd - Design Document

## Overview

This document outlines the design for an adversarial NTP (Network Time Protocol) daemon called "ChaosNTPd" that intentionally distributes inaccurate time information to clients for experimental and testing purposes. The server responds to standard NTP client requests with timestamps initially offset by a random value within ±N minutes of the actual system time, where N is defined by configuration or CLI attribute. The client response is stored in memory, and subsequent responses to that same client return a random value within ±X seconds of the previous time respone plus actual elapsed time.

## Key Configuration Parameters

### N (Initial Offset Minutes)
The **initial offset range** applied when a client first contacts ChaosNTPd. The server responds with a random time offset within **±N minutes** of the actual system time.

- **Default**: 30 minutes
- **Range**: ±N minutes (converted to seconds internally)
- **Use Case**: Determines how far off the initial time response will be
- **Example**: N=30 means initial response could be anywhere from -30 to +30 minutes off

### X (Jitter Seconds)
The **subsequent jitter range** applied to ongoing requests from the same client. After the initial response, the server maintains a "drifting clock" that adds **±X seconds** of random jitter to each response.

- **Default**: 5 seconds
- **Range**: ±X seconds
- **Use Case**: Simulates clock instability while maintaining semi-consistency
- **Example**: X=5 means each subsequent response varies by up to ±5 seconds from expected time

### Stratum (NTP Stratum Level)
The **stratum level** reported to clients, indicating the server's distance from a reference clock.

- **Default**: 1 (primary reference)
- **Range**: 0-15 (0=unspecified, 1=primary, 2-15=secondary, 16=unsynchronized)
- **Use Case**: Controls how much clients trust this time source
- **Configuration**: Set via `--stratum` CLI flag or config file
- **Example**: Stratum 1 makes clients trust this as a primary source (maximum chaos), stratum 15 makes it appear unreliable

### How They Work Together

```
Client Request #1 (Initial):
  actual_time = 12:00:00
  offset = random(-N*60, N*60)  // e.g., -456 seconds (-7.6 minutes)
  response_time = 11:52:24

Client Request #2 (5 minutes later):
  actual_time = 12:05:00
  elapsed = 300 seconds
  expected_time = 11:52:24 + 300 = 11:57:24
  jitter = random(-X, X)  // e.g., +2 seconds
  response_time = 11:57:26

Client Request #3 (5 minutes later):
  actual_time = 12:10:00
  elapsed = 300 seconds (from request #2)
  expected_time = 11:57:26 + 300 = 12:02:26
  jitter = random(-X, X)  // e.g., -3 seconds
  response_time = 12:02:23
```

This approach creates the illusion of a clock that:
- Started with a large initial offset (N)
- Continues to tick at roughly the correct rate
- Has small random instabilities (X) in each reading

## Purpose and Use Cases

### Primary Use Cases
- **Resilience Testing**: Evaluate how distributed systems handle time skew across nodes
- **Security Research**: Study the impact of time manipulation on authentication protocols, certificate validation, and cryptographic operations
- **Monitoring System Testing**: Validate that monitoring and alerting systems detect time drift anomalies
- **Educational Purposes**: Demonstrate the importance of secure time synchronization
- **CTF Challenges**: Create time-based security challenges
- **Chaos Engineering**: Introduce controlled time chaos into test environments
- **Practical Jokes**: No explanation needed

## Architecture

### High-Level Components

```
┌─────────────────┐
│   NTP Clients   │
│  (Test Systems) │
└────────┬────────┘
         │ NTP Request (Port 123/UDP)
         │
         ▼
┌─────────────────────────────────────┐
│   Adversarial NTP Daemon            │
│                                     │
│  ┌──────────────────────────────┐   │
│  │  NTP Protocol Handler        │   │
│  │  - Parse NTP packets         │   │
│  │  - Validate format           │   │
│  └──────────┬───────────────────┘   │
│             │                       │
│             ▼                       │
│  ┌──────────────────────────────┐   │
│  │  Time Manipulation Engine    │   │
│  │  - Get system time           │   │
│  │  - For initial response,     │   │
|  |    apply random offset       │   │
|  |    within ±N minutes of      |   |
|  |    actual system time.       |   |
|  |  - For subsequent response   |   |
|  |    to same client, apply     |   |
|  |    random offset within      |   |
|  |    ±X seconds of original    |   |
|  |    time response plus actual |   |
|  |    elapsed time.             |   |
│  └──────────┬───────────────────┘   │
│             │                       │
│             ▼                       │
│  ┌──────────────────────────────┐   │
│  │  Response Generator          │   │
│  │  - Build NTP response        │   │
│  │  - Calculate timestamps      │   │
│  │  - Set stratum level         │   │
│  └──────────┬───────────────────┘   │
│             │                       │
│             ▼                       │
│  ┌──────────────────────────────┐   │
│  │  Logging & Metrics           │   │
│  │  - Request and response      |   |  
|  |    logging                   │   │
│  │  - Offset tracking           │   │
│  │  - Client statistics         │   │
│  └──────────────────────────────┘   │
└─────────────────────────────────────┘
         │ NTP Response (manipulated time)
         │
         ▼
┌─────────────────┐
│   NTP Clients   │
└─────────────────┘
```

## NTP Protocol Implementation

### NTP Packet Structure
The daemon must handle standard NTPv4 packets (48 bytes):

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|LI | VN  |Mode |    Stratum    |     Poll      |   Precision   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Root Delay                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Root Dispersion                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Reference ID                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                     Reference Timestamp (64)                  +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                      Origin Timestamp (64)                    +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                      Receive Timestamp (64)                   +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                      Transmit Timestamp (64)                  +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Key NTP Fields to Populate

| Field | Value | Description |
|-------|-------|-------------|
| LI (Leap Indicator) | 0 (no warning) | 2 bits indicating leap second status |
| VN (Version Number) | 4 | NTP version (support v3 and v4) |
| Mode | 4 (server) | Server mode response |
| Stratum | 0-15 (configurable, default: 1) | Distance from reference clock (1=primary/most trusted, higher values=less trusted) |
| Poll | Echo client's poll | Maximum interval between messages |
| Precision | System precision | Precision of the system clock |
| Root Delay | ~0 | Round-trip delay to reference |
| Root Dispersion | ~0 | Nominal error relative to reference |
| Reference ID | Custom ID | Identifier (e.g., "CHAO" for "ChaosNTPd") |
| Reference Timestamp | Manipulated time | Last time clock was set |
| Origin Timestamp | Echo from request | From client's Transmit Timestamp |
| Receive Timestamp | Manipulated time | Time request was received (offset) |
| Transmit Timestamp | Manipulated time | Time response was sent (offset) |

## Time Manipulation Strategy

### Client Tracking and Time Offset Logic

ChaosNTPd uses a stateful approach where each client's time baseline is tracked:

1. **First Request (Initial Offset)**:
   - Generate random offset within ±N minutes
   - Calculate: `manipulated_time = actual_time + random(-N*60, N*60)`
   - Store client's baseline: `{client_addr: (manipulated_time, actual_time)}`

2. **Subsequent Requests (Drift Simulation)**:
   - Calculate elapsed time: `elapsed = actual_time_now - stored_actual_time`
   - Expected time: `expected_time = stored_manipulated_time + elapsed`
   - Apply jitter: `manipulated_time = expected_time + random(-X, X)`
   - Update client's baseline

This approach simulates a clock that:
- Starts with a random initial offset from reality
- Continues to "tick" at roughly the correct rate
- Has random jitter of ±X seconds on each response
- Appears semi-consistent to the client while still introducing chaos

### Implementation

```python
import random
import time
from typing import Dict, Tuple

class ClientTimeTracker:
    """Tracks manipulated time for each client"""

    def __init__(self, initial_offset_minutes: int, jitter_seconds: int):
        self.N = initial_offset_minutes  # ±N minutes for initial offset
        self.X = jitter_seconds          # ±X seconds for subsequent jitter
        self.client_state: Dict[tuple, Tuple[float, float]] = {}
        # Maps client_addr -> (last_manipulated_time, last_actual_time)

    def get_manipulated_time(self, client_addr: tuple) -> Tuple[float, float]:
        """
        Get manipulated time for a client.
        Returns: (manipulated_time, offset_applied)
        """
        actual_time = time.time()

        if client_addr not in self.client_state:
            # First request from this client - apply initial offset
            offset = random.uniform(-self.N * 60, self.N * 60)
            manipulated_time = actual_time + offset
            self.client_state[client_addr] = (manipulated_time, actual_time)
            return manipulated_time, offset
        else:
            # Subsequent request - apply jitter to drifting baseline
            last_manip_time, last_actual_time = self.client_state[client_addr]
            elapsed = actual_time - last_actual_time

            # Expected time if clock was ticking normally (but offset)
            expected_time = last_manip_time + elapsed

            # Apply small random jitter
            jitter = random.uniform(-self.X, self.X)
            manipulated_time = expected_time + jitter

            # Update state
            self.client_state[client_addr] = (manipulated_time, actual_time)

            # Calculate actual offset from true time
            offset = manipulated_time - actual_time
            return manipulated_time, offset

    def cleanup_stale_clients(self, max_age_seconds: int = 3600):
        """Remove clients that haven't been seen recently"""
        current_time = time.time()
        stale_clients = [
            addr for addr, (_, last_actual) in self.client_state.items()
            if current_time - last_actual > max_age_seconds
        ]
        for addr in stale_clients:
            del self.client_state[addr]
```

### Configurable Parameters

```yaml
time_manipulation:
  # Initial offset range for first client request
  initial_offset_minutes: 30  # N: Random offset within ±N minutes

  # Subsequent jitter for ongoing requests
  jitter_seconds: 5  # X: Random jitter within ±X seconds

  # Distribution type for randomization
  distribution: "uniform"  # Options: uniform, normal, exponential

  # Client state management
  client_tracking:
    cleanup_interval_seconds: 300  # How often to clean up stale clients
    max_client_age_seconds: 3600   # Remove clients not seen for 1 hour
    max_tracked_clients: 10000     # Memory protection
```

## Implementation Details

### Technology Stack Options

#### Option A: Python
**Pros:**
- Rapid development
- Excellent for prototyping
- Rich standard library for NTP time handling
- Easy to extend and modify

**Cons:**
- Potentially lower performance for high request rates
- GIL may limit concurrent handling

**Key Libraries:**
- `socket` for UDP communication
- `struct` for binary packet parsing
- `asyncio` or `twisted` for async I/O

#### Option B: Go
**Pros:**
- Excellent concurrency with goroutines
- High performance
- Easy deployment (single binary)
- Good standard library support

**Cons:**
- More verbose than Python
- Steeper learning curve

**Key Libraries:**
- `net` package for UDP
- `encoding/binary` for packet handling
- Goroutines for concurrent client handling

#### Option C: Rust
**Pros:**
- Maximum performance
- Memory safety
- Zero-cost abstractions

**Cons:**
- Steepest learning curve
- Slower development cycle

### Recommended: Python for Prototype, Go for Production

### Core Components

#### 1. NTP Packet Parser

```python
import struct
from dataclasses import dataclass

@dataclass
class NTPPacket:
    """Represents an NTP packet structure"""
    leap_indicator: int
    version: int
    mode: int
    stratum: int
    poll: int
    precision: int
    root_delay: int
    root_dispersion: int
    reference_id: bytes
    reference_timestamp: float
    origin_timestamp: float
    receive_timestamp: float
    transmit_timestamp: float

    @classmethod
    def from_bytes(cls, data: bytes):
        """Parse NTP packet from raw bytes"""
        if len(data) < 48:
            raise ValueError("Invalid NTP packet: too short")

        # Unpack the packet structure
        # See RFC 5905 for format details
        unpacked = struct.unpack('!B B B b 11I', data[0:48])

        # Extract leap, version, mode from first byte
        leap = (unpacked[0] >> 6) & 0x3
        version = (unpacked[0] >> 3) & 0x7
        mode = unpacked[0] & 0x7

        # Convert NTP timestamps (seconds since 1900) to Unix timestamps
        def ntp_to_unix(ntp_time_high, ntp_time_low):
            ntp_timestamp = (ntp_time_high << 32) | ntp_time_low
            # NTP epoch: 1900-01-01, Unix epoch: 1970-01-01
            # Difference: 2208988800 seconds
            unix_timestamp = ntp_timestamp / 2**32 - 2208988800
            return unix_timestamp

        return cls(
            leap_indicator=leap,
            version=version,
            mode=mode,
            stratum=unpacked[1],
            poll=unpacked[2],
            precision=unpacked[3],
            root_delay=unpacked[4],
            root_dispersion=unpacked[5],
            reference_id=struct.pack('!I', unpacked[6]),
            reference_timestamp=ntp_to_unix(unpacked[7], unpacked[8]),
            origin_timestamp=ntp_to_unix(unpacked[9], unpacked[10]),
            receive_timestamp=ntp_to_unix(unpacked[11], unpacked[12]),
            transmit_timestamp=ntp_to_unix(unpacked[13], unpacked[14])
        )

    def to_bytes(self) -> bytes:
        """Convert NTP packet to raw bytes"""
        # Implementation details...
        pass
```

#### 2. UDP Server

```python
import socket
import logging

class NTPServer:
    def __init__(self, host='0.0.0.0', port=123):
        self.host = host
        self.port = port
        self.socket = None
        self.logger = logging.getLogger(__name__)

    def start(self):
        """Start the NTP server"""
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.socket.bind((self.host, self.port))
        self.logger.info(f"NTP server listening on {self.host}:{self.port}")

        while True:
            try:
                data, addr = self.socket.recvfrom(1024)
                self.handle_request(data, addr)
            except Exception as e:
                self.logger.error(f"Error handling request: {e}")

    def handle_request(self, data: bytes, addr: tuple):
        """Handle incoming NTP request"""
        try:
            # Parse incoming packet
            request = NTPPacket.from_bytes(data)

            # Generate response with manipulated time
            response = self.generate_response(request)

            # Send response
            self.socket.sendto(response.to_bytes(), addr)

            # Log the transaction
            self.log_transaction(addr, request, response)

        except Exception as e:
            self.logger.error(f"Error processing request from {addr}: {e}")
```

#### 3. Response Generator

```python
import time
import random

class ResponseGenerator:
    def __init__(self, config):
        self.config = config
        # Initialize client time tracker with N and X from config
        initial_offset_min = config.get('initial_offset_minutes', 30)
        jitter_sec = config.get('jitter_seconds', 5)
        self.time_tracker = ClientTimeTracker(initial_offset_min, jitter_sec)

    def generate_response(self, request: NTPPacket, client_addr: tuple) -> tuple:
        """Generate NTP response with manipulated time"""

        # Get manipulated time using stateful tracking
        manip_time, offset = self.time_tracker.get_manipulated_time(client_addr)

        # Determine if this is initial or subsequent request
        is_initial = client_addr not in self.time_tracker.client_state or \
                     len(self.time_tracker.client_state[client_addr]) == 0

        # Create response packet
        response = NTPPacket(
            leap_indicator=0,  # No warning
            version=request.version,  # Echo client version
            mode=4,  # Server mode
            stratum=self.config.get('stratum', 1),  # Default: stratum 1
            poll=request.poll,  # Echo client poll
            precision=-20,  # ~1 microsecond (2^-20 seconds)
            root_delay=0,
            root_dispersion=0,
            reference_id=b'CHAO',  # ChaosNTPd identifier
            reference_timestamp=manip_time - 1,  # Recent reference
            origin_timestamp=request.transmit_timestamp,  # Echo origin
            receive_timestamp=manip_time,  # When we "received" it
            transmit_timestamp=manip_time  # When we're sending
        )

        return response, offset, is_initial
```

## Configuration File Format

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 123
  name: "ChaosNTPd"

ntp:
  stratum: 1  # Default: stratum 1 (primary reference - maximum trust/chaos)
             # Options: 0 (unspecified), 1 (primary), 2-15 (secondary), 16 (unsync)
  reference_id: "CHAO"  # ChaosNTPd identifier (4 bytes)
  precision: -20  # ~1 microsecond

time_manipulation:
  # N: Initial offset range for first client request (in minutes)
  initial_offset_minutes: 30  # Default: ±30 minutes

  # X: Subsequent jitter for ongoing requests (in seconds)
  jitter_seconds: 5  # Default: ±5 seconds

  # Distribution type for randomization
  distribution: "uniform"  # Options: uniform, normal, exponential

  # Client state management
  client_tracking:
    cleanup_interval_seconds: 300  # How often to clean up stale clients
    max_client_age_seconds: 3600   # Remove clients not seen for 1 hour
    max_tracked_clients: 10000     # Memory protection limit

# Command-line override examples:
# --initial-offset 60  (sets N=60 for ±60 minute initial offset)
# --jitter 10          (sets X=10 for ±10 second jitter)
# --stratum 2          (sets stratum=2, appear as secondary reference)

logging:
  level: "INFO"  # DEBUG | INFO | WARNING | ERROR
  format: "json"  # json | text
  output: "stdout"  # stdout | file
  file_path: "/var/log/chaosntpd.log"

  # Log each transaction
  log_transactions: true
  transaction_fields:
    - client_ip
    - client_port
    - timestamp
    - actual_time
    - offset_applied
    - manipulated_time
    - request_type  # "initial" or "subsequent"
    - request_mode
    - request_version
    - is_new_client

metrics:
  enabled: true
  port: 9090  # Prometheus metrics endpoint

  tracked_metrics:
    - total_requests
    - initial_requests  # First contact from client
    - subsequent_requests  # Follow-up requests
    - requests_per_client
    - average_offset
    - offset_distribution
    - jitter_distribution
    - response_time
    - active_tracked_clients

security:
  # IP-based access control
  allow_list:
    - "192.168.0.0/16"
    - "10.0.0.0/8"
    # Empty list = allow all (use with caution)

  # Rate limiting per client
  rate_limit:
    enabled: true
    max_requests_per_minute: 60

  # Prevent amplification attacks
  max_response_size: 1024
```

## Logging and Monitoring

### Transaction Logging

Each NTP request/response should be logged with:
- Timestamp (actual system time)
- Client IP and port
- Request type (initial or subsequent)
- NTP version and mode from request
- Actual system time at request
- Offset applied (in seconds)
- Manipulated time sent to client
- Stratum level reported
- Request processing time

**Example Log Entry - Initial Request (JSON format):**
```json
{
  "timestamp": "2025-11-26T00:30:15.123456Z",
  "event": "ntp_request",
  "request_type": "initial",
  "client": {
    "ip": "192.168.1.100",
    "port": 54321,
    "is_new": true
  },
  "request": {
    "version": 4,
    "mode": 3,
    "transmit_timestamp": 1732582215.123
  },
  "response": {
    "stratum": 1,
    "reference_id": "CHAO",
    "actual_time": 1732582215.123456,
    "offset_seconds": -456.789,
    "offset_minutes": -7.613,
    "manipulated_time": 1732581758.334456
  },
  "config": {
    "N_minutes": 30,
    "X_seconds": 5,
    "stratum": 1
  },
  "processing_time_ms": 0.234
}
```

**Example Log Entry - Subsequent Request (JSON format):**
```json
{
  "timestamp": "2025-11-26T00:35:20.456789Z",
  "event": "ntp_request",
  "request_type": "subsequent",
  "client": {
    "ip": "192.168.1.100",
    "port": 54321,
    "is_new": false,
    "last_seen": "2025-11-26T00:30:15.123456Z"
  },
  "request": {
    "version": 4,
    "mode": 3,
    "transmit_timestamp": 1732582520.456
  },
  "response": {
    "stratum": 1,
    "reference_id": "CHAO",
    "actual_time": 1732582520.456789,
    "elapsed_seconds": 305.333,
    "previous_manipulated_time": 1732581758.334456,
    "expected_time_before_jitter": 1732582063.667789,
    "jitter_applied": 2.145,
    "offset_seconds": -454.644,
    "manipulated_time": 1732582065.812789
  },
  "config": {
    "N_minutes": 30,
    "X_seconds": 5,
    "stratum": 1
  },
  "processing_time_ms": 0.187
}
```

### Metrics to Track

1. **Request Metrics**
   - Total requests received
   - Requests per second
   - Requests per client (track top clients)
   - Request errors/malformed packets

2. **Offset Metrics**
   - Distribution of offsets applied
   - Average offset magnitude
   - Min/max offsets in time window

3. **Performance Metrics**
   - Request processing latency
   - Response send latency
   - Memory usage
   - CPU usage

### Prometheus Metrics Endpoint

```python
from prometheus_client import Counter, Histogram, Gauge

class ChaosNTPMetrics:
    def __init__(self, config):
        # Get N and X from config
        N = config.get('initial_offset_minutes', 30)
        X = config.get('jitter_seconds', 5)

        # Define metrics
        self.requests_total = Counter(
            'chaosntpd_requests_total',
            'Total NTP requests',
            ['client_ip', 'request_type']  # initial or subsequent
        )

        # Dynamic buckets based on N (in seconds)
        offset_buckets = self._generate_offset_buckets(N * 60)
        self.offset_seconds = Histogram(
            'chaosntpd_offset_seconds',
            'Applied time offsets from actual time',
            buckets=offset_buckets
        )

        # Jitter buckets based on X
        jitter_buckets = self._generate_jitter_buckets(X)
        self.jitter_seconds = Histogram(
            'chaosntpd_jitter_seconds',
            'Applied jitter for subsequent requests',
            buckets=jitter_buckets
        )

        self.response_time = Histogram(
            'chaosntpd_response_time_seconds',
            'Response processing time'
        )

        self.active_clients = Gauge(
            'chaosntpd_active_clients',
            'Number of currently tracked clients'
        )

        self.initial_requests = Counter(
            'chaosntpd_initial_requests_total',
            'Total initial client requests'
        )

        self.subsequent_requests = Counter(
            'chaosntpd_subsequent_requests_total',
            'Total subsequent client requests'
        )

    def _generate_offset_buckets(self, max_offset_sec):
        """Generate histogram buckets for offsets based on N"""
        # Create buckets: -N, -N/2, -N/4, -N/8, 0, N/8, N/4, N/2, N
        buckets = []
        divisors = [1, 2, 4, 8]
        for d in reversed(divisors):
            buckets.append(-max_offset_sec / d)
        buckets.append(0)
        for d in reversed(divisors):
            buckets.append(max_offset_sec / d)
        return sorted(buckets)

    def _generate_jitter_buckets(self, max_jitter_sec):
        """Generate histogram buckets for jitter based on X"""
        # Create buckets: -X, -X/2, 0, X/2, X
        return sorted([
            -max_jitter_sec,
            -max_jitter_sec / 2,
            0,
            max_jitter_sec / 2,
            max_jitter_sec
        ])
```

## Security Considerations

### Access Control
1. **IP Allowlist**: Only respond to requests from configured IP ranges
2. **Rate Limiting**: Prevent abuse by limiting requests per client
3. **Packet Validation**: Strictly validate NTP packet format
4. **No Amplification**: Keep responses same size as requests

### Operational Safety
1. **Clear Identification**: Use custom reference ID ("CHAO") to identify this as ChaosNTPd
2. **Network Isolation**: Deploy only in isolated test networks
3. **Stratum Configuration**: Default stratum 1 (primary reference) for maximum chaos - clients will trust this highly
   - **Warning**: Stratum 1 makes clients prioritize ChaosNTPd over other sources
   - Set to stratum 2-15 if less aggressive behavior is desired
   - Never use stratum 0 (invalid) or 16 (unsynchronized)
4. **Audit Logging**: Comprehensive logging for accountability with both initial and subsequent request tracking
5. **Banner/MOTD**: Warning messages in any administrative interface
6. **Startup Warning**: Display configured N, X, and stratum values on startup
7. **Memory Limits**: Enforce `max_tracked_clients` to prevent memory exhaustion

### Preventing Misuse
1. **Require Explicit Configuration**: No "auto-discovery" mode
2. **Warning Banners**: Display warnings on startup
3. **Documentation**: Clear documentation about intended use
4. **Authentication** (optional): Require authentication for server configuration changes

## Testing Strategy

### Unit Tests
- NTP packet parsing and generation
- Time offset calculation
- Configuration loading
- Client session management

### Integration Tests
- Full request/response cycle
- Different NTP client implementations (ntpd, chrony, Windows Time)
- Various offset modes
- Rate limiting behavior
- Access control

### Chaos Testing
Deploy in a test cluster and validate:
1. Application behavior under time skew
2. Distributed system consensus with time drift
3. Certificate validation with time manipulation
4. Authentication token expiration handling
5. Log correlation across time-skewed systems

### Client Compatibility Testing
Test against common NTP clients:
- ntpd (traditional Unix/Linux)
- chronyd (modern Linux)
- systemd-timesyncd
- Windows Time Service (w32time)
- macOS ntpd
- Network equipment (routers, switches)

## Deployment Guide

### Prerequisites
- UDP port 123 available (requires root/admin on most systems)
- Python 3.8+ (for Python implementation) or Go 1.19+
- Isolated network environment
- Monitoring infrastructure (optional but recommended)

### Installation Steps

1. **Clone Repository**
```bash
git clone https://github.com/yourusername/chaosntpd.git
cd chaosntpd
```

2. **Install Dependencies**
```bash
# Python
pip install -r requirements.txt

# Go
go mod download
```

3. **Configure Server**
```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your settings
```

4. **Run Server**
```bash
# Python (requires root for port 123)
sudo python3 chaosntpd.py --config config.yaml

# With custom N and X values via CLI
sudo python3 chaosntpd.py --config config.yaml --initial-offset 60 --jitter 10

# With custom stratum (appear as secondary reference)
sudo python3 chaosntpd.py --config config.yaml --stratum 2

# Full customization
sudo python3 chaosntpd.py --config config.yaml -N 45 -X 15 -s 3

# Go
sudo ./chaosntpd --config config.yaml

# With CLI overrides
sudo ./chaosntpd --config config.yaml -N 45 -X 15 -s 1
```

**Command-Line Arguments:**
- `--config` or `-c`: Path to configuration file
- `--initial-offset` or `-N`: Initial offset range in minutes (overrides config)
- `--jitter` or `-X`: Jitter range in seconds (overrides config)
- `--stratum` or `-s`: NTP stratum level, 0-15 (default: 1, overrides config)
- `--port` or `-p`: UDP port (default: 123)
- `--host`: Bind address (default: 0.0.0.0)
- `--log-level`: Logging level (DEBUG, INFO, WARNING, ERROR)

### Docker Deployment

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 123/udp
EXPOSE 9090/tcp

# Default: N=30 minutes, X=5 seconds, stratum=1
ENV INITIAL_OFFSET_MINUTES=30
ENV JITTER_SECONDS=5
ENV STRATUM=1

CMD ["python3", "chaosntpd.py", "--config", "/etc/chaosntpd/config.yaml"]
```

```bash
docker build -t chaosntpd .

# Run with default settings (N=30, X=5)
docker run -d -p 123:123/udp -p 9090:9090 \
  -v $(pwd)/config.yaml:/etc/chaosntpd/config.yaml \
  chaosntpd

# Run with custom N and X via environment variables
docker run -d -p 123:123/udp -p 9090:9090 \
  -e INITIAL_OFFSET_MINUTES=60 \
  -e JITTER_SECONDS=10 \
  -v $(pwd)/config.yaml:/etc/chaosntpd/config.yaml \
  chaosntpd

# Run with custom stratum (appear as secondary reference)
docker run -d -p 123:123/udp -p 9090:9090 \
  -e STRATUM=2 \
  -v $(pwd)/config.yaml:/etc/chaosntpd/config.yaml \
  chaosntpd

# Run with CLI arguments (full customization)
docker run -d -p 123:123/udp -p 9090:9090 \
  -v $(pwd)/config.yaml:/etc/chaosntpd/config.yaml \
  chaosntpd python3 chaosntpd.py --config /etc/chaosntpd/config.yaml -N 45 -X 15 -s 1
```

### Systemd Service

```ini
[Unit]
Description=ChaosNTPd - Adversarial NTP Daemon
After=network.target

[Service]
Type=simple
User=root
Environment="INITIAL_OFFSET_MINUTES=30"
Environment="JITTER_SECONDS=5"
Environment="STRATUM=1"
ExecStart=/usr/local/bin/chaosntpd --config /etc/chaosntpd/config.yaml
Restart=on-failure
RestartSec=5

# Security hardening (optional)
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

**Installation:**
```bash
sudo cp chaosntpd.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable chaosntpd
sudo systemctl start chaosntpd
sudo systemctl status chaosntpd
```

## Client Configuration Examples

### Linux (chrony)

```bash
# /etc/chrony/chrony.conf
# Comment out default pools
# pool pool.ntp.org iburst

# Add ChaosNTPd server
server chaosntpd.test.local iburst

# Allow large time steps (adjust based on your N value)
# For N=30 minutes, allow up to 1800 seconds
# For N=60 minutes, allow up to 3600 seconds
makestep 3600 -1

# Optional: Increase polling interval for testing
maxpoll 6
```

### Linux (ntpd)

```bash
# /etc/ntp.conf
# Disable default servers
# server 0.pool.ntp.org

# Add ChaosNTPd server
server chaosntpd.test.local iburst

# Allow panic threshold for large offsets
# Default panic is 1000s; disable it for testing with large N values
tinker panic 0

# Optional: Reduce polling for faster testing
minpoll 4
maxpoll 6
```

### Windows

```powershell
# Set NTP server
w32tm /config /manualpeerlist:"chaosntpd.test.local" /syncfromflags:manual /reliable:yes /update

# Allow large time corrections (adjust based on N)
# Default is 15 hours; increase if N > 15 minutes
w32tm /config /update /MaxPosPhaseCorrection:86400
w32tm /config /update /MaxNegPhaseCorrection:86400

# Restart Windows Time service
net stop w32time && net start w32time

# Force sync
w32tm /resync /force

# Check status
w32tm /query /status
```

### macOS

```bash
# Disable default time sync
sudo systemsetup -setusingnetworktime off

# Use ntpdate for manual sync (deprecated but still works)
sudo ntpdate -u chaosntpd.test.local

# Or configure ntpd
echo "server chaosntpd.test.local iburst" | sudo tee /etc/ntp.conf
echo "tinker panic 0" | sudo tee -a /etc/ntp.conf  # Allow large offsets
sudo launchctl unload /System/Library/LaunchDaemons/org.ntp.ntpd.plist
sudo launchctl load /System/Library/LaunchDaemons/org.ntp.ntpd.plist

# Verify configuration
ntpq -p
```

## Monitoring and Observability

### Dashboard Metrics
Create a monitoring dashboard to track:
1. Request rate over time
2. Distribution of time offsets applied
3. Number of unique clients
4. Geographic distribution of clients (if applicable)
5. Error rates
6. Server resource usage

### Alerting
Set up alerts for:
- Unusual request patterns (potential DDoS)
- Server errors or crashes
- Requests from unexpected IP ranges
- Resource exhaustion

### Sample Grafana Dashboard Queries

```promql
# Total request rate
rate(chaosntpd_requests_total[5m])

# Initial vs subsequent request rates
rate(chaosntpd_requests_total{request_type="initial"}[5m])
rate(chaosntpd_requests_total{request_type="subsequent"}[5m])

# Average offset magnitude from actual time
avg(abs(chaosntpd_offset_seconds))

# Average jitter applied (subsequent requests only)
avg(abs(chaosntpd_jitter_seconds))

# 95th percentile response time
histogram_quantile(0.95, chaosntpd_response_time_seconds_bucket)

# Active clients being tracked
chaosntpd_active_clients

# Offset distribution (shows spread across N range)
sum by (le) (chaosntpd_offset_seconds_bucket)

# Jitter distribution (shows spread across X range)
sum by (le) (chaosntpd_jitter_seconds_bucket)

# New client rate (initial requests)
rate(chaosntpd_initial_requests_total[5m])

# Ratio of subsequent to initial requests
rate(chaosntpd_subsequent_requests_total[5m]) / rate(chaosntpd_initial_requests_total[5m])
```

## Future Enhancements

### Phase 2 Features
1. **Per-Client N/X/Stratum Values**: Configure different N, X, and stratum for different client groups (by IP or subnet)
2. **Time-of-Day Variations**: Apply different N/X/stratum values based on time of day
3. **Coordinated Chaos**: Multiple ChaosNTPd servers with coordinated offset patterns
4. **Gradual Convergence**: Mode where offset slowly converges to zero over time
5. **Event-Driven Offsets**: Trigger specific N/X/stratum changes via API
6. **Distribution Modes**: Support normal and exponential distributions for offset/jitter
7. **Client Reset**: API endpoint to reset a client's baseline (force re-initialization)
8. **Dynamic Stratum**: Randomly vary stratum on each response to confuse client selection algorithms

### Phase 3 Features
1. **NTP Pool Support**: Respond as if part of an NTP pool
2. **Authentication Support**: Support NTP authentication (and break it in interesting ways)
3. **IPv6 Support**: Full IPv6 compatibility
4. **RESTful API**: HTTP API for runtime configuration changes
   - GET /config - View current N, X, and stratum values
   - POST /config - Update N, X, and stratum on the fly
   - GET /clients - List tracked clients and their states
   - DELETE /clients/{ip} - Remove specific client from tracking
   - POST /clients/{ip}/stratum - Set custom stratum for specific client
5. **Web Dashboard**: Real-time visualization of clients, offsets, and jitter
6. **Replay Mode**: Record and replay specific offset/jitter patterns
7. **Client Groups**: Organize clients into groups with different N/X/stratum profiles
8. **Stratum Chaos Mode**: Randomly vary stratum between responses to specific clients

## References

### NTP Protocol
- [RFC 5905 - Network Time Protocol Version 4](https://tools.ietf.org/html/rfc5905)
- [RFC 5906 - NTP Security](https://tools.ietf.org/html/rfc5906)
- [NTP.org Documentation](https://www.ntp.org/documentation/)

### Time Synchronization Security
- [Attacking NTP - DEF CON 24](https://www.youtube.com/watch?v=hkw9tFnJk8k)
- [Time Security Research](https://www.usenix.org/conference/usenixsecurity16/technical-sessions/presentation/dowling)

### Implementation Examples
- [Python NTP Server Example](https://github.com/limifly/ntpserver)
- [Go NTP Library](https://github.com/beevik/ntp)

## Appendix

### NTP Timestamp Conversion

NTP uses a 64-bit timestamp:
- 32 bits for seconds since 1900-01-01 00:00:00 UTC
- 32 bits for fractional seconds

Conversion to Unix timestamp (seconds since 1970-01-01):
```python
NTP_EPOCH_OFFSET = 2208988800  # Seconds between 1900 and 1970

def ntp_to_unix(ntp_seconds, ntp_fraction):
    """Convert NTP timestamp to Unix timestamp"""
    return (ntp_seconds - NTP_EPOCH_OFFSET) + (ntp_fraction / 2**32)

def unix_to_ntp(unix_timestamp):
    """Convert Unix timestamp to NTP timestamp"""
    ntp_seconds = int(unix_timestamp) + NTP_EPOCH_OFFSET
    ntp_fraction = int((unix_timestamp % 1) * 2**32)
    return ntp_seconds, ntp_fraction
```

### Stratum Levels

| Stratum | Description |
|---------|-------------|
| 0 | Unspecified or invalid |
| 1 | Primary reference (atomic clock, GPS) |
| 2-15 | Secondary reference (synced to stratum n-1) |
| 16 | Unsynchronized |

**ChaosNTPd Default**: Stratum 1 (primary reference) to maximize client trust and chaos impact.

**Recommendation for Testing**:
- Use stratum 1 (default) for maximum chaos - clients will prioritize ChaosNTPd
- Use stratum 2-3 to appear as secondary reference with moderate trust
- Use stratum 10-15 for low trust scenarios where clients should prefer other sources

### Common NTP Client Commands

```bash
# Query NTP server
ntpq -p chaosntpd.test.local

# Get detailed information
ntpdate -q chaosntpd.test.local

# Force sync
sudo ntpdate -u chaosntpd.test.local

# Check system time status
timedatectl status

# Monitor NTP sync status (watch offset changes due to jitter)
watch -n 1 chronyc tracking

# Check current offset from system clock
ntpdate -q chaosntpd.test.local

# Verbose NTP query (shows reference ID "CHAO")
ntpq -c "rv 0" chaosntpd.test.local

# Monitor ongoing offset/jitter (useful for observing X parameter effect)
watch -n 5 'ntpdate -q chaosntpd.test.local | grep offset'
```

---

**Document Version**: 1.0
**Last Updated**: 2025-11-26
**Authors**: System Design Team
**Status**: Draft for Review
