package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	clientsPr  tview.Primitive
	replicasPr tview.Primitive

	protocolPr *tview.DropDown

	pages *tview.Pages

	quorumPr      *tview.TextView
	leaderPr      *tview.TextView
	latencyPr     *tview.TextView
	clientsInfoPr *tview.TextView

	selectedClients  []string
	selectedReplicas []string

	application *tview.Application
)

func Regions(label string, t *LatencyTable, f func(rs map[string]struct{})) tview.Primitive {
	form := tview.NewForm()
	s := map[string]struct{}{}

	for _, r := range t.regions {
		c := tview.NewCheckbox()
		c.SetLabel(t.Site(r))
		c.SetChecked(false)
		rr := r
		c.SetChangedFunc(func(checked bool) {
			r := rr
			if checked {
				s[r] = struct{}{}
			} else if _, exists := s[r]; exists {
				delete(s, r)
			}
			if f != nil {
				f(s)
			}
		})
		form.AddFormItem(c)
	}

	form.SetBorder(true).SetTitle(label).SetTitleAlign(tview.AlignLeft)
	form.SetItemPadding(0)
	form.ClearButtons()
	form.SetLabelColor(tcell.ColorWhite)
	form.SetFieldTextColor(tcell.ColorWhite)
	form.SetFieldBackgroundColor(tcell.ColorGrey)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch key := event.Key(); key {
		case tcell.KeyDown:
			return tcell.NewEventKey(tcell.KeyTab, 't', tcell.ModNone)
		case tcell.KeyUp:
			return tcell.NewEventKey(tcell.KeyBacktab, 't', tcell.ModNone)
		}
		return event
	})

	return form
}

func Redraw(t *LatencyTable) {
	if protocolPr == nil || selectedReplicas == nil || selectedClients == nil {
		if latencyPr != nil {
			latencyPr.Clear()
		}
		if clientsInfoPr != nil {
			clientsInfoPr.Clear()
		}
		return
	}
	if len(selectedClients) == 0 || len(selectedReplicas) == 0 {
		quorumPr.Clear()
		leaderPr.Clear()
		latencyPr.Clear()
		clientsInfoPr.Clear()
		return
	}

	sp := NewSwiftPaxos(selectedReplicas, t)
	c := NewCurpN2Paxos(selectedReplicas, t)
	p := NewPaxos(selectedReplicas, t, false)
	n := NewPaxos(selectedReplicas, t, true)
	quorumSp, leaderSp, _ := sp.SetAverageBestFixedQuorumAndLeader(selectedClients, NoFilter)
	leaderC, _ := c.SetAverageBestLeader(selectedClients)
	leaderP, _ := p.SetAverageBestLeader(selectedClients)
	leaderN, _ := n.SetAverageBestLeader(selectedClients)

	_, protocol := protocolPr.GetCurrentOption()
	switch protocol {
	case "SwiftPaxos":
		UpdateClientInfo(leaderSp, quorumSp, sp, t, true, false, []Algorithm{c, p, n})
	case "Paxos":
		UpdateClientInfo(leaderP, nil, p, t, false, false, []Algorithm{sp, c, n})
	case "N²Paxos":
		UpdateClientInfo(leaderN, nil, n, t, false, true, []Algorithm{sp, c, p})
	case "CURP (N²Paxos)":
		UpdateClientInfo(leaderC, nil, c, t, true, true, []Algorithm{sp, p, n})
	}
}

func Faster(g, l float64) float64 {
	if g == 0 {
		if l == 0 {
			return 0
		} else if l > 0 {
			return math.Inf(1)
		}
	}
	return Div(Mul((g-l), 100), g)
}

func UpdateClientInfo(leader string, quorum Quorum, alg Algorithm, t *LatencyTable, printWorstL, printClosest bool, compareTo []Algorithm) {
	if quorum != nil {
		quorumPr.SetText(fmt.Sprintf("%v", quorum))
	} else {
		quorumPr.Clear()
	}
	latency := Average(alg, selectedClients, true)
	leaderPr.SetText(fmt.Sprintf("%v", leader))
	ls := fmt.Sprintf("%0.3f (best)\n%0.3f (worst)", latency, Average(alg, selectedClients, false))
	if !printWorstL {
		ls = fmt.Sprintf("%0.3f", latency)
	}
	latencyPr.SetText(ls)

	ls = ""
	longest := ""
	sort.Slice(selectedClients, func(i, j int) bool {
		return selectedClients[i] < selectedClients[j]
	})
	for _, c := range selectedClients {
		if utf8.RuneCountInString(c) > utf8.RuneCountInString(longest) {
			longest = c
		}
	}
	if compareTo != nil {
		ls = "         "
		for i := 0; i < utf8.RuneCountInString(longest); i++ {
			ls += " "
		}
		for _, a := range compareTo {
			ls += "\t  " + a.String()
		}
	}
	for _, c := range selectedClients {
		best := Average(alg, []string{c}, true)
		worst := Average(alg, []string{c}, false)

		if ls != "" {
			ls += "\n"
		}
		ls += c
		for i := 0; i < utf8.RuneCountInString(longest)-utf8.RuneCountInString(c); i++ {
			ls += " "
		}
		if best != worst {
			ls += "[#00BD56]"
		}
		ls += fmt.Sprintf("\t%7.3f[white]", best)

		for _, a := range compareTo {
			l := Average(a, []string{c}, true)
			if best <= l {
				ls += fmt.Sprintf("\t[green]%3.0f%%[white]", Faster(l, best))
			} else {
				ls += fmt.Sprintf("\t[red]%3.0f%%[white]", Faster(best, l))
			}
		}

		if best != worst {
			ls += "\n"
			for range longest {
				ls += " "
			}
			ls += fmt.Sprintf("\t[#FF424D]%7.3f[white]", worst)
			for _, a := range compareTo {
				l := Average(a, []string{c}, false)
				if worst <= l {
					ls += fmt.Sprintf("\t[green]%3.0f%%[white]", Faster(l, worst))
				} else {
					ls += fmt.Sprintf("\t[red]%3.0f%%[white]", Faster(worst, l))
				}
			}
		}
		if printClosest {
			ls += "\n"
			for range longest {
				ls += " "
			}
			ls += "\t[#668AAC]" + t.Site(Client(c).ClosestReplica(selectedReplicas, t)) + "[white]"
		}
	}
	clientsInfoPr.SetText(ls)
}

