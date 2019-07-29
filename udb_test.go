package udb

import (
	"testing"

	"github.com/michelia/ulog"
)

func TestNew(t *testing.T) {
	d := Open(ulog.NewConsole(), ":memory:", 5)
	tab := d.New("table_test")
	tab.SetRaw("k", "dfsddf", 0)
	tab.slog.Print(tab.GetRaw("k"))
	d.Close()
}
