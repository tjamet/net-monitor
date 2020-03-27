package main

import (
	"time"

	"github.com/tjamet/net-monitor/pkg/monitor"
)

func main() {

	elastic, err := monitor.NewElasticIndexer()
	if err != nil {
		panic(err)
	}
	m := monitor.Monitor{
		Indexer:     elastic,
		Locate:      monitor.CurrentLocation,
		SpeedTester: &monitor.SpeedTest{},
	}
	go m.StartLocationUpdate(24 * time.Hour)
	m.StartSpeedTest(1 * time.Hour)
}
