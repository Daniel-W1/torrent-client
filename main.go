package main

import (
	"sync"
	"time"
	"torrent-dsp/leecher"
	"torrent-dsp/seeder"
)

func main() {
	var wg sync.WaitGroup
  	wg.Add(1)
  
  	go func() {
    	defer wg.Done()
    	seeder.SeederMain()
  	}()


	time.Sleep(5 * time.Second)
	leecher.Leech("./torrent-files/debian-11.6.0-amd64-netinst.iso.torrent")

	wg.Wait()
}
