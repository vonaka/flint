package main

import "sort"

var (
	MinWorstLatency = true
)

type Algorithm interface {
	String() string
	SetReplicas(rs []string)
	GetReplicas() []string
	Accept(client string, fast bool) float64
}

func Average(alg Algorithm, cs []string, fast bool) float64 {
	l := 0.0
	for _, c := range cs {
		l += alg.Accept(c, fast)
	}
	return Div(l, float64(len(cs)))
}

type Configuration struct {
	rs []string
	cs []string
	r  int
}

// Computes all possible configurations of `repNum` number of replicas and
// `repNum` + `clientNum` number of clients with `repNum` being co-located
// with servers. The resulting list is ordered by ratio between alg1 and alg2
// in decreasing order.
func Configs(ms []string, repNum, clientNum int, alg1, alg2 Algorithm, fast1, fast2 bool, reconf1, reconf2 func(rs, cs []string), t *LatencyTable) []*Configuration {
	var configs []*Configuration
	qs := QuorumsOfSize(repNum, ms, func(r string, rs []string) bool {
		if len(rs) < repNum-1 {
			return true
		}
		if _, exists := t.us[r]; exists {
			return true
		}
		for _, r := range rs {
			if _, exists := t.us[r]; exists {
				return true
			}
		}
		return false
	})

	for _, q := range qs {
		rs := SliceOfQuorum(q)

		var ns []string
		for _, s := range ms {
			ok := true
			for _, z := range rs {
				if s == z {
					ok = false
					break
				}
			}
			if ok {
				ns = append(ns, s)
			}
		}

		ps := QuorumsOfSize(clientNum-2, ns, NoFilter)
		for _, p := range ps {
			zs := SliceOfQuorum(p)

			pps := QuorumsOfSize(2, rs, NoFilter)
			for _, q := range pps {
				ys := SliceOfQuorum(q)
				cs := make([]string, len(zs)+2)
				copy(cs, zs)
				copy(cs[len(zs):], ys)
				reconf1(rs, cs)
				reconf2(rs, cs)
				ai1 := Average(alg1, cs, fast1)
				ai2 := Average(alg2, cs, fast2)
				r := int(10000 * (1 - Div(ai1, ai2)))
				if r >= 1000 {
					configs = append(configs, &Configuration{
						rs: rs,
						cs: cs,
						r:  r,
					})
				}
			}
		}
	}
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].r > configs[j].r
	})
	return configs
}
