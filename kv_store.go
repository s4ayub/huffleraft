package huffleraft

import (
	"github.com/dgraph-io/badger"
	"os"
)

type BadgerKV struct {
	kv *badger.KV
}

// NewBadgerKV returns a BadgerKV. If the directory
// in which the data will be stored does not exist,
// then that directory will be made for the user.
func NewBadgerKV(dir string) (*BadgerKV, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir

	bkv, err := badger.NewKV(&opts)
	if err != nil {
		return nil, err
	}

	bagderKV := &BadgerKV{
		kv: bkv,
	}
	return bagderKV, nil
}

// Get a value using a key from a BadgerKV
func (b *BadgerKV) Get(key []byte) ([]byte, error) {
	var item badger.KVItem
	err := b.kv.Get(key, &item)
	if err != nil {
		return nil, err
	}

	return item.Value(), nil
}

// Set a key-value pair in a BadgerKV
func (b *BadgerKV) Set(key, val []byte) error {
	return b.kv.Set(key, val, 0)
}

// Delete a key-value pair in a BadgerKV
func (b *BadgerKV) Delete(key []byte) error {
	return b.kv.Delete(key)
}
