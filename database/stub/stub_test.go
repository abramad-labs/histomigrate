package stub

import (
	"github.com/abramad-labs/histomigrate"
	"github.com/abramad-labs/histomigrate/source"
	"github.com/abramad-labs/histomigrate/source/stub"
	"testing"

	dt "github.com/abramad-labs/histomigrate/database/testing"
)

func Test(t *testing.T) {
	s := &Stub{}
	d, err := s.Open("")
	if err != nil {
		t.Fatal(err)
	}
	dt.Test(t, d, []byte("/* foobar migration */"))
}

func TestMigrate(t *testing.T) {
	s := &Stub{}
	d, err := s.Open("")
	if err != nil {
		t.Fatal(err)
	}

	stubMigrations := source.NewMigrations()
	stubMigrations.Append(&source.Migration{Version: 1, Direction: source.Up, Identifier: "CREATE 1"})
	stubMigrations.Append(&source.Migration{Version: 1, Direction: source.Down, Identifier: "DROP 1"})
	src := &stub.Stub{}
	srcDrv, err := src.Open("")
	if err != nil {
		t.Fatal(err)
	}
	srcDrv.(*stub.Stub).Migrations = stubMigrations
	m, err := migrate.NewWithInstance("stub", srcDrv, "", d)
	if err != nil {
		t.Fatal(err)
	}

	dt.TestMigrate(t, m)
}
