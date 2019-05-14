// Copyright 2017-2019 Lei Ni (nilei81@gmail.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build dragonboat_pebble

package logdb

// WARNING: pebble support is expermental, DO NOT USE IT IN PRODUCTION.

import (
	"bytes"

	"github.com/lni/dragonboat/raftio"
	"github.com/petermattis/pebble"
	"github.com/petermattis/pebble/db"
)

const (
	// LogDBType is the logdb type name
	LogDBType = "sharded-pebble"
)

type pebbleWriteBatch struct {
	wb    *pebble.Batch
	db    *pebble.DB
	wo    *db.WriteOptions
	count int
}

func (w *pebbleWriteBatch) Destroy() {
	w.wb.Close()
}

func (w *pebbleWriteBatch) Put(key []byte, val []byte) {
	w.wb.Set(key, val, w.wo)
	w.count++
}

func (w *pebbleWriteBatch) Delete(key []byte) {
	w.wb.Delete(key, w.wo)
	w.count++
}

func (w *pebbleWriteBatch) Clear() {
	w.wb = w.db.NewBatch()
	w.count = 0
}

func (w *pebbleWriteBatch) Count() int {
	return w.count
}

func newKVStore(dir string, wal string) (IKvStore, error) {
	return openPebbleDB(dir, wal)
}

type pebbleKV struct {
	db   *pebble.DB
	opts *db.Options
	ro   *db.IterOptions
	wo   *db.WriteOptions
}

func openPebbleDB(dir string, walDir string) (*pebbleKV, error) {
	plog.Warningf("pebble support is experimental, DO NOT USE IN PRODUCTION")
	lopts := db.LevelOptions{Compression: db.NoCompression}
	opts := &db.Options{
		Levels: []db.LevelOptions{lopts},
	}
	if len(walDir) > 0 {
		opts.WALDir = walDir
	}
	pdb, err := pebble.Open(dir, opts)
	if err != nil {
		return nil, err
	}
	ro := &db.IterOptions{}
	wo := &db.WriteOptions{Sync: true}
	return &pebbleKV{
		db:   pdb,
		ro:   ro,
		wo:   wo,
		opts: opts,
	}, nil
}

func (r *pebbleKV) Name() string {
	return "pebble"
}

// Close closes the RDB object.
func (r *pebbleKV) Close() error {
	if r.db != nil {
		r.db.Close()
	}
	r.db = nil
	return nil
}

func iteratorIsValid(iter *pebble.Iterator) bool {
	v := iter.Valid()
	if err := iter.Error(); err != nil {
		panic(err)
	}
	return v
}

func (r *pebbleKV) IterateValue(fk []byte, lk []byte, inc bool,
	op func(key []byte, data []byte) (bool, error)) error {
	iter := r.db.NewIter(r.ro)
	defer iter.Close()
	for iter.SeekGE(fk); iteratorIsValid(iter); iter.Next() {
		key := iter.Key()
		val := iter.Value()
		if inc {
			if bytes.Compare(key, lk) > 0 {
				return nil
			}
		} else {
			if bytes.Compare(key, lk) >= 0 {
				return nil
			}
		}
		cont, err := op(key, val)
		if err != nil {
			return err
		}
		if !cont {
			break
		}
	}
	return nil
}

func (r *pebbleKV) GetValue(key []byte,
	op func([]byte) error) error {
	val, err := r.db.Get(key)
	if err != nil && err != db.ErrNotFound {
		return err
	}
	return op(val)
}

func (r *pebbleKV) SaveValue(key []byte, value []byte) error {
	return r.db.Set(key, value, r.wo)
}

func (r *pebbleKV) DeleteValue(key []byte) error {
	return r.db.Delete(key, r.wo)
}

func (r *pebbleKV) GetWriteBatch(ctx raftio.IContext) IWriteBatch {
	if ctx != nil {
		wb := ctx.GetWriteBatch()
		if wb != nil {
			return ctx.GetWriteBatch().(*pebbleWriteBatch)
		}
	}
	return &pebbleWriteBatch{wb: r.db.NewBatch(), db: r.db, wo: r.wo}
}

func (r *pebbleKV) CommitWriteBatch(wb IWriteBatch) error {
	pwb, ok := wb.(*pebbleWriteBatch)
	if !ok {
		panic("unknown type")
	}
	return r.db.Apply(pwb.wb, r.wo)
}

func (r *pebbleKV) CommitDeleteBatch(wb IWriteBatch) error {
	return r.CommitWriteBatch(wb)
}

func (r *pebbleKV) RemoveEntries(fk []byte, lk []byte) error {
	iter := r.db.NewIter(r.ro)
	defer iter.Close()
	wb := r.GetWriteBatch(nil)
	for iter.SeekGE(fk); iteratorIsValid(iter); iter.Next() {
		if bytes.Compare(iter.Key(), lk) >= 0 {
			break
		}
		wb.Delete(iter.Key())
	}
	if wb.Count() > 0 {
		return r.CommitDeleteBatch(wb)
	}
	return nil
}

func (r *pebbleKV) Compaction(fk []byte, lk []byte) error {
	return r.db.Compact(fk, lk)
}