package udb

import (
	"testing"

	"github.com/michelia/ulog"
)

func TestNew(t *testing.T) {
	d := Open(ulog.NewConsole(), ":memory:", 5)
	tab := d.New("table_test")
	tab.SetRaw("k", "dfsddf", 0)
	v, _ := tab.GetRaw("k")
	tab.slog.Print(*v)
	d.Close()
}
