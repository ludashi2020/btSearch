package torrent

import (
	"context"
	"errors"
	"github.com/Bmixo/btSearch/common"
	"github.com/Bmixo/btSearch/common/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type MTorrent struct {
}

type FileServer struct {
	Path   []interface{} `bson:"path"`
	Length int64         `bson:"length"`
}

type BitTorrent struct {
	ID         primitive.ObjectID `bson:"_id"`
	InfoHash   string             `bson:"infohash"`
	Name       string             `bson:"name"`
	Extension  string             `bson:"extension"`
	Files      []FileServer       `bson:"files"`
	Length     int64              `bson:"length"`
	CreateTime int64              `bson:"create_time"`
	LastTime   int64              `bson:"last_time"`
	Hot        int                `bson:"hot"`
	FileType   string             `bson:"category"`
	KeyWord    []string           `bson:"key_word"`
}

type TorrentInfo struct {
	ID        primitive.ObjectID `json:"_id"`
	InfoHash  string             `json:"infohash"`
	Category  string             `json:"category"`
	Name      string             `json:"name"`
	Extension string             `json:"extension"`
	Files     []struct {
		Length int64    `json:"length"`
		Path   []string `json:"path"`
	} `json:"files"`
	Length     int64    `json:"length"`
	Hot        int64    `json:"hot"`
	CreateTime int64    `json:"create_time"`
	LastTime   int64    `json:"last_time"`
	KeyWord    []string `json:"key_word"`
}

func (m *MTorrent) FindTorrent(id string) (data *TorrentInfo, err error) {
	for _, j := range id {
		if !((48 <= j && j <= 57) || (65 <= j && j <= 90) || (97 <= j && j <= 122)) {
			return nil, errors.New("database inject")
		}
	}

	data = &TorrentInfo{}
	c := common.Commons.DB.DBEngine.Collection(config.Config.MongoDatabase.Collection)
	idx, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	err = c.FindOne(context.Background(), bson.M{"_id": idx}).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (m *MTorrent) GetStatus() (result int64, err error) {
	c := common.Commons.DB.DBEngine.RunCommand(context.Background(), bson.M{"collStats": config.Config.MongoDatabase.Collection})
	var document bson.M
	err = c.Decode(&document)
	if err != nil {
		return 0, err
	}
	if v, ok := document["count"].(int32); ok {
		result = int64(v)
	}
	return
}

func (m *MTorrent) AddTorrentHot(id string) (err error) {

	c := common.Commons.DB.DBEngine.Collection(config.Config.MongoDatabase.Collection)
	idx, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = c.UpdateOne(
		context.Background(),
		bson.M{
			"_id": idx,
		},
		bson.D{
			{"$inc", bson.D{
				{"hot", int64(1)},
			}},
			{"$set", bson.D{
				{"last_time", time.Now().Unix()},
			}},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (m *MTorrent) InsertTorrent(data BitTorrent) (err error) {
	_, err = common.Commons.DB.DBEngine.Collection(config.Config.MongoDatabase.Collection).InsertOne(context.Background(), &data)
	if err != nil {
		return err
	}
	return nil
}

func (m *MTorrent) FindTorrentByHash(infoHash string) (data *TorrentInfo, err error) {
	data = &TorrentInfo{}
	c := common.Commons.DB.DBEngine.Collection(config.Config.MongoDatabase.Collection)
	err = c.FindOne(context.Background(), bson.M{"infohash": infoHash}).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

//func (m *Server) syncmongodb(data bitTorrent) (err error) {
//
//	c := m.Mon.DB(config.Config.MongoDatabase.Database).C(config.Config.MongoDatabase.Collection)
//	err = c.Insert(data)
//	<-m.mongoLimit
//	return
//}

//func (m *Server) findHash(infoHash string) (result map[string]interface{}, err error) {
//	if redisEnable {
//		val, redisErr := m.RedisClient.Get(infoHash).Result()
//		if redisErr == redis.Nil {
//			c := m.Mon.DB(config.Config.MongoDatabase.Database).C(config.Config.MongoDatabase.Collection)
//			selector := bson.M{"infohash": infoHash}
//			err = c.Find(selector).One(&result)
//			if result != nil {
//				m.RedisClient.Set(infoHash, result["_id"].(bson.ObjectId), 0)
//			}
//			return
//		} else if redisErr != nil {
//			c := m.Mon.DB(config.Config.MongoDatabase.Database).C(config.Config.MongoDatabase.Collection)
//			selector := bson.M{"infohash": infoHash}
//			err = c.Find(selector).One(&result)
//		} else {
//			result["_id"] = bson.ObjectId(val)
//		}
//	} else {
//		c := m.Mon.DB(config.Config.MongoDatabase.Database).C(config.Config.MongoDatabase.Collection)
//		selector := bson.M{"infohash": infoHash}
//		err = c.Find(selector).One(&result)
//	}
//	<-m.mongoLimit
//	return
//}
//
//func (m *Server) syncmongodb(data bitTorrent) (err error) {
//
//	c := m.Mon.DB(config.Config.MongoDatabase.Database).C(config.Config.MongoDatabase.Collection)
//	err = c.Insert(data)
//	<-m.mongoLimit
//	return
//}

//func (m *Server) updateTimeHot(objectID bson.ObjectId) (err error) {
//
//	c := m.Mon.DB(config.Config.MongoDatabase.Database).C(config.Config.MongoDatabase.Collection)
//
//	data := make(map[string]interface{})
//	data["$inc"] = map[string]int{"hot": 1}
//	data["$set"] = map[string]int64{"last_time": time.Now().Unix()}
//
//	selector := bson.M{"_id": objectID}
//	err = c.Update(selector, data)
//	<-m.mongoLimit
//	return
//}
