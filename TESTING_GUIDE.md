# ChaosNTPd Testing Guide

Complete guide for testing ChaosNTPd with the monitoring client.

## Quick Test Scenarios

### Scenario 1: Basic Functionality Test

**Goal**: Verify ChaosNTPd is working correctly

```bash
# Start server with moderate settings
./chaosntpd -N 5 -X 2 -p 10123 &

# Quick test with monitoring client (5 second intervals, 10 requests)
go run monitor_client.go -server 127.0.0.1:10123 -interval 5 -requests 10

# Analyze results
python3 analyze_results.py ntp_monitor.csv

# Expected results:
# - Initial offset within ±5 minutes
# - Jitter within ±2 seconds
# - Low standard deviation
```

### Scenario 2: Jitter Analysis

**Goal**: Observe and measure jitter behavior

```bash
# Start server with high jitter
./chaosntpd -N 10 -X 5 -p 10123 &

# Monitor with short interval to see jitter clearly
go run monitor_client.go -server 127.0.0.1:10123 -interval 3 -requests 30 -output jitter_test.csv

# Analyze
python3 analyze_results.py jitter_test.csv

# Expected results:
# - Average jitter around 2-3 seconds
# - Maximum jitter up to 5 seconds
# - Visible variations in detailed measurements
```

### Scenario 3: Long-term Drift Test

**Goal**: Monitor behavior over extended period

```bash
# Start server
./chaosntpd -N 30 -X 5 &

# Long test with realistic NTP intervals
go run monitor_client.go -interval 64 -requests 100 -output longterm.csv

# This will run for ~106 minutes (64s * 100 requests)

# Analyze
python3 analyze_results.py longterm.csv

# Expected results:
# - Initial offset within ±30 minutes
# - Gradual drift accumulation
# - Jitter statistics over extended period
```

### Scenario 4: Stratum Comparison

**Goal**: Test different stratum levels

```bash
# Test 1: Stratum 1 (high trust)
./chaosntpd -s 1 -N 15 -X 3 -p 10123 &
go run monitor_client.go -server 127.0.0.1:10123 -interval 10 -requests 20 -output stratum1.csv
pkill chaosntpd

# Test 2: Stratum 15 (low trust)
./chaosntpd -s 15 -N 15 -X 3 -p 10123 &
go run monitor_client.go -server 127.0.0.1:10123 -interval 10 -requests 20 -output stratum15.csv
pkill chaosntpd

# Compare
python3 analyze_results.py stratum1.csv
python3 analyze_results.py stratum15.csv

# Note: Time behavior should be identical, only stratum field differs
```

## Interpreting Test Results

### 1. Verifying Initial Offset (N)

Check the **first request** in the monitoring output:

```
[22:22:18] Request #1 (elapsed: 0s)
  Offset:        32.104 sec (0.535 min)
```

**Verification**:
- Offset should be within ±N minutes
- For N=10: offset should be -600 to +600 seconds
- For N=30: offset should be -1800 to +1800 seconds

### 2. Verifying Jitter (X)

Look at consecutive requests and calculate the change:

```
Request 1: offset = 32.104 sec
Request 2: offset = 31.339 sec  → change = -0.765 sec
Request 3: offset = 30.688 sec  → change = -0.651 sec
```

**Verification**:
- Changes should be within ±X seconds
- Average jitter should be < X
- Maximum jitter should be ≤ X

### 3. Clock Behavior

The "drifting clock" should show:
- Relatively stable offset (not jumping wildly)
- Small variations around expected value
- Jitter-driven randomness

**Good behavior** (working correctly):
```
Req #1: 32.1 sec
Req #2: 31.3 sec (change: -0.8)
Req #3: 30.7 sec (change: -0.6)
Req #4: 29.3 sec (change: -1.4)
```

**Bad behavior** (bug!):
```
Req #1: 32.1 sec
Req #2: 567.2 sec (change: +535.1)  ← Way too large!
Req #3: -234.5 sec (change: -801.7)  ← Massive jump!
```

## Analysis Checklist

After running tests, verify:

- [ ] **Initial offset** is within configured N range
- [ ] **Average jitter** is reasonable (typically < X)
- [ ] **Maximum jitter** does not exceed X
- [ ] **Standard deviation** is low (< 5 seconds for typical configs)
- [ ] **Stratum** matches configured value
- [ ] **Reference ID** shows "CHAO"
- [ ] **Round-trip time** is reasonable (< 10ms for localhost)

## Common Test Patterns

### Pattern 1: Stability Test

Verify the clock stays relatively stable:

```bash
./chaosntpd -N 5 -X 1 &  # Low jitter
go run monitor_client.go -interval 10 -requests 30
python3 analyze_results.py ntp_monitor.csv
```

Expected: Std deviation < 1.5 seconds

### Pattern 2: Chaos Test

Verify high jitter works:

