package common

import (
	"github.com/Bmixo/btSearch/common/db"
)

var Commons *MCommons

type MCommons struct {
	DB *db.MDB
}

func Init() {
	Commons = &MCommons{
		DB: db.Init(),
	}
}
