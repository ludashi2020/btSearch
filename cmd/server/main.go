package main

import (
	"github.com/Bmixo/btSearch/common"
	"github.com/Bmixo/btSearch/model"
	"github.com/Bmixo/btSearch/service"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	service.InitConfig()
	common.Init()
	model.Init()
	service.InitServer()
	go http.ListenAndServe("0.0.0.0:6060", nil)
	self := service.NewSniffer()
	go self.PrintLog()
	go self.NewServerConn()
	go self.Refresh()
	self.CheckSpeed()
}