```bash
./chaosntpd -N 20 -X 10 &  # High jitter
go run monitor_client.go -interval 10 -requests 30
python3 analyze_results.py ntp_monitor.csv
```

Expected: Std deviation 5-10 seconds

### Pattern 3: Realistic Simulation

Simulate real NTP client:

```bash
./chaosntpd -N 30 -X 5 &
go run monitor_client.go -interval 64 -requests 50
python3 analyze_results.py ntp_monitor.csv
```

Expected: Gradual drift over ~50 minutes

## Automated Testing Script

Create a test script to run multiple scenarios:

```bash
#!/bin/bash
# test_all.sh - Run comprehensive tests

echo "Running ChaosNTPd Test Suite"
echo "=============================="

# Cleanup
pkill chaosntpd 2>/dev/null
rm -f test_*.csv

# Test 1: Basic functionality
echo "Test 1: Basic Functionality"
./chaosntpd -N 5 -X 2 -p 10123 &
PID=$!
sleep 2
go run monitor_client.go -server 127.0.0.1:10123 -interval 5 -requests 10 -output test_basic.csv
kill $PID
python3 analyze_results.py test_basic.csv > test_basic_analysis.txt
echo "✓ Basic test complete"
sleep 2

# Test 2: High jitter
echo "Test 2: High Jitter"
./chaosntpd -N 10 -X 8 -p 10123 &
PID=$!
sleep 2
go run monitor_client.go -server 127.0.0.1:10123 -interval 5 -requests 10 -output test_jitter.csv
kill $PID
python3 analyze_results.py test_jitter.csv > test_jitter_analysis.txt
echo "✓ Jitter test complete"
sleep 2

# Test 3: Large offset
echo "Test 3: Large Offset"
./chaosntpd -N 60 -X 3 -p 10123 &
PID=$!
sleep 2
go run monitor_client.go -server 127.0.0.1:10123 -interval 5 -requests 10 -output test_offset.csv
kill $PID
python3 analyze_results.py test_offset.csv > test_offset_analysis.txt
echo "✓ Offset test complete"

echo ""
echo "All tests complete!"
echo "Results:"
echo "  - test_basic.csv / test_basic_analysis.txt"
echo "  - test_jitter.csv / test_jitter_analysis.txt"
echo "  - test_offset.csv / test_offset_analysis.txt"
```

## Performance Testing

### Test Server Load

```bash
# Start server
./chaosntpd &

# Multiple concurrent clients
for i in {1..10}; do
  go run monitor_client.go -interval 1 -requests 100 -output client${i}.csv &
done

# Monitor server performance
watch -n 1 'ps aux | grep chaosntpd'
```

### Expected Performance

- **Requests/second**: 1000+ (localhost)
- **Memory usage**: < 50MB for 10,000 clients
- **CPU usage**: < 10% on modern hardware
- **Latency**: < 1ms round-trip (localhost)

## Troubleshooting Test Issues

### Issue: "No offset variation"

**Symptom**: All offsets are identical
**Cause**: Jitter (X) set to 0 or client caching
**Solution**: Set X > 0, restart server

### Issue: "Offset too large"

**Symptom**: Offset exceeds configured N range
**Cause**: Bug in offset calculation
**Solution**: Check server logs, verify N parameter

### Issue: "Connection refused"

**Symptom**: Client cannot connect
**Cause**: Server not running or wrong port
**Solution**: Verify server with `ps aux | grep chaosntpd`

### Issue: "Inconsistent CSV data"

**Symptom**: CSV has missing or malformed data
**Cause**: Client error or disk full
**Solution**: Check disk space, verify CSV writing

## Best Practices

1. **Always analyze results** - Don't just collect CSV, run analysis
2. **Use realistic intervals** - Match actual NTP client behavior (64-1024s)
3. **Test multiple scenarios** - Vary N, X, and stratum
4. **Compare baselines** - Keep reference test results
5. **Document configurations** - Note N, X, stratum in test names
6. **Clean between tests** - Kill servers, remove old CSV files
7. **Version your data** - Include date/config in CSV filenames

## Example Test Session

```bash
# Complete test session
cd adversary-ntpd

# Build everything
go build -o chaosntpd
go build -o ntp_monitor monitor_client.go

# Test 1: Quick validation
./chaosntpd -N 5 -X 2 -p 10123 &
SERVER_PID=$!
sleep 2
./ntp_monitor -server 127.0.0.1:10123 -interval 5 -requests 10 -output quick.csv
python3 analyze_results.py quick.csv
kill $SERVER_PID

# Test 2: Realistic scenario
./chaosntpd -N 30 -X 5 &
SERVER_PID=$!
sleep 2
./ntp_monitor -interval 64 -requests 20 -output realistic.csv
python3 analyze_results.py realistic.csv
kill $SERVER_PID

# Review all results
ls -lh *.csv
cat quick.csv
cat realistic.csv

# Success!
```

---

**Testing Guide v1.0** - Comprehensive testing for ChaosNTPd
