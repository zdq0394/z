package mgoutil

import (
	"testing"

	"labix.org/v2/mgo"
)

type testCollectionSet struct {
	StudentsColl Collection      `coll:"students"`
	ClassesColl  *mgo.Collection `coll:"classes"`
}

func TestMgoOpen(t *testing.T) {

	var ret testCollectionSet
	session, err := Open(&ret, &Config{Host: "100.64.0.34", DB: "test"})
	if err != nil {
		t.Fatal("Open session failed:", err)
	}
	defer session.Close()

	if ret.StudentsColl.Name != "students" {
		t.Fatal(`ret.studentsColl.Name != "students"`)
	}
	if ret.ClassesColl.Name != "classes" {
		t.Fatal(`ret.classesColl.Name != "classes"`)
	}
}
