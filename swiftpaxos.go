package main

import "math"

type SwiftPaxos struct {
	rs      []string
	fastQ   Quorum
	leader  string
	latency *LatencyTable
}

func NewSwiftPaxos(rs []string, t *LatencyTable) *SwiftPaxos {
	return &SwiftPaxos{
		rs:      rs,
		fastQ:   nil,
		leader:  "",
		latency: t,
	}
}

func (s *SwiftPaxos) SetReplicas(rs []string) {
	s.rs = rs
}

func (s *SwiftPaxos) GetReplicas() []string {
	return s.rs
}

func (s *SwiftPaxos) Accept(client string, fast bool) float64 {
	m := 0.0
	if fast {
		for r := range s.fastQ {
			l := s.Propagate(client, r) + s.FastAck(r, client)
			m = math.Max(m, l)
		}
		return math.Min(m, s.Accept(client, false))
	}
	m = math.Inf(1)
	slowQs := QuorumsOfSize(len(s.rs)/2+1, s.rs, NoFilter)
	for _, q := range slowQs {
		qm := 0.0
		for r := range q {
			l := s.SlowAck(client, r, client)
			qm = math.Max(qm, l)
		}
		m = math.Min(m, qm)
	}
	return m
}

func (s *SwiftPaxos) Propagate(client, replica string) float64 {
	return s.latency.OneWayLatency(client, replica)
}

func (s *SwiftPaxos) FastAck(replica, to string) float64 {
	return s.latency.OneWayLatency(replica, to)
}

func (s *SwiftPaxos) SlowAck(client, replica, to string) float64 {
	l1 := s.Propagate(client, replica)
	l2 := s.Propagate(client, s.leader) + s.FastAck(s.leader, replica)
	return math.Max(l1, l2) + s.latency.OneWayLatency(replica, to)
}

func (s *SwiftPaxos) SetAverageBestLeader(cs []string) (string, float64) {
	min := math.Inf(1)
	leader := ""

	for r := range s.fastQ {
		s.leader = r
		if MinWorstLatency {
			l := Average(s, cs, false)
			if l < min {
				min = l
				leader = r
			} else if l == min {
				l1 := Average(s, cs, true)
				s.leader = leader
				l2 := Average(s, cs, true)
				if l1 < l2 {
					leader = r
				}
			}
		} else {
			l := Average(s, cs, true)
			if l < min {
				min = l
				leader = r
			}
		}
	}

	s.leader = leader
	return leader, min
}

func (s *SwiftPaxos) SetAverageBestFixedQuorumAndLeader(cs []string, f QuorumFilter) (Quorum, string, float64) {
	var (
		fastQ  Quorum
		leader string
	)
	min := math.Inf(1)
	fastQs := QuorumsOfSize(len(s.rs)/2+1, s.rs, f)
	for _, q := range fastQs {
		s.fastQ = q
		l, m := s.SetAverageBestLeader(cs)
		if m < min {
			min = m
			fastQ = q
			leader = l
		} else if m == min {
			l1 := Average(s, cs, true)
			s.fastQ = fastQ
			s.leader = leader
			l2 := Average(s, cs, true)
			if l1 < l2 {
				leader = l
				fastQ = q
			}
		}
	}

	s.fastQ = fastQ
	s.leader = leader
	return fastQ, leader, min
}

func (s *SwiftPaxos) String() string {
	return "SP"
}
