# ChaosNTPd Monitoring Client

A comprehensive monitoring tool for testing and analyzing ChaosNTPd behavior over time.

## Overview

The monitoring client (`monitor_client.go`) continuously polls an NTP server at configurable intervals, recording time offsets and variations to a CSV file. This allows you to:

- Observe initial offset behavior (N parameter)
- Track jitter over time (X parameter)
- Validate ChaosNTPd's time manipulation strategy
- Generate data for analysis and visualization
- Simulate real NTP client behavior

## Features

- **Configurable Polling**: Set interval to match real NTP clients (typically 64-1024 seconds)
- **CSV Output**: Detailed records of each request/response
- **Real-time Display**: Live statistics during monitoring
- **Analysis Tools**: Python script for statistical analysis
- **Round-trip Tracking**: Measures network latency

## Quick Start

### Basic Usage

```bash
# Monitor ChaosNTPd with default settings (64s interval, 20 requests)
go run monitor_client.go

# Custom monitoring
go run monitor_client.go -server 127.0.0.1:123 -interval 30 -requests 50

# Save to specific file
go run monitor_client.go -output my_test.csv
```

### Command-Line Options

```
  -server string
        NTP server address (default "127.0.0.1:10123")
  -interval int
        Poll interval in seconds (default 64)
  -requests int
        Maximum number of requests to send (default 20)
  -output string
        Output CSV file (default "ntp_monitor.csv")
```

## CSV Output Format

The monitoring client generates a CSV file with the following columns:

| Column | Description |
|--------|-------------|
| `timestamp` | UTC timestamp of the request |
| `request_number` | Sequential request number (1, 2, 3...) |
| `elapsed_seconds` | Time since monitoring started |
| `poll_interval_seconds` | Configured poll interval |
| `server_time` | Time reported by NTP server |
| `actual_time` | Actual system time |
| `offset_seconds` | Time offset in seconds |
| `offset_minutes` | Time offset in minutes |
| `stratum` | NTP stratum reported by server |
| `reference_id` | Server's reference ID |
| `round_trip_ms` | Round-trip time in milliseconds |

## Example Output

### Real-time Display

```
╔════════════════════════════════════════════════════════════════╗
║              ChaosNTPd Monitoring Client                      ║
╚════════════════════════════════════════════════════════════════╝

Server:         127.0.0.1:10123
Poll Interval:  5s
Max Requests:   10
Output File:    chaos_monitor.csv

Starting monitoring... (Press Ctrl+C to stop)

[22:22:18] Request #1 (elapsed: 0s)
  Server Time:   22:22:50.248
  Actual Time:   22:22:18.145
  Offset:        32.104 sec (0.535 min)
  Stratum:       1
  Reference ID:  CHAO
  Round Trip:    0.30 ms

[22:22:23] Request #2 (elapsed: 5s)
  Server Time:   22:22:54.484
  Actual Time:   22:22:23.145
  Offset:        31.339 sec (0.522 min)
  Stratum:       1
  Reference ID:  CHAO
  Round Trip:    0.27 ms
```

### CSV Sample

```csv
timestamp,request_number,elapsed_seconds,poll_interval_seconds,server_time,actual_time,offset_seconds,offset_minutes,stratum,reference_id,round_trip_ms
2025-11-27T03:22:18.145069Z,1,0.001,5,2025-11-27T03:22:50.248978834Z,2025-11-27T03:22:18.145047Z,32.103932,0.535066,1,CHAO,0.305
2025-11-27T03:22:23.145837Z,2,5.002,5,2025-11-27T03:22:54.48488965Z,2025-11-27T03:22:23.145815Z,31.339075,0.522318,1,CHAO,0.272
```

## Analysis

### Using the Analysis Script

```bash
# Analyze results
python3 analyze_results.py chaos_monitor.csv
```

### Analysis Output

The analysis script provides:

1. **Offset Statistics**
   - Initial offset
   - Final offset
   - Total drift over time
   - Mean offset
   - Standard deviation

2. **Jitter Statistics**
   - Maximum jitter between consecutive samples
   - Average jitter

3. **Detailed Measurements**
   - Per-request breakdown
   - Offset changes
   - Jitter values

4. **Observations**
   - Automated assessment of clock stability
   - Jitter behavior analysis
   - Drift trend identification

### Example Analysis

