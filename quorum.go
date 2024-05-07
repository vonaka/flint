package main

import (
	"fmt"
	"sort"
)

type Quorum map[string]struct{}

type QuorumFilter func(string, []string) bool

var (
	NoFilter = func(string, []string) bool {
		return true
	}
)

func GetNotInFilter(rs []string) QuorumFilter {
	return func(r string, _ []string) bool {
		for _, s := range rs {
			if r == s {
				return false
			}
		}
		return true
	}
}

func (q Quorum) Equals(q2 Quorum) bool {
	if len(q) != len(q2) {
		return false
	}

	for n := range q {
		if _, exists := q2[n]; !exists {
			return false
		}
	}

	return false
}

func (q Quorum) String() string {
	s := ""
	i := 0
	rs := make([]string, len(q))
	for r := range q {
		rs[i] = r
		i++
	}
	sort.Slice(rs, func(i, j int) bool {
		return rs[i] < rs[j]
	})
	for i, r := range rs {
		s += fmt.Sprint(r)
		if i != len(q)-1 {
			s += "\n"
		}
	}
	return s
}

func QuorumOfSlice(s []string) Quorum {
	q := make(map[string]struct{})
	for _, r := range s {
		q[r] = struct{}{}
	}
	return q
}

func SliceOfQuorum(q Quorum) []string {
	s := []string{}
	for r := range q {
		s = append(s, r)
	}
	return s
}

func QuorumsOfSize(size int, rs []string, filter QuorumFilter) []Quorum {
	var subsets func(int, []string, []string, []Quorum) []Quorum
	subsets = func(s int, sub, rs []string, ans []Quorum) []Quorum {
		if s == 0 {
			return append(ans, QuorumOfSlice(sub))
		} else {
			if len(rs) == 0 {
				return ans
			} else {
				a := rs[0]
				if !filter(a, sub) {
					return subsets(s, sub, rs[1:], ans)
				}
				nsub := make([]string, len(sub))
				copy(nsub, sub)
				nsub = append(nsub, a)
				return subsets(s, sub, rs[1:], subsets(s-1, nsub, rs[1:], ans))
			}
		}
	}
	return subsets(size, []string{}, rs, []Quorum{})
}
