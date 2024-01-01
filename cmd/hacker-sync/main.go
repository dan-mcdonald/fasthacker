package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/dan-mcdonald/fasthacker/internal/sync"
)

func main() {
	fmt.Println("hacker-sync starting")
	chInterrupt := make(chan os.Signal, 1)
	signal.Notify(chInterrupt, os.Interrupt)
	synk := sync.Sync{DBPath: "hacker.db"}
	go synk.Start()
	<-chInterrupt
	fmt.Println("interrupt received")
}
