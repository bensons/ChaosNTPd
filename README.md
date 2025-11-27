# ChaosNTPd - Adversarial NTP Daemon

**Version 1.0**

ChaosNTPd is an experimental NTP server that intentionally distributes inaccurate time information for testing and chaos engineering purposes.

⚠️ **WARNING**: This server provides INACCURATE time information. Deploy ONLY in isolated test environments!

## Features

- **Configurable Initial Offset (N)**: Random time offset of ±N minutes for first client contact
- **Configurable Jitter (X)**: Random jitter of ±X seconds for subsequent requests
- **Configurable Stratum**: NTP stratum level (default: 1 for maximum client trust)
- **Client Tracking**: Maintains stateful "drifting clock" for each client
- **JSON Logging**: Detailed transaction logs with offset tracking
- **Concurrent Handling**: Go's goroutines for high-performance request handling

## Quick Start

### Prerequisites

- Go 1.21 or later
- Root/administrator access (required for binding to port 123)

### Build

```bash
go mod download
go build -o chaosntpd
```

### Run

```bash
# With default settings (N=30 min, X=5 sec, stratum=1)
sudo ./chaosntpd

# With custom configuration file
sudo ./chaosntpd --config config.yaml

# With CLI overrides
sudo ./chaosntpd -N 60 -X 10 -s 2

# Full customization
sudo ./chaosntpd --config config.yaml --initial-offset 45 --jitter 15 --stratum 1
```

### Test

```bash
# Query the server
ntpdate -q localhost

# Force sync (WARNING: This will change your system time!)
sudo ntpdate -u localhost

# Continuous monitoring (observe jitter)
watch -n 5 'ntpdate -q localhost | grep offset'
```

## Configuration

### Command-Line Flags

```
  -c, --config          Path to configuration file (default: config.yaml)
  -N, --initial-offset  Initial offset in minutes (overrides config)
  -X, --jitter          Jitter in seconds (overrides config)
  -s, --stratum         NTP stratum level 0-15 (overrides config)
  -p, --port            UDP port (default: 123)
  --host                Bind address (default: 0.0.0.0)
  --log-level           Logging level (DEBUG, INFO, WARNING, ERROR)
```

### Configuration File

Copy `config.example.yaml` to `config.yaml` and customize:

```yaml
ntp:
  stratum: 1  # 1=primary (max trust), 2-15=secondary

time_manipulation:
  initial_offset_minutes: 30  # N: ±30 minutes initial offset
  jitter_seconds: 5           # X: ±5 seconds jitter
```

See `config.example.yaml` for all options.

## How It Works

### Time Manipulation Strategy

1. **First Request (Initial Offset)**:
   - Client contacts ChaosNTPd for the first time
   - Server applies random offset within ±N minutes
   - Stores baseline for this client

2. **Subsequent Requests (Jitter)**:
   - Server calculates elapsed time since last request
   - Adds elapsed time to previous manipulated time
   - Applies random jitter of ±X seconds
   - Updates client baseline

Example:
```
Request #1 at 12:00:00 (initial):
  Actual time: 12:00:00
  Offset: -456 seconds (-7.6 minutes)
  Response: 11:52:24

Request #2 at 12:05:00 (300 seconds later):
  Expected: 11:52:24 + 300s = 11:57:24
  Jitter: +2 seconds
  Response: 11:57:26
```

This creates the illusion of a clock that:
- Started with a large offset
- Ticks at roughly the correct rate
- Has small random instabilities

## Example Output

```
╔════════════════════════════════════════════════════════════════╗
║                        ChaosNTPd v1.0                          ║
║           Adversarial NTP Daemon for Testing                   ║
╚════════════════════════════════════════════════════════════════╝

⚠️  WARNING: This server distributes INACCURATE time information!
⚠️  Deploy ONLY in isolated test environments!

Configuration:
  Listening:      0.0.0.0:123
  Stratum:        1 (0=invalid, 1=primary, 2-15=secondary)
  Reference ID:   CHAO
  Initial Offset: ±30 minutes
  Jitter:         ±5 seconds
  Distribution:   uniform
  Log Format:     json

Starting server...
```

