package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testTester struct {
	t              *testing.T
	expectedServer int
}

func (t testTester) ListServers() ([]Server, error) {
	fd, err := os.Open("server-list.json")
	assert.NoError(t.t, err)

	list := ServerList{}
	assert.NoError(t.t, json.NewDecoder(fd).Decode(&list))
	return list.Servers, nil
}

func (t testTester) Test(server *Server) (*TestResult, error) {
	assert.Equal(t.t, t.expectedServer, server.ID)
	fd, err := os.Open("result.json")
	assert.NoError(t.t, err)

	result := TestResult{}
	assert.NoError(t.t, json.NewDecoder(fd).Decode(&result))
	return &result, nil
}

func TestParseServer(t *testing.T) {
	fd, err := os.Open("server-list.json")
	assert.NoError(t, err)

	list := ServerList{}
	assert.NoError(t, json.NewDecoder(fd).Decode(&list))
	assert.Equal(
		t,
		ServerList{
			Header: Header{
				Type:      "serverList",
				Timestamp: "2020-03-25T19:50:11Z",
			},
			Servers: []Server{
				Server{
					ID:       1695,
					Name:     "Adamo",
					Location: "Barcelona",
					Country:  "Spain",
					Host:     "speedtest.bcn.adamo.es",
					Port:     8080,
				},
				Server{
					ID:       21516,
					Name:     "Grupo MasMovil",
					Location: "Barcelona",
					Country:  "Spain",
					Host:     "speedtest-bcn.masmovil.com",
					Port:     8080,
				},
				Server{
					ID:       2254,
					Name:     "CSUC",
					Location: "Barcelona",
					Country:  "Spain",
					Host:     "speedtest.catnix.cat",
					Port:     8080,
				},
				Server{
					ID:       20672,
					Name:     "apfutura",
					Location: "Barcelona",
					Country:  "Spain",
					Host:     "msibitnap.apfutura.net",
					Port:     8080,
				},
				Server{
					ID:       5105,
					Name:     "Eurona Wireless Telecom S.A.",
					Location: "Barcelona",
					Country:  "Spain",
					Host:     "speedtest.eurona.net",
					Port:     8080,
				},
				Server{
					ID:       15199,
					Name:     "Meswifi",
					Location: "Barcelona",
					Country:  "Spain",
					Host:     "speedtest.meswifi.com",
					Port:     8080,
				},
				Server{
					ID:       14449,
					Name:     "Vodafone ES",
					Location: "Castelldefels",
					Country:  "Spain",
					Host:     "speedtestbarcelona2.vodafone.es",
					Port:     8080,
				},
				Server{
					ID:       23765,
					Name:     "GhoFi",
					Location: "Barcelona",
					Country:  "Spain",
					Host:     "bcn-speedtest.ghofi.net",
					Port:     8080,
				},
				Server{
					ID:       19146,
					Name:     "Iguana",
					Location: "Igualada",
					Country:  "Spain",
					Host:     "speedtest.iguana.cat",
					Port:     8080,
				},
				Server{
					ID:       31702,
					Name:     "Cingles Comunicacions",
					Location: "Caldes de Montbuí",
					Country:  "Spain",
					Host:     "speedtest.cinglescomunicacions.com",
					Port:     8080,
				},
			},
		},
		list,
	)

	data, err := ioutil.ReadFile("server-list.json")
	assert.NoError(t, err)
	b := bytes.NewBuffer(nil)
	json.NewEncoder(b).Encode(list)
	assert.JSONEq(t, string(data), b.String())
}

func TestParseResult(t *testing.T) {
	fd, err := os.Open("result.json")
	assert.NoError(t, err)

	result := TestResult{}
	assert.NoError(t, json.NewDecoder(fd).Decode(&result))

	data, err := ioutil.ReadFile("result.json")
	assert.NoError(t, err)
	b := bytes.NewBuffer(nil)
	json.NewEncoder(b).Encode(result)
	assert.JSONEq(t, string(data), b.String())
}

func TestRun(t *testing.T) {
	s, err := Run(testTester{t, 19146}, "Not Exist", "Iguana")
	assert.NoError(t, err)
	fmt.Println(s)
}

func TestLocationUnmarshalJson(t *testing.T) {
	data, err := ioutil.ReadFile("ipapi.json")
	assert.NoError(t, err)
	l := Location{}
	assert.NoError(t, json.Unmarshal(data, &l))
	assert.Equal(
		t,
		Location{
			Country:            "ES",
			CountryCode:        "ES",
			CountryCodeISO3:    "ESP",
			CountryCapital:     "Madrid",
			CountryTLD:         ".es",
			CountryName:        "Spain",
			InEurope:           true,
			Postal:             "08014",
			Timezone:           "Europe/Madrid",
			UTCOffset:          "+0100",
			CountryCallingCode: "+34",
			Currency:           "EUR",
			CurrencyName:       "Euro",
			Languages:          "es-ES,ca,gl,eu,oc",
			ContryArea:         504782,
			ContryPopulation:   4.6505963e+07,
			ASN:                "AS12479",
			Org:                "Orange Espagne SA",
			GeoPoint: GeoPoint{
				Latitude:  4.3891,
				Longitude: 1.1611,
			},
		},
		l,
	)
}

func TestCurrentLocation(t *testing.T) {
	l, err := CurrentLocation()
	assert.NoError(t, err)
	assert.NotNil(t, l)
}