func NewExportBox(t *LatencyTable) {
	filename := "latency.txt"
	modal := func(p tview.Primitive, w, h int) tview.Primitive {
		return tview.NewGrid().SetColumns(0, w, 0).SetRows(0, h, 0).AddItem(p, 1, 1, 1, 1, 0, 0, true)
	}
	form := tview.NewForm().AddInputField("Save as", filename, 15, nil, func(f string) {
		filename = f
	})
	form.SetBorder(true).SetTitle("Export latency table")
	form.SetLabelColor(tcell.ColorWhite)
	form.SetFieldTextColor(tcell.ColorWhite)
	form.SetFieldBackgroundColor(tcell.ColorGrey)
	form.AddButton("Export", func() {
		if filename != "" {
			fs := strings.Split(filename, ".")
			if !(len(fs) >= 2 && fs[1] == "go") {
				// TODO: check errors
				t.Export(append(selectedReplicas, selectedClients...), filename)
				pages.SwitchToPage("main page")

				proxies := "proxies.conf"
				os.Remove(proxies)
				str := ""
				for _, r := range selectedReplicas {
					str += "server_alias " + t.IdOf(r) + "\n"
					for _, c := range selectedClients {
						ls := t.IdOf(Client(c).ClosestReplica(selectedReplicas, t))
						if ls == t.IdOf(r) {
							str += t.IdOf(c)
							if ls == t.IdOf(c) {
								str += " (local)"
							}
							str += "\n"
						}
					}
					str += "\n"
				}
				ioutil.WriteFile(proxies, []byte(str), 0644)

				quorum := "config.info"
				os.Remove(quorum)
				str = ""
				sp := NewSwiftPaxos(selectedReplicas, t)
				q, l, _ := sp.SetAverageBestFixedQuorumAndLeader(selectedClients, NoFilter)
				for _, r := range SliceOfQuorum(q) {
					if t.IdOf(r) == t.IdOf(l) {
						str += "l "
					}
					str += t.IdOf(r) + "\n"
				}
				ioutil.WriteFile(quorum, []byte(str), 0644)
			}
		}
	})
	form.AddButton("Cancel", func() {
		pages.SwitchToPage("main page")
	})
	pages.AddPage("export box", modal(form, 30, 8), true, false)
}

func NewImportBox(t *LatencyTable) {
	filename := "latency.txt"
	modal := func(p tview.Primitive, w, h int) tview.Primitive {
		return tview.NewGrid().SetColumns(0, w, 0).SetRows(0, h, 0).AddItem(p, 1, 1, 1, 1, 0, 0, true)
	}
	var form *tview.Form
	form = tview.NewForm().AddInputField("Filename", filename, 30, nil, func(f string) {
		filename = f
		if form.GetFormItemCount() >= 2 {
			form.RemoveFormItem(1)
		}
	})
	form.SetBorder(true).SetTitle("Import latency table")
	form.SetLabelColor(tcell.ColorWhite)
	form.SetFieldTextColor(tcell.ColorWhite)
	form.SetFieldBackgroundColor(tcell.ColorGrey)
	form.AddButton("Import", func() {
		if filename != "" {
			if t, err := NewLatencyTableFromFile(filename); err == nil {
				application.Stop()
				defer RunUI(t)
			} else {
				if form.GetFormItemCount() >= 2 {
					form.RemoveFormItem(1)
				}
				form.AddTextView("Error", err.Error(), 40, 2, false, true)
			}
		}
	})
	form.AddButton("Cancel", func() {
		pages.SwitchToPage("main page")
	})
	pages.AddPage("import box", modal(form, 40, 10), true, false)
}

