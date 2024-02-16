package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/dan-mcdonald/fasthacker/internal/sync"
	"github.com/dan-mcdonald/fasthacker/internal/web"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	fmt.Println("hacker-sync starting")

	chInterrupt := make(chan os.Signal, 1)
	signal.Notify(chInterrupt, os.Interrupt)
	synk := sync.NewSync("hacker.db")
	go synk.Start()
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("starting metrics server http://localhost:9999/metrics")
		log.Fatal(http.ListenAndServe("localhost:9999", nil))
	}()
	go web.Start()
	<-chInterrupt
	fmt.Println("interrupt received")
}
