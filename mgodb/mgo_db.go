package mgodb

import (
	"reflect"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"labix.org/v2/mgo"
)

// Dial establishes a new session to the cluster identified by the given seed
// server(s). The session will enable communication with all of the servers in
// the cluster, so the seed servers are used only to find out about the cluster
// topology.
func Dial(host string, mode int, syncTimeoutInS int64) (session *mgo.Session, err error) {
	session, err = mgo.Dial(host)
	if err != nil {
		log.Errorln("Connect MongoDB failed:", err, "- host:", host)
		return
	}

	if mode <= 2 && mode >= 0 {
		SetMode(session, mode, true)
	}
	if syncTimeoutInS != 0 {
		session.SetSyncTimeout(time.Duration(int64(time.Second) * syncTimeoutInS))
	}
	return
}

//Config ...
type Config struct {
	Host           string `json:"host"`
	DB             string `json:"db"`
	Mode           int    `json:"mode"`
	SyncTimeoutInS int64  `json:"timeout"`
}

//Open a mgon.Session
func Open(ret interface{}, cfg *Config) (session *mgo.Session, err error) {

	session, err = Dial(cfg.Host, cfg.Mode, cfg.SyncTimeoutInS)
	if err != nil {
		return
	}

	db := session.DB(cfg.DB)
	err = InitCollections(ret, db)
	if err != nil {
		session.Close()
		session = nil
	}
	return
}

// InitCollections initialize a set of collections
func InitCollections(ret interface{}, db *mgo.Database) (err error) {

	v := reflect.ValueOf(ret)
	if v.Kind() != reflect.Ptr {
		log.Errorln("Open: ret must be a pointer")
		return syscall.EINVAL
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		log.Errorln("Open: ret must be a struct pointer")
		return syscall.EINVAL
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Tag == "" {
			continue
		}
		coll := sf.Tag.Get("coll")
		if coll == "" {
			continue
		}
		switch elem := v.Field(i).Addr().Interface().(type) {
		case *Collection:
			elem.Collection = db.C(coll)
		case **mgo.Collection:
			*elem = db.C(coll)
		default:
			log.Errorln("Open: coll must be *mgo.Collection or mgo.Collection")
			return syscall.EINVAL
		}
	}
	return
}
