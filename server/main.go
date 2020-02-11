package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/Bmixo/btSearch/server/common"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	runtime.GOMAXPROCS(runtime.NumCPU())
	sniffer := common.NewSniffer()
	defer sniffer.Mon.Close()
	go sniffer.PrintLog()
	go sniffer.NewServerConn()
	go sniffer.Reboot()
	go sniffer.Metadata()
	sniffer.CheckSpeed()
}
