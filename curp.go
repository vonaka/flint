package main

import "math"

type CurpN2Paxos struct {
	rs      []string
	leader  string
	latency *LatencyTable
}

func NewCurpN2Paxos(rs []string, t *LatencyTable) *CurpN2Paxos {
	return &CurpN2Paxos{
		rs:      rs,
		leader:  "",
		latency: t,
	}
}

func (c *CurpN2Paxos) SetReplicas(rs []string) {
	c.rs = rs
}

func (c *CurpN2Paxos) GetReplicas() []string {
	return c.rs
}

func (c *CurpN2Paxos) Accept(client string, fast bool) float64 {
	if fast {
		size := (3*len(c.rs))/4 + 1
		if (3*len(c.rs))%4 == 0 {
			size--
		}
		filter := func(a string, rs []string) bool {
			if len(rs) == size-1 {
				if a == c.leader {
					return true
				}
				for _, r := range rs {
					if r == c.leader {
						return true
					}
				}
				return false
			}
			return true
		}
		m := math.Inf(1)
		fastQs := QuorumsOfSize(size, c.rs, filter)
		for _, q := range fastQs {
			qm := 0.0
			for r := range q {
				l := Mul(c.latency.OneWayLatency(client, r), 2)
				qm = math.Max(qm, l)
			}
			m = math.Min(m, qm)
		}
		return math.Min(m, c.Accept(client, false))
	}

	n2paxos := NewPaxos(c.rs, c.latency, true)
	n2paxos.leader = c.leader
	return n2paxos.Accept(client, fast)
}

func (c *CurpN2Paxos) SetAverageBestLeader(cs []string) (string, float64) {
	min := math.Inf(1)
	leader := ""

	for _, r := range c.rs {
		c.leader = r
		if MinWorstLatency {
			l := Average(c, cs, false)
			if l < min {
				min = l
				leader = r
			} else if l == min {
				l1 := Average(c, cs, true)
				c.leader = leader
				l2 := Average(c, cs, true)
				if l1 < l2 {
					leader = r
				}
			}
		} else {
			l := Average(c, cs, true)
			if l < min {
				min = l
				leader = r
			}
		}
	}

	c.leader = leader
	return leader, min
}

func (c *CurpN2Paxos) String() string {
	return "Cu"
}
