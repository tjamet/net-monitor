package monitor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	elastic "github.com/elastic/go-elasticsearch/v7"
)

type HardwareAddr net.HardwareAddr

func (a *HardwareAddr) UnmarshalJSON(data []byte) error {
	s := ""
	json.Unmarshal(data, &s)
	addr, err := net.ParseMAC(s)
	if err != nil {
		return err
	}
	*a = HardwareAddr(addr)
	return nil
}

func (a HardwareAddr) MarshalJSON() ([]byte, error) {
	addr := net.HardwareAddr(a)
	return []byte(strings.ToUpper(fmt.Sprintf(`"%s"`, addr.String()))), nil
}

type ElasticReporter struct {
	Client   elastic.Client
	Location *Location
}

type Monitor struct {
	Indexer     Indexer
	Locate      Locator
	SpeedTester Tester
	Preferred   []string
	Location    *Location
}

type Indexer interface {
	Index(*LocatedSpeed) error
}

type Locator func() (*Location, error)

type GeoPoint struct {
	// elasticsearch notation
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

func firstFloat64(vars ...interface{}) float64 {
	for _, v := range vars {
		if f, ok := v.(float64); ok {
			return f
		}
		if f, ok := v.(float32); ok {
			return float64(f)
		}
	}
	return 0
}

func (l *GeoPoint) UnmarshalJSON(b []byte) error {
	// unmarshalling ipapi.co format
	m := map[string]interface{}{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	l.Latitude = firstFloat64(m["latitude"], m["lat"])
	l.Longitude = firstFloat64(m["longitude"], m["lat"])
	return nil
}

type Location struct {
	Country            string   `json:"country"`
	CountryCode        string   `json:"country_code"`
	CountryCodeISO3    string   `json:"country_code_iso3"`
	CountryCapital     string   `json:"country_capital"`
	CountryTLD         string   `json:"country_tld"`
	CountryName        string   `json:"country_name"`
	InEurope           bool     `json:"in_eu"`
	Postal             string   `json:"postal"`
	Timezone           string   `json:"timezone"`
	UTCOffset          string   `json:"utc_offset"`
	CountryCallingCode string   `json:"country_calling_code"`
	Currency           string   `json:"currency"`
	CurrencyName       string   `json:"currency_name"`
	Languages          string   `json:"languages"`
	ContryArea         float64  `json:"country_area"`
	ContryPopulation   float64  `json:"country_population"`
	ASN                string   `json:"asn"`
	Org                string   `json:"org"`
	GeoPoint           GeoPoint `json:"geo_point"`
}

func (l *Location) UnmarshalJSON(b []byte) error {
	type Alias Location
	err := json.Unmarshal(b, (*Alias)(l))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &l.GeoPoint)
}

type RunDocument struct {
	ID         string     `json:"id"`
	TestResult TestResult `json:"results"`
}

type Header struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

type Server struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Country  string `json:"country"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	IP       net.IP `json:"ip,omitempty"`
}

type ServerList struct {
	Header
	Servers []Server `json:"servers"`
}

type Ping struct {
	Jitter  float64 `json:"jitter"`
	Latency float64 `json:"latency"`
}

type Speed struct {
	Bandwidth uint32 `json:"bandwidth"`
	Bytes     uint32 `json:"bytes"`
	Elapsed   uint32 `json:"elapsed"`
}

type Interface struct {
	InternalIP net.IP       `json:"internalIp"`
	Name       string       `json:"name"`
	MacAddr    HardwareAddr `json:"macAddr"`
	IsVPN      bool         `json:"isVpn"`
	ExternalIP net.IP       `json:"externalIp"`
}

type Result struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type TestResult struct {
	Header
	Ping       Ping      `json:"ping"`
	Download   Speed     `json:"download"`
	Upload     Speed     `json:"upload"`
	PacketLoss float64   `json:"packetLoss"`
	ISP        string    `json:"isp"`
	Interface  Interface `json:"interface"`
	Server     Server    `json:"server"`
	Result     Result    `json:"result"`
}

type LocatedSpeed struct {
	Result   *TestResult `json:"result"`
	Location *Location   `json:"location,omitempty"`
}

type Tester interface {
	ListServers() ([]Server, error)
	Test(*Server) (*TestResult, error)
}

type SpeedTest struct {
	Executable string
}

func (s *SpeedTest) run(args ...string) (io.Reader, error) {
	if s == nil {
		s = &SpeedTest{}
	}
	out := bytes.NewBuffer(nil)
	if s.Executable == "" {
		s.Executable = "speedtest"
	}
	c := exec.Command(s.Executable, args...)
	c.Stdout = out
	err := c.Run()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SpeedTest) ListServers() ([]Server, error) {
	out, err := s.run("-L", "-fjson")
	if err != nil {
		return nil, err
	}
	list := ServerList{}
	err = json.NewDecoder(out).Decode(&list)
	if err != nil {
		return nil, err
	}
	return list.Servers, nil
}

func (s *SpeedTest) Test(server *Server) (*TestResult, error) {
	out, err := s.run("--accept-license", "--accept-gdpr", fmt.Sprintf("-s%d", server.ID), "-fjson")
	if err != nil {
		return nil, nil
	}
	results := &TestResult{}
	err = json.NewDecoder(out).Decode(&results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func oneOf(candidate string, candidates []string) bool {
	for _, c := range candidates {
		if candidate == c {
			return true
		}
	}
	return false
}

func Run(s Tester, preferred ...string) (*TestResult, error) {
	servers, err := s.ListServers()
	if err != nil {
		return nil, nil
	}

	var server *Server
	if len(preferred) > 0 {
		svs := []Server{}
		for _, s := range servers {
			if oneOf(s.Name, preferred) {
				svs = append(svs, s)
			}
		}
		if len(svs) > 0 {
			servers = svs
		}
	}
	if len(servers) > 0 {
		server = &servers[rand.Intn(len(servers))]
	}
	if server == nil {
		return nil, errors.New("did not find any matching server")
	}
	return s.Test(server)
}

func CurrentLocation() (*Location, error) {
	resp, err := http.Get("https://ipapi.co/json/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	l := Location{}
	return &l, json.NewDecoder(resp.Body).Decode(&l)
}

func IPLocation(ip string) (*Location, error) {
	resp, err := http.Get("https://ipapi.co/" + ip + "/json/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	l := Location{}
	return &l, json.NewDecoder(resp.Body).Decode(&l)
}

func (m *Monitor) updateLocation() {
	l, err := m.Locate()
	if err == nil {
		m.Location = l
	}
}

func (m *Monitor) StartLocationUpdate(d time.Duration) {
	for range time.Tick(d) {
		m.updateLocation()
	}
}

func (m *Monitor) runSpeedTest() {
	log.Printf("starting speedtest")
	results, err := Run(m.SpeedTester, m.Preferred...)
	if err != nil {
		log.Printf("error running speed test: %v", err)
		return
	}
	doc := &LocatedSpeed{
		Location: m.Location,
		Result:   results,
	}
	b := bytes.NewBuffer(nil)
	if json.NewEncoder(b).Encode(doc) == nil {
		log.Println(b.String())
	}
	err = m.Indexer.Index(doc)
	if err != nil {
		log.Printf("error indexing test result: %v", err)
	}
}

func (m *Monitor) StartSpeedTest(d time.Duration) {
	log.Printf("getting location")
	m.updateLocation()
	m.runSpeedTest()
	for range time.Tick(d) {
		m.runSpeedTest()
	}
}
