package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type LatencyTable struct {
	regions []string
	latency map[string]map[string]float64

	us     map[string]struct{}
	asia   map[string]struct{}
	europe map[string]struct{}
}

func NewLatencyTable() (*LatencyTable, error) {
	t := &LatencyTable{
		regions: []string{},
		latency: make(map[string]map[string]float64),

		us:     make(map[string]struct{}),
		asia:   make(map[string]struct{}),
		europe: make(map[string]struct{}),
	}

	res, err := http.Get(cloudping)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err := fmt.Sprintf("status code error: %d %s", res.StatusCode, res.Status)
		return nil, errors.New(err)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	doc.Find("th").Each(func(_ int, s *goquery.Selection) {
		if s.HasClass("region_title") {
			region := s.Text()
			t.regions = append(t.regions, region)
			if strings.Contains(region, " us-") {
				t.us[region] = struct{}{}
			} else if strings.Contains(region, " ap-") {
				t.asia[region] = struct{}{}
			} else if strings.Contains(region, " eu-") {
				t.europe[region] = struct{}{}
			}
		}
	})

	for _, region := range t.regions {
		t.latency[region] = make(map[string]float64)
	}

	i := 0
	doc.Find("td").Each(func(_ int, s *goquery.Selection) {
		if s.HasClass("destination") || s.HasClass("source") {
			return
		}
		r1, r2 := t.regions[i/len(t.regions)], t.regions[i%len(t.regions)]
		t.latency[r1][r2], _ = strconv.ParseFloat(s.Text(), 64)
		i++
	})

	return t, nil
}

func NewLatencyTableFromFile(latencyConf string) (*LatencyTable, error) {
	t := &LatencyTable{
		regions: []string{},
		latency: make(map[string]map[string]float64),

		us:     make(map[string]struct{}),
		asia:   make(map[string]struct{}),
		europe: make(map[string]struct{}),
	}

	lf, err := os.Open(latencyConf)
	if err != nil {
		return nil, err
	}
	defer lf.Close()

	s := bufio.NewScanner(lf)
	for s.Scan() {
		data := strings.Fields(s.Text())
		if len(data) != 3 {
			continue
		}
		add1, add2 := true, data[0] != data[1]
		for _, r := range t.regions {
			if add1 && r == data[0] {
				add1 = false
			}
			if add2 && r == data[1] {
				add2 = false
			}
		}
		if add1 {
			t.regions = append(t.regions, data[0])
			t.latency[data[0]] = make(map[string]float64)
			if strings.Contains(data[0], "us-") {
				t.us[data[0]] = struct{}{}
			} else if strings.Contains(data[0], "ap-") {
				t.asia[data[0]] = struct{}{}
			} else if strings.Contains(data[0], "eu-") {
				t.europe[data[0]] = struct{}{}
			}
		}
		if add2 {
			t.regions = append(t.regions, data[1])
			t.latency[data[1]] = make(map[string]float64)
			if strings.Contains(data[1], "us-") {
				t.us[data[1]] = struct{}{}
			} else if strings.Contains(data[1], "ap-") {
				t.asia[data[1]] = struct{}{}
			} else if strings.Contains(data[1], "eu-") {
				t.europe[data[1]] = struct{}{}
			}
		}
		d, err := time.ParseDuration(data[2])
		if err != nil {
			return nil, err
		}
		t.latency[data[0]][data[1]] = float64(d.Milliseconds())
	}
	return t, nil
}

func (t *LatencyTable) OneWayLatency(r1, r2 string) float64 {
	if r1 == r2 {
		return 0.0
	}

	ls1, exists1 := t.latency[r1]
	ls2, exists2 := t.latency[r2]
	if exists1 && !exists2 {
		if l, exists := ls1[r2]; exists {
			return Div(l, 2.0)
		}
		return 0.0
	} else if exists2 && !exists1 {
		if l, exists := ls2[r1]; exists {
			return Div(l, 2.0)
		}
		return 0.0
	}

	l1, exists1 := ls1[r2]
	l2, exists2 := ls2[r1]
	if exists1 && exists2 {
		return Div(Div(l1, 2.0)+Div(l2, 2.0), 2.0)
	} else if exists1 {
		return Div(l1, 2.0)
	} else if exists2 {
		return Div(l2, 2.0)
	}

	return 0.0
}

func (t *LatencyTable) IdOf(r string) string {
	for _, region := range t.regions {
		if r == region {
			f := strings.Fields(region)
			return f[len(f)-1]
		}
	}
	return "none"
}

func (t *LatencyTable) Site(r string) string {
	for _, region := range t.regions {
		if r == region {
			s := strings.Split(region, "(")
			if len(s) < 2 {
				return region
			}
			s = strings.Split(s[1], ")")
			if len(s) >= 1 && s[0] != "" {
				return s[0]
			}
			return region
		}
	}
	return "none"
}

func (t *LatencyTable) String() string {
	s := ""
	i := 0
	for region, ls := range t.latency {
		s += fmt.Sprintln(region, ":")
		j := 0
		for dregion, l := range ls {
			if i == len(t.latency)-1 && j == len(ls)-1 {
				s += fmt.Sprintf("---  %v %v", dregion, l)
			} else {
				s += fmt.Sprintln("--- ", dregion, l)
			}
			j++
		}
		if i != len(t.latency)-1 {
			s += "\n"
		}
		i++
	}
	return s
}

func (t *LatencyTable) StringOf(rs []string) string {
	s := ""
	for _, r1 := range rs {
		for _, r2 := range rs {
			s += fmt.Sprintf("%v %v %0.0fms\n",
				t.IdOf(r1), t.IdOf(r2), Mul(t.OneWayLatency(r1, r2), 2))
		}
	}
	return s
}

func (t *LatencyTable) Export(rs []string, filename string) error {
	nrs := []string{}
	seen := map[string]struct{}{}

	for _, r := range rs {
		if _, exists := seen[r]; !exists {
			seen[r] = struct{}{}
			nrs = append(nrs, r)
		}
	}

	return ioutil.WriteFile(filename, []byte(t.StringOf(nrs)), 0644)
}
