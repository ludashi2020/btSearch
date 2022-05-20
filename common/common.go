package common

import (
	"github.com/Bmixo/btSearch/common/config"
	"github.com/Bmixo/btSearch/common/db"
)

var Commons *MCommons

type MCommons struct {
	DB     *db.MDB
	Config *config.MConfig
}

func Init() {
	Commons = &MCommons{
		Config: config.Init(),
		DB:     db.Init(),
	}
}
