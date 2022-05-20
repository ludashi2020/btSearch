package main

import (
	"github.com/Bmixo/btSearch/service"
)

func main() {
	service.InitWorker()
	m := service.NewWorkerServer()
	go m.PrintLog()
	go m.FindNode()
	go m.GenerNodes()
	go m.AutoSendFindNode()
	go m.HandleMsg()
	go m.Refresh()
	go m.Metadata()
	m.Server()
}
