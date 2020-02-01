package main

import (
	"runtime"

	"github.com/Bmixo/btSearch/server/common"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	sniffer := common.NewSniffer()
	defer sniffer.Mon.Close()
	go sniffer.PrintLog()
	go sniffer.NewServerConn()
	go sniffer.Reboot()
	go sniffer.Metadata()
	sniffer.CheckSpeed()
}
