package store

import (
	"bytes"
	"sync"

	"github.com/elastos/Elastos.ELA/common"

	"github.com/syndtr/goleveldb/leveldb"
)

// Ensure customID implement CustomID interface.
var _ CustomID = (*customID)(nil)

var BKTReservedCustomID = []byte("R")

type customID struct {
	batch
	sync.RWMutex
	db                *leveldb.DB
	b                 *leveldb.Batch
	cache             map[common.Uint256]uint32
	reservedCustomIDs map[string]struct{}
}

func NewCustomID(db *leveldb.DB) *customID {
	return &customID{
		db:                db,
		b:                 new(leveldb.Batch),
		cache:             make(map[common.Uint256]uint32),
		reservedCustomIDs: make(map[string]struct{}, 0),
	}
}

func (c *customID) Put(reservedCustomIDs []string) error {
	c.Lock()
	defer c.Unlock()
	batch := new(leveldb.Batch)
	if err := c.batchPut(reservedCustomIDs, batch); err != nil {
		return err
	}
	return c.db.Write(batch, nil)
}

func (c *customID) batchPut(reservedCustomIDs []string, batch *leveldb.Batch) error {
	// initialize cache.
	if len(c.reservedCustomIDs) == 0 {
		existedCustomIDs, err := c.getReservedCustomIDsFromDB()
		if err != nil {
			return err
		}
		c.reservedCustomIDs = existedCustomIDs
	}
	// add new custom ID into cache.
	for _, id := range reservedCustomIDs {
		c.reservedCustomIDs[id] = struct{}{}
	}

	// store reserved custom ID.
	w := new(bytes.Buffer)
	err := common.WriteUint32(w, uint32(len(c.reservedCustomIDs)+len(reservedCustomIDs)))
	if err != nil {
		return err
	}
	for k, _ := range c.reservedCustomIDs {
		if err := common.WriteVarString(w, k); err != nil {
			return err
		}
	}
	for _, id := range reservedCustomIDs {
		if err := common.WriteVarString(w, id); err != nil {
			return err
		}
	}
	batch.Put(BKTReservedCustomID, w.Bytes())
	return nil
}

func (c *customID) BatchPut(reservedCustomIDs []string, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPut(reservedCustomIDs, batch)
}

func (c *customID) GetReservedCustomIDs() (map[string]struct{}, error) {
	c.RLock()
	defer c.RUnlock()
	return c.getReservedCustomIDs()
}

func (c *customID) getReservedCustomIDs() (map[string]struct{}, error) {
	if len(c.reservedCustomIDs) != 0 {
		return c.reservedCustomIDs, nil
	}

	ids, err := c.getReservedCustomIDsFromDB()
	if err != nil {
		return nil, err
	}
	// refresh the cache.
	c.reservedCustomIDs = ids
	return ids, nil
}

func (c *customID) getReservedCustomIDsFromDB() (map[string]struct{}, error) {
	var val []byte
	val, err := c.db.Get(BKTReservedCustomID, nil)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(val)
	count, err := common.ReadUint32(r)
	if err != nil {
		return nil, err
	}
	reservedCustomIDs := make(map[string]struct{}, 0)
	for i := uint32(0); i < count; i++ {
		id, err := common.ReadVarString(r)
		if err != nil {
			return nil, err
		}
		reservedCustomIDs[id] = struct{}{}
	}
	return reservedCustomIDs, nil
}

func (c *customID) Close() error {
	c.Lock()
	return nil
}

func (c *customID) Clear() error {
	c.Lock()
	defer c.Unlock()
	c.b.Delete(BKTReservedCustomID)
	return c.db.Write(c.b, nil)
}

func (c *customID) Commit() error {
	return c.db.Write(c.b, nil)
}

func (c *customID) Rollback() error {
	c.b.Reset()
	return nil
}

func (c *customID) CommitBatch(batch *leveldb.Batch) error {
	return c.db.Write(batch, nil)
}

func (c *customID) RollbackBatch(batch *leveldb.Batch) error {
	batch.Reset()
	return nil
}
