package main

import (
	"github.com/Bmixo/btSearch/server/common"
)

func main() {

	self := common.NewSniffer()
	defer self.Mon.Close()
	go self.PrintLog()
	go self.NewServerConn()
	go self.Reboot()
	go self.Metadata()
	self.CheckSpeed()
}
