package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/Bmixo/btSearch/server/common"
)

func main() {
	go http.ListenAndServe("0.0.0.0:6060", nil)
	self := common.NewSniffer()
	defer self.Mon.Close()
	go self.PrintLog()
	go self.NewServerConn()
	go self.Reboot()
	go self.Metadata()
	self.CheckSpeed()
}
