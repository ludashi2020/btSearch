package main

import (
	"github.com/Bmixo/btSearch/common"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	common.InitServer()
	go http.ListenAndServe("0.0.0.0:6060", nil)
	self := common.NewSniffer()
	defer self.Mon.Close()
	go self.PrintLog()
	go self.NewServerConn()
	go self.Refresh()
	self.CheckSpeed()
}
