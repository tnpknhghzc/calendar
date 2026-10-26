package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/xuri/excelize/v2"
)

type Event struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Start       string `json:"start"`
	End         string `json:"end"`
	City        string `json:"city"`
	Venue       string `json:"venue"`
	Age         string `json:"age"`
	Features    string `json:"features"`
	Comment     string `json:"comment"`
	Status      string `json:"status"`
	SportStatus string `json:"sportStatus"`
	Category    string `json:"category"`
	Sheet       string `json:"sheet"`
}
type Season struct {
	SY int `json:"sy"`
	SM int `json:"sm"`
	EY int `json:"ey"`
	EM int `json:"em"`
}
type State struct {
	Events []Event `json:"events"`
	Season Season  `json:"season"`
	Today  string  `json:"today"`
}
type App struct {
	mu sync.Mutex
	st State
}

func NewApp() *App {
	a := &App{}
	a.load()
	return a
}

func krsk() *time.Location {
	if l, err := time.LoadLocation("Asia/Krasnoyarsk"); err == nil {
		return l
	}
	return time.FixedZone("KRAT", 7*3600)
}
func (a *App) Today() string { return time.Now().In(krsk()).Format("2006-01-02") }

func dataPath() string {
	d, _ := os.UserConfigDir()
	d = filepath.Join(d, "curling-calendar")
	os.MkdirAll(d, 0o755)
	return filepath.Join(d, "data.json")
}
func (a *App) load() {
	if b, err := os.ReadFile(dataPath()); err == nil {
		json.Unmarshal(b, &a.st)
	}
	if a.st.Season.SY == 0 {
		y := time.Now().In(krsk()).Year()
		a.st.Season = Season{y, 1, y, 12}
	}
	if a.st.Events == nil {
		a.st.Events = []Event{}
	}
}
func (a *App) save() {
	b, _ := json.MarshalIndent(a.st, "", " ")
	os.WriteFile(dataPath(), b, 0o644)
}
func (a *App) GetState() State {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.st
	s.Today = a.Today()
	return s
}
func (a *App) SaveEvent(e Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if strings.TrimSpace(e.Name) == "" || e.Start == "" {
		return errors.New("нужны название и дата")
	}
	if e.End == "" || e.End < e.Start {
		e.End = e.Start
	}
	if e.ID == "" {
		e.ID = strconv.FormatInt(time.Now().UnixNano(), 36)
		a.st.Events = append(a.st.Events, e)
	} else {
		for i := range a.st.Events {
			if a.st.Events[i].ID == e.ID {
				a.st.Events[i] = e
			}
		}
	}
	a.save()
	return nil
}
func (a *App) filter(keep func(Event) bool) {
	n := []Event{}
	for _, e := range a.st.Events {
		if keep(e) {
			n = append(n, e)
		}
	}
	a.st.Events = n
	a.save()
}
func (a *App) DeleteEvent(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.filter(func(e Event) bool { return e.ID != id })
}
func (a *App) DeleteAll() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.filter(func(Event) bool { return false })
}
func (a *App) DeleteMonth(ym string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.filter(func(e Event) bool { return !strings.HasPrefix(e.Start, ym) })
}
func (a *App) SetSeason(s Season) error {
	if s.SY*12+s.SM > s.EY*12+s.EM || s.SM < 1 || s.EM > 12 {
		return errors.New("начало сезона позже конца")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.st.Season = s
	a.save()
	return nil
}

// ---------- Excel import ----------
func mapCols(r []string) map[string]int {
	c := map[string]int{}
	set := func(k string, i int) {
		if _, ok := c[k]; !ok {
			c[k] = i
		}
	}
	for i, h := range r {
		l := strings.ToLower(strings.TrimSpace(h))
		has := func(s ...string) bool {
			for _, x := range s {
				if strings.Contains(l, x) {
					return true
				}
			}
			return false
		}
		switch {
		case l == "":
		case has("спорт"):
			set("sport", i)
		case has("статус"):
			set("status", i)
		case has("начал"):
			set("start", i)
		case has("конц", "оконч"):
			set("end", i)
		case has("дат", "срок"):
			set("date", i)
		case has("назв", "наимен", "мероприят"):
			set("name", i)
		case has("тип", "уровень"):
			set("type", i)
		case has("город", "место"):
			set("city", i)
		case has("объект"):
			set("venue", i)
		case has("возраст"):
			set("age", i)
		case has("особ"):
			set("feat", i)
		case has("коммент", "примеч"):
			set("comment", i)
		case has("категор"):
			set("cat", i)
		}
	}
	return c
}

var numRe = regexp.MustCompile(`(\d{1,4})[./-](\d{1,2})[./-](\d{2,4})`)
var txtRe = regexp.MustCompile(`(\d{1,2})(?:\s*[-–—]\s*(\d{1,2}))?\s+([а-яё]+)\.?\s*(\d{4})?`)
var months = map[string]int{"янв": 1, "фев": 2, "мар": 3, "апр": 4, "мая": 5, "май": 5, "июн": 6, "июл": 7, "авг": 8, "сен": 9, "окт": 10, "ноя": 11, "дек": 12}

func iso(y, m, d int) string {
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", y, m, d)
}
func parseDates(s string, defY int) (string, string) {
	s = strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 20000 {
		t := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(f))
		d := t.Format("2006-01-02")
		return d, d
	}
	var out []string
	for _, m := range numRe.FindAllStringSubmatch(s, -1) {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		c, _ := strconv.Atoi(m[3])
		var y, mo, d int
		switch {
		case a > 31: // yyyy-mm-dd
			y, mo, d = a, b, c
		case c < 100 && strings.Contains(m[0], "-") && !strings.Contains(m[0], "."): // mm-dd-yy (Excel)
			y, mo, d = c+2000, a, b
		default:
			y, mo, d = c, b, a
			if y < 100 {
				y += 2000
			}
		}
		if v := iso(y, mo, d); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		if m := txtRe.FindStringSubmatch(strings.ToLower(s)); m != nil && len(m[3]) >= 3 {
			if mo, ok := months[m[3][:6][:3]]; ok || true {
				if !ok {
					mo = months[string([]rune(m[3])[:3])]
				}
				y := defY
				if m[4] != "" {
					y, _ = strconv.Atoi(m[4])
				}
				d1, _ := strconv.Atoi(m[1])
				d2 := d1
				if m[2] != "" {
					d2, _ = strconv.Atoi(m[2])
				}
				if mo > 0 {
					return iso(y, mo, d1), iso(y, mo, d2)
				}
			}
		}
		return "", ""
	}
	return out[0], out[len(out)-1]
}

