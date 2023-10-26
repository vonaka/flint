# Flint
[![Go Report Card](https://goreportcard.com/badge/github.com/vonaka/flint)](https://goreportcard.com/report/github.com/vonaka/flint)

Flint computes expected latencies for the selected set of AWS clients using different replication protocols.

![Screenshot](flint.png)

Flint imports latency table from [cloudping](https://www.cloudping.co/grid/p_90/timeframe/1D)
with the 90th percentile and a single day timeframe. One can change the link in ![config.go](config.go).
Instead of going through cloudping, flint can be launched with a provided latency table via `-l`
command-line option. See [latency_table_example.txt][latency] for an example of latency table configuration file.

## Supported protocols

- SwiftPaxos (only with fixed fast quorums)
- Paxos
- N<sup>2</sup>Paxos (an all-to-all variant of Paxos)
- CURP (over N<sup>2</sup>Paxos)

## Installation

```bash
go install github.com/vonaka/flint@latest
```

## Navigation

Flint's text-based UI is powered by [tview](https://github.com/rivo/tview).
It supports mouse events and the following hotkeys:

- __Tab__: switch the focus
- __p__: toggle the protocol
- __Esc__: print current latency table
- __e__: export latency table for the selected replicas and clients
- __i__: import latency table from file
- __q__: quit

[latency]: latency_table_example.txt