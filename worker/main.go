package main

import (
	"github.com/Bmixo/btSearch/worker/common"
)

func main() {
	self := common.NewServer()
	go self.PrintLog()
	go self.FindNode()
	go self.GenerNodes()
	go self.AutoSendFindNode()
	go self.HandleMsg()
	self.Server()
}