func NewReplicaClientSelections(t *LatencyTable) *tview.Flex {
	rs := Regions("Replicas", t, func(rs map[string]struct{}) {
		i := 0
		selectedReplicas = make([]string, len(rs))
		for r := range rs {
			selectedReplicas[i] = r
			i++
		}
		Redraw(t)
	})
	cs := Regions("Clients", t, func(rs map[string]struct{}) {
		i := 0
		selectedClients = make([]string, len(rs))
		for r := range rs {
			selectedClients[i] = r
			i++
		}
		Redraw(t)
	})
	f := tview.NewFlex()
	f.AddItem(rs, 0, 1, true)
	f.AddItem(cs, 0, 1, false)

	f2 := tview.NewFlex()
	f2.SetDirection(tview.FlexRow)
	f2.AddItem(f, 0, 12, true)

	d := tview.NewDropDown()
	d.SetLabel("Protocol: ")
	d.AddOption("SwiftPaxos", func() {
		Redraw(t)
	})
	d.AddOption("Paxos", func() {
		Redraw(t)
	})
	d.AddOption("N²Paxos", func() {
		Redraw(t)
	})
	d.AddOption("CURP (N²Paxos)", func() {
		Redraw(t)
	})
	d.SetCurrentOption(0)
	d.SetLabelColor(tcell.ColorWhite)
	d.SetFieldTextColor(tcell.ColorWhite)
	d.SetFieldBackgroundColor(tcell.ColorGrey)
	styleU := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorGrey)
	styleS := tcell.StyleDefault.Foreground(tcell.ColorGrey).Background(tcell.ColorWhite)
	d.SetListStyles(styleU, styleS)

	worstL := tview.NewCheckbox()
	worstL.SetLabel("minimize worst latency ")
	worstL.SetChangedFunc(func(bool) {
		MinWorstLatency = worstL.IsChecked()
		Redraw(t)
	})
	worstL.SetChecked(true)
	worstL.SetLabelColor(tcell.ColorWhite)
	worstL.SetFieldTextColor(tcell.ColorWhite)
	worstL.SetFieldBackgroundColor(tcell.ColorGrey)

	dw := tview.NewFlex()
	dw.AddItem(d, 0, 1, false)
	dw.AddItem(worstL, 0, 1, false)

	f2.AddItem(dw, 0, 1, false)

	newTextView := func(label string) *tview.TextView {
		v := tview.NewTextView()
		v.SetChangedFunc(func() {
			application.Draw()
		})
		v.SetLabel(label + ": ")
		return v
	}

	f3 := tview.NewFlex()
	quorumPr = newTextView("quorum")
	leaderPr = newTextView("leader")
	latencyPr = newTextView("latency")
	f3.AddItem(quorumPr, 0, 4, false)
	f3.AddItem(leaderPr, 0, 4, false)
	f3.AddItem(latencyPr, 0, 2, false)
	f3.SetBorder(true)

	clientsInfoPr = newTextView("clients")
	clientsInfoPr.SetLabel("")
	clientsInfoPr.SetDynamicColors(true).SetRegions(true).SetBorder(true)
	f4 := tview.NewFlex()
	f4.SetBorder(true)
	f4.SetDirection(tview.FlexRow)
	f4.AddItem(f3, 0, 1, false)
	f4.AddItem(clientsInfoPr, 0, 3, false)

	f2.AddItem(f4, 0, 12, false)

	clientsPr = cs
	replicasPr = rs
	protocolPr = d

	return f2
}

func RunUI(t *LatencyTable) error {
	s := NewReplicaClientSelections(t)
	pages = tview.NewPages().AddPage("main page", s, true, true)
	NewExportBox(t)
	NewImportBox(t)
	application = tview.NewApplication().SetRoot(pages, true).EnableMouse(true)

	i := 0
	ps := []tview.Primitive{replicasPr, clientsPr, protocolPr}

	shown := false
	lt := tview.NewTextView()
	lt.SetChangedFunc(func() {
		application.Draw()
	})
	lt.SetLabel("Latency Table ")
	lt.SetText(t.String())

	s.SetMouseCapture(func(a tview.MouseAction, e *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if p, _ := pages.GetFrontPage(); p != "main page" {
			return a, nil
		}
		return a, e
	})

	application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if p, _ := pages.GetFrontPage(); p != "main page" {
			return event
		}

		switch key := event.Key(); key {
		case tcell.KeyEsc:
			if !shown {
				shown = true
				application.SetRoot(lt, true)
			} else {
				shown = false
				application.SetRoot(pages, true)
			}
		case tcell.KeyTab:
			i = (i + 1) % len(ps)
			application.SetFocus(ps[i])
			return nil
		}
		switch key := event.Rune(); key {
		case 'e':
			pages.ShowPage("export box")
		case 'i':
			pages.ShowPage("import box")
		case 'r':
			application.SetFocus(replicasPr)
		case 'c':
			application.SetFocus(clientsPr)
		case 'p':
			i, _ := protocolPr.GetCurrentOption()
			i++
			if i >= protocolPr.GetOptionCount() {
				i = 0
			}
			protocolPr.SetCurrentOption(i)
		case 'q':
			application.Stop()
		}
		return event
	})

	return application.Run()
}
