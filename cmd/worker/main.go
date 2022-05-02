package main

import (
	"github.com/Bmixo/btSearch/common"
)

func main() {
	common.InitWorker()
	m := common.NewWorkerServer()
	go m.PrintLog()
	go m.FindNode()
	go m.GenerNodes()
	go m.AutoSendFindNode()
	go m.HandleMsg()
	go m.Refresh()
	go m.Metadata()
	m.Server()
}
