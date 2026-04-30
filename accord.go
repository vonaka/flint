package main

import "math"

type Accord struct {
	rs      []string
	latency *LatencyTable

	// current (one-way) latency to quorums (ms)
	toFastQuorum float64
	toSlowQuorum float64

	// current quorums
	fastQuorum Quorum
	slowQuorum Quorum
}

func NewAccord(rs []string, t *LatencyTable) *Accord {
	return &Accord{
		rs:      rs,
		latency: t,
	}
}

func (a *Accord) SetReplicas(rs []string) {
	a.rs = rs
}

func (a *Accord) GetReplicas() []string {
	return a.rs
}

func (a *Accord) FindBestQuorums(client string) {
	n := len(a.rs)
	f := (n - 1) / 2
	e := (n - f + 1) / 2
	if e > f {
		e = f
	}

	min := math.Inf(1)
	slowQuorums := QuorumsOfSize(n-f, a.rs, NoFilter)
	var slow Quorum
	for _, q := range slowQuorums {
		max := 0.0
		for r := range q {
			max = math.Max(max, a.latency.OneWayLatency(client, r))
		}
		if min > max {
			slow = q.Copy()
			min = max
		}
	}
	a.toSlowQuorum = min
	a.slowQuorum = slow

	if f == e {
		a.toFastQuorum = min
		a.fastQuorum = a.slowQuorum.Copy()
		return
	}

	min = math.Inf(1)
	fastQuorums := QuorumsOfSize(n-e, a.rs, NoFilter)
	var fast Quorum
	for _, q := range fastQuorums {
		max := 0.0
		for r := range q {
			max = math.Max(max, a.latency.OneWayLatency(client, r))
		}
		if min > max {
			fast = q.Copy()
			min = max
		}
	}
	a.toFastQuorum = min
	a.fastQuorum = fast
}

func (a *Accord) MediumPath() float64 {
	return math.Min(6*a.toSlowQuorum, 2*a.toSlowQuorum+2*a.toFastQuorum)
}

// start(c', c) = max( d(c,r) - d(c',r) ),
// for all r in the intersection of quorums of c and c'
// convoy(c) = max_c'( start(c',c) + 4q(c') + d(c,c') )
func (a *Accord) Convoy(client string) float64 {
	fastQ := a.fastQuorum.Copy()
	slowQ := a.slowQuorum.Copy()

	// clock skew (ms)
	epsilon := 5.0

	convoy := 0.0
	for _, c := range a.rs {
		if c == client {
			continue
		}

		a.FindBestQuorums(c)
		start := math.Inf(-1)
		for r := range slowQ {
			if _, exists := a.slowQuorum[r]; !exists {
				continue
			}
			l := a.latency.OneWayLatency(client, r) - a.latency.OneWayLatency(c, r)
			if _, exists := fastQ[r]; exists {
				if l+epsilon+1+a.latency.OneWayLatency(c, r) <= 3*a.latency.OneWayLatency(client, r) {
					l += epsilon + 1
				} else {
					l = 3*a.latency.OneWayLatency(client, r)
				}
			}
			start = math.Max(start, l)
		}
		start += epsilon + 1
		l := start + a.MediumPath() + a.latency.OneWayLatency(client, c)
		convoy = math.Max(convoy, l)
	}

	a.FindBestQuorums(client)
	return convoy
}

// Assuming co-location (i.e., `client` is among `a.rs`),
// coordinator optimization and both versions of medium path
func (a *Accord) Accept(client string, fast bool) float64 {
	a.FindBestQuorums(client)
	if fast {
		return 2 * a.toFastQuorum
	}
	return math.Max(a.MediumPath(), a.Convoy(client))
}

func (*Accord) String() string {
	return "Accord"
}
