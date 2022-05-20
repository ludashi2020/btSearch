package config

import (
	"os"
)

var Config *MConfig

type MConfig struct {
	MongoDatabase MongoDatabaseConfig
}
type MongoDatabaseConfig struct {
	Addr       string
	Database   string
	Collection string
}

func Init() *MConfig {
	var err error
	Config, err = LoadConfConfig()
	if err != nil {
		panic(err.Error())
	}
	return Config
}

func LoadConfConfig() (config *MConfig, err error) {
	config = &MConfig{
		MongoDatabase: MongoDatabaseConfig{
			Addr:       os.Getenv("MongoDatabaseAddr"),
			Database:   os.Getenv("mongoDatabase"),
			Collection: os.Getenv("mongoCollection"),
		},
	}
	return config, nil
}
