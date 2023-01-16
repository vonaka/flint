package main

import "math"

type Client string

func (c Client) ClosestReplica(rs []string, t *LatencyTable) string {
	min := math.Inf(1)
	closest := ""

	for _, r := range rs {
		l := t.OneWayLatency(r, string(c))
		if l < min {
			min = l
			closest = r
		}
	}

	return closest
}
