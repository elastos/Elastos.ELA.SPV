package store

import (
	"errors"
	"sync"

	"github.com/elastos/Elastos.ELA.SPV/sdk"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Ensure addrs implement Addrs interface.
var _ TxTypes = (*txTypes)(nil)

type txTypes struct {
	sync.RWMutex
	db     *leveldb.DB
	filter *sdk.TxTypesFilter
}

func NewTxTypes(db *leveldb.DB) (*txTypes, error) {
	store := txTypes{db: db}

	addrs, err := store.getAll()
	if err != nil {
		return nil, err
	}
	store.filter = sdk.NewTxTypesFilter(addrs)

	return &store, nil
}

func (a *txTypes) GetFilter() *sdk.TxTypesFilter {
	a.Lock()
	defer a.Unlock()
	return a.filter
}

func (a *txTypes) Put(txType uint8) error {
	a.Lock()
	defer a.Unlock()

	if a.filter.ContainTxType(txType) {
		return nil
	}

	a.filter.AddTxType(txType)
	return a.db.Put(toKey(BKTTxTypes, txType), []byte{txType}, nil)
}

func (a *txTypes) GetAll() []uint8 {
	a.RLock()
	defer a.RUnlock()
	return a.filter.GetTxTypes()
}

func (a *txTypes) getAll() (txTypes []uint8, err error) {
	it := a.db.NewIterator(util.BytesPrefix(BKTTxTypes), nil)
	defer it.Release()
	for it.Next() {
		if len(it.Value()) != 1 {
			return nil, errors.New("invalid tx types")
		}
		txType := it.Value()[0]
		txTypes = append(txTypes, txType)
	}

	return txTypes, it.Error()
}

func (a *txTypes) Clear() error {
	a.Lock()
	defer a.Unlock()

	it := a.db.NewIterator(util.BytesPrefix(BKTTxTypes), nil)
	defer it.Release()
	batch := new(leveldb.Batch)
	for it.Next() {
		batch.Delete(it.Key())
	}
	return a.db.Write(batch, nil)
}

func (a *txTypes) Close() error {
	a.Lock()
	return nil
}