```
═══════════════════════════════════════════════════════════════
OFFSET STATISTICS
═══════════════════════════════════════════════════════════════
Initial Offset:         32.104 seconds (0.54 min)
Final Offset:           34.510 seconds (0.58 min)
Total Drift:             2.406 seconds (0.04 min)
Mean Offset:            32.019 seconds (0.53 min)
Std Deviation:           1.927 seconds

═══════════════════════════════════════════════════════════════
JITTER STATISTICS
═══════════════════════════════════════════════════════════════
Maximum Jitter:          2.915 seconds
Average Jitter:          1.176 seconds

═══════════════════════════════════════════════════════════════
OBSERVATIONS
═══════════════════════════════════════════════════════════════
✓ Low offset variance - clock appears stable with small jitter
✓ Average jitter (1.18s) indicates controlled chaos
✓ Moderate total drift (2.4s) - clock tracking well
```

## Common Use Cases

### 1. Quick Validation Test

Test ChaosNTPd quickly with frequent polling:

```bash
go run monitor_client.go -interval 5 -requests 10 -output quick_test.csv
```

### 2. Realistic NTP Client Simulation

Simulate a real NTP client with typical polling intervals:

```bash
go run monitor_client.go -interval 64 -requests 100 -output realistic_test.csv
```

### 3. Long-term Stability Test

Monitor for extended period with longer intervals:

```bash
go run monitor_client.go -interval 256 -requests 500 -output longterm_test.csv
```

### 4. Jitter Analysis

Short interval to observe jitter clearly:

```bash
go run monitor_client.go -interval 10 -requests 50 -output jitter_test.csv
python3 analyze_results.py jitter_test.csv
```

## Understanding the Results

### Initial Offset (N Parameter)

The first request shows the **initial offset** applied by ChaosNTPd:
- Configured as ±N minutes in ChaosNTPd
- Should fall within the configured range
- Remains relatively stable throughout the session

### Jitter (X Parameter)

Subsequent requests show **jitter** around the expected time:
- Each response varies by ±X seconds from expected
- Expected time = previous_time + elapsed_time
- Jitter = actual_offset - expected_offset

### Example Behavior

With N=10 minutes and X=3 seconds:

```
Request 1: offset = 32.1 sec    (initial, within ±10 min)
Request 2: offset = 31.3 sec    (jitter: -0.8 sec, within ±3 sec)
Request 3: offset = 30.7 sec    (jitter: -0.6 sec, within ±3 sec)
Request 4: offset = 29.3 sec    (jitter: -1.4 sec, within ±3 sec)
```

### Interpreting Statistics

**Low Jitter (< 3s average)**:
- ChaosNTPd is applying controlled chaos
- Clock appears relatively stable
- Good for testing drift detection

**High Jitter (> 5s average)**:
- Highly unstable clock simulation
- Tests extreme scenarios
- May trigger client panic thresholds

**Drift Trend**:
- Positive drift: Clock running fast
- Negative drift: Clock running slow
- Random walk: Jitter-driven variation

## Integration with ChaosNTPd

### Testing Different Configurations

```bash
# Test with large initial offset
./chaosntpd -N 60 -X 5 &
go run monitor_client.go -interval 10 -requests 30

# Test with small jitter
./chaosntpd -N 10 -X 1 &
go run monitor_client.go -interval 10 -requests 30

# Test with high jitter
./chaosntpd -N 5 -X 10 &
go run monitor_client.go -interval 10 -requests 30
```

### Comparing Stratum Levels

```bash
# Stratum 1 (high trust)
./chaosntpd -s 1 &
go run monitor_client.go -output stratum1.csv

# Stratum 15 (low trust)
./chaosntpd -s 15 &
go run monitor_client.go -output stratum15.csv
```

## Building

```bash
# Build standalone binary
go build -o ntp_monitor monitor_client.go

# Run binary
./ntp_monitor -server 127.0.0.1:123 -interval 64 -requests 100
```

## Tips

1. **Poll Interval**: Use realistic values (64, 128, 256, 512, 1024 seconds) to simulate actual NTP clients

2. **Request Count**: Balance between:
   - Short tests (10-20 requests): Quick validation
   - Medium tests (50-100 requests): Jitter analysis
   - Long tests (500+ requests): Drift trends

3. **Analysis**: Always run the analysis script to get statistical insights

4. **CSV Storage**: Keep CSV files organized by test scenario for comparison

5. **Background Running**: Use `nohup` for long tests:
   ```bash
   nohup ./ntp_monitor -interval 128 -requests 1000 &
   ```

## Troubleshooting

**"connection error"**:
- Check ChaosNTPd is running
- Verify server address and port
- Check firewall settings

**"Invalid response size"**:
- NTP server may not be responding correctly
- Check server logs for errors

**High round-trip times**:
- Network latency issues
- Server overload
- Use localhost for testing

## Future Enhancements

- Graphical visualization (matplotlib/plotly)
- Real-time plotting
- Multiple server monitoring
- Statistics export (JSON)
- Configurable alert thresholds

---

**Monitor Client v1.0** - For testing ChaosNTPd and NTP behavior analysis
