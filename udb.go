package udb

import (
	"encoding/json"
	"os"
	"time"

	"github.com/michelia/ulog"
	"github.com/tidwall/buntdb"
)

var (
	ErrNotFound            = buntdb.ErrNotFound
	IndexFloat             = buntdb.IndexFloat
	IndexInt               = buntdb.IndexInt
	IndexJSON              = buntdb.IndexJSON
	IndexJSONCaseSensitive = buntdb.IndexJSONCaseSensitive
	IndexString            = buntdb.IndexString
)

type Tx = *buntdb.Tx

type DB struct {
	*buntdb.DB
	slog ulog.Logger
}

// Table value 是json的表
type Table struct {
	*buntdb.DB
	Pre         string // 即表名, 只是key的前缀
	slog        ulog.Logger
	DefautIndex string //  表默认的时间索引 字段是 updated
}

// Open
// autoShrinkMinSize 触发收缩的文件大小, 单位 M, 默认是5M
func Open(slog ulog.Logger, path string, autoShrinkMinSize int) *DB {
	d, err := buntdb.Open(path)
	if err != nil {
		os.Remove(path)
		slog.Fatal().Err(err).Msg("can't open file, so remove db and exit")
	}
	var config buntdb.Config
	if err := d.ReadConfig(&config); err != nil {
		slog.Fatal().Err(err).Msg("can't read db config")
	}
	if autoShrinkMinSize == 0 {
		autoShrinkMinSize = 5
	}
	config.AutoShrinkMinSize = autoShrinkMinSize * 1024 * 1024
	if err := d.SetConfig(config); err != nil {
		slog.Fatal().Err(err).Msg("can't set db config")
	}
	return &DB{
		DB:   d,
		slog: slog,
	}
}

// New 创建一个val是json的表 buntdb没有桶的概念, 这里使用前缀代替表
// name 表名, 或者桶名, 而这里只是key的前缀
// 参考: https://github.com/tidwall/buntdb/is/sues/47
func (d *DB) New(tableName string) *Table {
	slog := d.slog.With().Str("table", tableName).Logger()
	t := Table{
		DB:          d.DB,
		Pre:         tableName + ":",
		slog:        &slog,
		DefautIndex: tableName + ":index-updated",
	}
	err := t.DB.Update(func(tx Tx) error {
		return tx.CreateIndex(t.DefautIndex, t.Pre+"*", IndexJSON("updated"))
	})
	if err != nil {
		t.slog.Fatal().Err(err).Msg("can't CreateIndexUpdated")
	}
	return &t
}

func (t *Table) CreateIndex(name string, less ...func(a, b string) bool) {
	err := t.DB.Update(func(tx Tx) error {
		return tx.CreateIndex(t.Pre+"index-"+name, t.Pre+"*", less...)
	})
	if err != nil {
		t.slog.Fatal().Err(err).Msg("can't CreateIndex: " + name)
	}
}

// SetRaw 对buntdb.Tx封装
// ttl单位是 分钟
func (t *Table) SetRaw(key, val string, ttl int) error {
	err := t.DB.Update(func(tx Tx) error {
		if ttl > 0 {
			_, _, err := tx.Set(t.Pre+key, val, &buntdb.SetOptions{
				Expires: true,
				TTL:     time.Minute * time.Duration(ttl),
			})
			return err
		}
		_, _, err := tx.Set(t.Pre+key, val, nil)
		return err
	})
	return err
}

// SetJSON 包含了json.Marshal
// v 必须是对象指针
// ttl 单位是分钟
func (t *Table) Set(key string, v interface{}, ttl int) error {
	val, err := json.Marshal(v)
	if err != nil {
		return err
	}
	err = t.SetRaw(key, string(val), ttl)
	return err
}

// Delete
func (t *Table) Delete(key string) error {
	err := t.DB.Update(func(tx Tx) error {
		_, err := tx.Delete(t.Pre + key)
		return err
	})
	return err
}

// Get 获取key对应的val, 然后encoded val and stores the result in the value pointed to by v.
func (t *Table) Get(key string, v interface{}) error {
	var val string
	err := t.DB.View(func(tx Tx) error {
		var err error
		val, err = tx.Get(t.Pre + key)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(val), v)
	if err != nil {
		return err
	}
	return nil
}

func (t *Table) GetRaw(key string) (string, error) {
	var val string
	err := t.DB.View(func(tx Tx) error {
		var err error
		val, err = tx.Get(t.Pre + key)
		if err != nil {
			return err
		}
		return nil
	})
	return val, err
}

func (t *Table) GetAll() ([]string, error) {
	var vals []string
	err := t.DB.View(func(tx Tx) error {
		err := tx.Ascend(t.Pre+"index-updated", func(key, value string) bool {
			vals = append(vals, value)
			return true
		})
		return err
	})
	return vals, err
}

func (t *Table) GetFirst(index string, vi interface{}) error {
	var v *string
	err := t.DB.View(func(tx Tx) error {
		return tx.Ascend(index, func(key, val string) bool {
			v = &val
			return true
		})
	})
	if err != nil {
		return err
	}
	if v == nil {
		return ErrNotFound
	}
	err = json.Unmarshal([]byte(*v), vi)
	return err
}

func (t *Table) GetLast(index string, vi interface{}) error {
	var v *string
	err := t.DB.View(func(tx Tx) error {
		return tx.Descend(index, func(key, val string) bool {
			v = &val
			return true
		})
	})
	if err != nil {
		return err
	}
	if v == nil {
		return ErrNotFound
	}
	err = json.Unmarshal([]byte(*v), vi)
	return err
}
