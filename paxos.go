package main

import "math"

type Paxos struct {
	rs      []string
	n2      bool
	leader  string
	latency *LatencyTable
}

func NewPaxos(rs []string, t *LatencyTable, n2 bool) *Paxos {
	return &Paxos{
		rs:      rs,
		n2:      n2,
		leader:  "",
		latency: t,
	}
}

func (p *Paxos) SetReplicas(rs []string) {
	p.rs = rs
}

func (p *Paxos) GetReplicas() []string {
	return p.rs
}

func (p *Paxos) Accept(c string, _ bool) float64 {
	closest := p.leader
	if p.n2 {
		closest = Client(c).ClosestReplica(p.rs, p.latency)
	}
	m := math.Inf(1)
	slowQs := QuorumsOfSize(len(p.rs)/2+1, p.rs, NoFilter)
	for _, q := range slowQs {
		qm := 0.0
		for r := range q {
			l := p.m2b(c, r, closest)
			qm = math.Max(qm, l)
		}
		m = math.Min(m, qm)
	}
	return Round(m + p.latency.OneWayLatency(closest, c))
}

func (p *Paxos) m2b(client, replica, closest string) float64 {
	l1 := p.latency.OneWayLatency(client, p.leader)
	l2 := p.latency.OneWayLatency(p.leader, replica)
	l3 := p.latency.OneWayLatency(replica, closest)
	return Round(l1 + l2 + l3)
}

func (p *Paxos) SetAverageBestLeader(cs []string) (string, float64) {
	min := math.Inf(1)
	leader := ""

	for _, r := range p.rs {
		p.leader = r
		l := Average(p, cs, true)
		if l < min {
			min = l
			leader = r
		}
	}

	p.leader = leader
	return leader, min
}

func (p *Paxos) String() string {
	if p.n2 {
		return "N²"
	}
	return "Pa"
}