### JSON Transaction Log

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
  "response": {
    "stratum": 1,
    "reference_id": "CHAO",
    "offset_seconds": -456.789,
    "offset_minutes": -7.613,
    "manipulated_time": "2025-11-26T00:22:38.334456Z"
  },
  "config": {
    "N_minutes": 30,
    "X_seconds": 5,
    "stratum": 1
  }
}
```

## Architecture

```
┌─────────────┐
│ NTP Clients │
└──────┬──────┘
       │ UDP port 123
       ▼
┌──────────────────────────────────┐
│      ChaosNTPd (Go)              │
│                                  │
│  ┌────────────────────────────┐  │
│  │ NTP Packet Parser          │  │
│  │ (ntp.go)                   │  │
│  └───────────┬────────────────┘  │
│              ▼                   │
│  ┌────────────────────────────┐  │
│  │ Client Time Tracker        │  │
│  │ (tracker.go)               │  │
│  │ - Track N clients          │  │
│  │ - Initial offset ±N min    │  │
│  │ - Jitter ±X sec            │  │
│  └───────────┬────────────────┘  │
│              ▼                   │
│  ┌────────────────────────────┐  │
│  │ Response Generator         │  │
│  │ (server.go)                │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

## Files

- `main.go` - Entry point and CLI handling
- `config.go` - Configuration loading and parsing
- `ntp.go` - NTP packet structures and conversion
- `tracker.go` - Client state tracking and time manipulation
- `server.go` - UDP server and request handling
- `logger.go` - Simple logging utilities
- `config.example.yaml` - Example configuration file

## Safety Considerations

### Operational Safety

1. **Network Isolation**: Deploy only in isolated test networks
2. **Stratum Warning**: Default stratum 1 makes clients highly trust this source
3. **Client Tracking**: Automatic cleanup of stale clients to prevent memory exhaustion
4. **Reference ID**: "CHAO" identifier clearly marks this as ChaosNTPd

### Security

- Not intended for production use
- No authentication implemented
- Rate limiting not yet implemented
- IP allowlisting not yet implemented

## Monitoring Client

A Go-based monitoring tool (`monitor_client.go`) for testing ChaosNTPd behavior:

```bash
# Basic usage (64s interval, 20 requests)
go run monitor_client.go

# Custom monitoring
go run monitor_client.go -server 127.0.0.1:123 -interval 30 -requests 50 -output test.csv
```

Features:
- Configurable polling intervals and request counts
- CSV output with timestamps, offsets, stratum, and round-trip times
- Real-time display of offset and jitter statistics
- Python analysis script (`analyze_results.py`) for statistical insights

## Testing

Common test scenarios:

```bash
# Basic functionality (moderate settings)
./chaosntpd -N 5 -X 2 -p 10123 &
go run monitor_client.go -server 127.0.0.1:10123 -interval 5 -requests 10
python3 analyze_results.py ntp_monitor.csv

# High jitter test
./chaosntpd -N 10 -X 5 -p 10123 &
go run monitor_client.go -server 127.0.0.1:10123 -interval 3 -requests 30

# Realistic NTP simulation (long-term)
./chaosntpd -N 30 -X 5 &
go run monitor_client.go -interval 64 -requests 100
```

**Verification checklist**:
- Initial offset within ±N minutes
- Jitter within ±X seconds
- Stratum matches configured value
- Reference ID shows "CHAO"

## Use Cases

- **Resilience Testing**: Test distributed systems under time skew
- **Security Research**: Study time-dependent security mechanisms
- **Monitoring Testing**: Validate drift detection systems
- **Chaos Engineering**: Introduce controlled time chaos
- **Educational**: Demonstrate importance of time synchronization
- **Practical Jokes**: Some people have no sense of humor

## Building for Production

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o chaosntpd-linux-amd64

# macOS
GOOS=darwin GOARCH=arm64 go build -o chaosntpd-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o chaosntpd-windows-amd64.exe
```

## License

This is experimental software for testing purposes only. Use at your own risk.
If you wan to use it in production, you're doing it wrong.
But if you do want to use it, in any environment, it's MIT licensed.

## Contributing

Just submit a PR.
