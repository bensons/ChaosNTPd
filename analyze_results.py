#!/usr/bin/env python3
"""
Analyze ChaosNTPd monitoring results from CSV file.
"""

import csv
import sys
from datetime import datetime
from statistics import mean, stdev

def analyze_csv(filename):
    """Analyze the monitoring CSV file."""

    print("╔════════════════════════════════════════════════════════════════╗")
    print("║          ChaosNTPd Monitoring Results Analysis                 ║")
    print("╚════════════════════════════════════════════════════════════════╝")
    print()

    # Read CSV data
    offsets = []
    timestamps = []

    with open(filename, 'r') as f:
        reader = csv.DictReader(f)
        rows = list(reader)

        if not rows:
            print("No data in CSV file")
            return

        print(f"Total Requests: {len(rows)}")
        print()

        # Extract data
        for row in rows:
            offsets.append(float(row['offset_seconds']))
            timestamps.append(float(row['elapsed_seconds']))

        # Calculate statistics
        initial_offset = offsets[0]
        final_offset = offsets[-1]
        offset_change = final_offset - initial_offset

        if len(offsets) > 1:
            offset_std = stdev(offsets)
            offset_mean = mean(offsets)

            # Calculate jitter (change between consecutive samples)
            jitters = [abs(offsets[i] - offsets[i-1]) for i in range(1, len(offsets))]
            max_jitter = max(jitters)
            avg_jitter = mean(jitters)
        else:
            offset_std = 0
            offset_mean = offsets[0]
            max_jitter = 0
            avg_jitter = 0

        # Print summary
        print("═══════════════════════════════════════════════════════════════")
        print("OFFSET STATISTICS")
        print("═══════════════════════════════════════════════════════════════")
        print(f"Initial Offset:     {initial_offset:>10.3f} seconds ({initial_offset/60:.2f} min)")
        print(f"Final Offset:       {final_offset:>10.3f} seconds ({final_offset/60:.2f} min)")
        print(f"Total Drift:        {offset_change:>10.3f} seconds ({offset_change/60:.2f} min)")
        print(f"Mean Offset:        {offset_mean:>10.3f} seconds ({offset_mean/60:.2f} min)")
        print(f"Std Deviation:      {offset_std:>10.3f} seconds")
        print()

        print("═══════════════════════════════════════════════════════════════")
        print("JITTER STATISTICS")
        print("═══════════════════════════════════════════════════════════════")
        print(f"Maximum Jitter:     {max_jitter:>10.3f} seconds")
        print(f"Average Jitter:     {avg_jitter:>10.3f} seconds")
        print()

        # Print detailed table
        print("═══════════════════════════════════════════════════════════════")
        print("DETAILED MEASUREMENTS")
        print("═══════════════════════════════════════════════════════════════")
        print(f"{'Req':>4} {'Elapsed':>10} {'Offset':>12} {'Change':>10} {'Jitter':>10}")
        print(f"{'#':>4} {'(sec)':>10} {'(sec)':>12} {'(sec)':>10} {'(sec)':>10}")
        print("───────────────────────────────────────────────────────────────")

        prev_offset = None
        for i, row in enumerate(rows, 1):
            elapsed = float(row['elapsed_seconds'])
            offset = float(row['offset_seconds'])

            if prev_offset is not None:
                change = offset - prev_offset
                jitter = abs(change)
            else:
                change = 0
                jitter = 0

            print(f"{i:>4} {elapsed:>10.1f} {offset:>12.3f} {change:>10.3f} {jitter:>10.3f}")
            prev_offset = offset

        print()

        # Print configuration info
        first_row = rows[0]
        print("═══════════════════════════════════════════════════════════════")
        print("SERVER CONFIGURATION")
        print("═══════════════════════════════════════════════════════════════")
        print(f"Stratum:            {first_row['stratum']}")
        print(f"Reference ID:       {first_row['reference_id']}")
        print(f"Poll Interval:      {first_row['poll_interval_seconds']} seconds")
        print(f"Duration:           {timestamps[-1]:.1f} seconds")
        print()

        # Observations
        print("═══════════════════════════════════════════════════════════════")
        print("OBSERVATIONS")
        print("═══════════════════════════════════════════════════════════════")

        if abs(offset_std) < 5:
            print("✓ Low offset variance - clock appears stable with small jitter")
        else:
            print("⚠ High offset variance - significant time drift observed")

        if avg_jitter < 5:
            print(f"✓ Average jitter ({avg_jitter:.2f}s) indicates controlled chaos")
        else:
            print(f"⚠ Large average jitter ({avg_jitter:.2f}s) - highly unstable")

        if abs(offset_change) > 60:
            print(f"⚠ Large total drift ({offset_change/60:.1f} min) over test period")
        else:
            print(f"✓ Moderate total drift ({offset_change:.1f}s) - clock tracking well")

        print()
        print("═══════════════════════════════════════════════════════════════")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 analyze_results.py <csv_file>")
        sys.exit(1)

    analyze_csv(sys.argv[1])
