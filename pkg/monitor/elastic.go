package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	elastic "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

const (
	version       = "v1"
	mappingFormat = `
{
    "index_patterns": "%s-%s-*",
    "mappings" : {
        "_source": {
            "enabled": true
        },
        "dynamic_templates": [
            {
                "logtext": {
                    "match_mapping_type": "string",
                    "mapping": {
                        "type": "keyword"
                    }
                }
            }
        ],
        "properties": {
			"result": {
				"properties": {
					"timestamp": {
						"type" : "date"
					},
					"interface": {
						"properties": {
							"internalIp": {
								"type":"ip"
							},
							"externalIp": {
								"type":"ip"
							}
						}
					},
					"server": {
						"properties": {
							"ip": {
								"type":"ip"
							},
							"externalIp": {
								"type":"ip"
							}
						}
					}
				}
			},
			"location": {
				"properties": {
					"ip": {
						"type":"ip"
					},
					"geo_point": {
						"type" : "geo_point"
					}
				}
			}
        }
    }
}
`
	clusterSettingsFormat = `{
		"persistent": {
			"action.auto_create_index": "%s-%s-*" 
		}
	}`
)

type ElasticErrorCause struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type ElasticError struct {
	RootCause []ElasticErrorCause `json:"root_cause"`
	Status    int                 `json:"status"`
}

func (e ElasticError) Error() string {
	for _, e := range e.RootCause {
		return e.Reason
	}
	return ""
}

type ElasticIndexer struct {
	Client      *elastic.Client
	IndexPrefix string
}

type IndexResult struct {
	Result string        `json:"result"`
	Error  *ElasticError `json:"error"`
}

func NewElasticIndexer() (*ElasticIndexer, error) {

	indexPrefix := os.Getenv("ELASTIC_INDEX_PREFIX")
	if indexPrefix == "" {
		indexPrefix = "speed-test"
	}

	config := elastic.Config{
		Username:  os.Getenv("ELASTIC_USER"),
		Password:  os.Getenv("ELASTIC_PASSWORD"),
		Addresses: []string{os.Getenv("ELASTIC_HOST")},
	}
	c, err := elastic.NewClient(config)
	resp, err := esapi.IndicesPutTemplateRequest{
		Name: "speed-test",
		Body: strings.NewReader(fmt.Sprintf(mappingFormat, indexPrefix, version)),
	}.Do(context.Background(), c)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := IndexResult{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	if false {
		settingsResp, err := esapi.ClusterPutSettingsRequest{
			Body: strings.NewReader(fmt.Sprintf(clusterSettingsFormat, indexPrefix, version)),
		}.Do(context.Background(), c)
		if err != nil {
			return nil, err
		}
		settingsResp.Body.Close()
	}
	return &ElasticIndexer{c, indexPrefix}, err
}

func (e *ElasticIndexer) Index(results *LocatedSpeed) error {
	b := bytes.NewBuffer(nil)
	err := json.NewEncoder(b).Encode(results)
	if err != nil {
		return err
	}
	ts, err := time.Parse("2006-01-02T15:04:05Z07", results.Result.Header.Timestamp)
	index := fmt.Sprintf("%s-%s", e.IndexPrefix, version)
	if err == nil {
		//index += "-" + ts.Format("2006-01")
		index += "-" + ts.Format("2006")
	}
	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: results.Result.Result.ID,
		Body:       b,
		Refresh:    "true",
	}
	resp, err := req.Do(context.Background(), e.Client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result := IndexResult{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}
	return nil
}
