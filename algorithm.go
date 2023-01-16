package main

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
