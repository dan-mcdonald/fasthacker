package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/dan-mcdonald/fasthacker/internal/sync"
	"github.com/hashicorp/go-metrics/prometheus"
)

func main() {
	fmt.Println("hacker-sync starting")
	sink, err := prometheus.NewPrometheusSink()
	if err != nil {
		log.Fatal(err)
	}
	chInterrupt := make(chan os.Signal, 1)
	signal.Notify(chInterrupt, os.Interrupt)
	synk := sync.NewSync("hacker.db", sink)
	go synk.Start()
	<-chInterrupt
	fmt.Println("interrupt received")
}
