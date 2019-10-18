package main

import (
	"log"

	"runtime"

	"github.com/Bmixo/btSearch/server/common"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	sniffer := common.NewSniffer()
	defer sniffer.Mon.Close()
	go sniffer.PrintLog()
	log.Println("Wait for Connect...")
	go sniffer.NewServerConn()
	go sniffer.Reboot()
	go sniffer.Metadata()
	go sniffer.CheckSpeed()
	hold := make(chan bool)
	<-hold
}