func normType(t string) string {
	l := strings.ToLower(t)
	for k, v := range map[string]string{"междунар": "Международные", "всеросс": "Всероссийские", "межрегион": "Межрегиональные", "регион": "Региональные", "муницип": "Муниципальные"} {
		if strings.Contains(l, k) {
			return v
		}
	}
	return "Региональные"
}

func (a *App) ImportExcel(b64 string) (int, error) {
	if i := strings.Index(b64, ","); i >= 0 && strings.HasPrefix(b64, "data:") {
		b64 = b64[i+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return 0, err
	}
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return 0, errors.New("не удалось открыть Excel-файл")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exist := map[string]bool{}
	for _, e := range a.st.Events {
		exist[e.Name+"|"+e.Start] = true
	}
	n := 0
	for _, sh := range f.GetSheetList() {
		rows, _ := f.GetRows(sh)
		hi := -1
		var col map[string]int
		for i, r := range rows {
			c := mapCols(r)
			_, hasName := c["name"]
			_, hasD := c["date"]
			_, hasS := c["start"]
			if hasName && (hasD || hasS) {
				hi, col = i, c
				break
			}
		}
		if hi < 0 {
			continue
		}
		for _, r := range rows[hi+1:] {
			g := func(k string) string {
				if i, ok := col[k]; ok && i < len(r) {
					return strings.TrimSpace(r[i])
				}
				return ""
			}
			name := g("name")
			if name == "" {
				continue
			}
			var s, e string
			if g("start") != "" {
				s, _ = parseDates(g("start"), a.st.Season.SY)
				_, e = parseDates(g("end"), a.st.Season.SY)
			} else {
				s, e = parseDates(g("date"), a.st.Season.SY)
			}
			if s == "" || exist[name+"|"+s] {
				continue
			}
			if e == "" || e < s {
				e = s
			}
			st := g("status")
			if st == "" || strings.Contains(strings.ToLower(st), "подтвержд") && !strings.Contains(strings.ToLower(st), "не ") {
				st = "Подтверждено"
			}
			ev := Event{ID: strconv.FormatInt(time.Now().UnixNano()+int64(n), 36), Name: name, Type: normType(g("type")), Start: s, End: e,
				City: g("city"), Venue: g("venue"), Age: g("age"), Features: g("feat"), Comment: g("comment"), Status: st,
				SportStatus: g("sport"), Category: g("cat"), Sheet: sh}
			a.st.Events = append(a.st.Events, ev)
			exist[name+"|"+s] = true
			n++
			sy, _ := strconv.Atoi(s[:4])
			sm, _ := strconv.Atoi(s[5:7])
			ey, _ := strconv.Atoi(e[:4])
			em, _ := strconv.Atoi(e[5:7])
			if sy*12+sm < a.st.Season.SY*12+a.st.Season.SM {
				a.st.Season.SY, a.st.Season.SM = sy, sm
			}
			if ey*12+em > a.st.Season.EY*12+a.st.Season.EM {
				a.st.Season.EY, a.st.Season.EM = ey, em
			}
		}
	}
	a.save()
	if n == 0 {
		return 0, errors.New("не найдено строк: нужны колонки «Название» и «Дата»")
	}
	return n, nil
}
