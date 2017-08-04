package mgoutil

import (
	log "github.com/Sirupsen/logrus"
	"labix.org/v2/mgo"
)

// MgoMode ...
var MgoMode = struct {
	Eventual  int
	Monotonic int
	Strong    int
}{
	Eventual:  0,
	Monotonic: 1,
	Strong:    2,
}

// SetMode changes the consistency mode for the session.
func SetMode(s *mgo.Session, mode int, refresh bool) {
	if mode < 0 || mode > 2 {
		log.Fatalln("Invalid mgo mode")
	}
	switch mode {
	case 0:
		s.SetMode(mgo.Eventual, refresh)
	case 1:
		s.SetMode(mgo.Monotonic, refresh)
	case 2:
		s.SetMode(mgo.Strong, refresh)
	}
}
