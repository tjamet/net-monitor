package monitor

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testRoundTripper struct {
	t *testing.T
}

func (t *testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	b, err := ioutil.ReadAll(r.Body)
	switch r.URL.Path {
	case "/_mapping":
		assert.NoError(t.t, err)
		assert.Equal(t.t, fmt.Sprintf(mappingFormat, "speed-test", version), string(b))
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(strings.NewReader(`{"status":"ok"}`)),
		}, nil
	case "/speed-test-v1-2020/_doc/1234":
		assert.NoError(t.t, err)
		assert.JSONEq(t.t, `{
			"result": {
			  "type": "",
			  "timestamp": "2020-03-25T20:00:17Z",
			  "ping": {
				"jitter": 0,
				"latency": 0
			  },
			  "download": {
				"bandwidth": 0,
				"bytes": 0,
				"elapsed": 0
			  },
			  "upload": {
				"bandwidth": 0,
				"bytes": 0,
				"elapsed": 0
			  },
			  "packetLoss": 0,
			  "isp": "",
			  "interface": {
				"internalIp": "",
				"name": "",
				"macAddr": "",
				"isVpn": false,
				"externalIp": ""
			  },
			  "server": {
				"id": 0,
				"name": "",
				"location": "",
				"country": "",
				"host": "",
				"port": 0
			  },
			  "result": {
				"id": "1234",
				"url": ""
			  }
			}
		  }`, string(b))
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(strings.NewReader(`{"status":"ok"}`)),
		}, nil
	case "/_cluster/settings", "/_template/speed-test":
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(strings.NewReader(`{"status":"ok"}`)),
		}, nil
	default:
		t.t.Errorf("Unexpected call to HTTP endpoint %s", r.URL.Path)
		return nil, fmt.Errorf("Unexpected call to HTTP endpoint %s", r.URL.Path)
	}
}

func TestIndex(t *testing.T) {
	oldTransport := http.DefaultTransport
	http.DefaultTransport = &testRoundTripper{t}
	defer func() {
		http.DefaultTransport = oldTransport
	}()

	indexer, err := NewElasticIndexer()
	assert.NoError(t, err)
	assert.NotNil(t, indexer)
	assert.NoError(t, indexer.Index(&LocatedSpeed{
		Result: &TestResult{
			Header: Header{
				Timestamp: "2020-03-25T20:00:17Z",
			},
			Result: Result{
				ID: "1234",
			},
		},
	}))
}
