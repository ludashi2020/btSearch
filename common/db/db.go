package db

import (
	"context"
	"github.com/Bmixo/btSearch/common/config"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *MDB

type MDB struct {
	DBEngine *mongo.Database
}

func Init() *MDB {
	DB = &MDB{}
	DB.mongodbInit()
	return DB
}

func (m *MDB) mongodbInit() {
	client, err := mongo.NewClient(options.Client().ApplyURI(
		config.Config.MongoDatabase.Addr,
	))
	if err != nil {
		log.Fatal(err)
	}
	for {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		err = client.Connect(ctx)
		if err != nil {
			time.Sleep(time.Second)
			log.Println(err)
			continue
		}
		//defer client.Disconnect(ctx)
		//
		m.DBEngine = client.Database(config.Config.MongoDatabase.Database)
		break
	}
}
