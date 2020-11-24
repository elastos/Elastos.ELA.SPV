package store

import (
	"bytes"
	"sync"

	"github.com/elastos/Elastos.ELA/common"

	"github.com/syndtr/goleveldb/leveldb"
)

// Ensure customID implement CustomID interface.
var _ CustomID = (*customID)(nil)

var BKTReservedCustomID = []byte("RS")
var BKTReceivedCustomID = []byte("RC")

type customID struct {
	batch
	sync.RWMutex
	db                *leveldb.DB
	b                 *leveldb.Batch
	cache             map[common.Uint256]uint32
	reservedCustomIDs map[string]struct{}
	receivedCustomIDs map[string]common.Uint168
}

func NewCustomID(db *leveldb.DB) *customID {
	return &customID{
		db:                db,
		b:                 new(leveldb.Batch),
		cache:             make(map[common.Uint256]uint32),
		reservedCustomIDs: make(map[string]struct{}, 0),
	}
}

func (c *customID) PutReservedCustomIDs(reservedCustomIDs []string) error {
	c.Lock()
	defer c.Unlock()
	batch := new(leveldb.Batch)
	if err := c.batchPutReservedCustomIDs(reservedCustomIDs, batch); err != nil {
		return err
	}
	return c.db.Write(batch, nil)
}

func (c *customID) PutReceivedCustomIDs(reservedCustomIDs []string, did common.Uint168) error {
	c.Lock()
	defer c.Unlock()
	batch := new(leveldb.Batch)
	if err := c.batchPutReceivedCustomIDs(reservedCustomIDs, did, batch); err != nil {
		return err
	}
	return c.db.Write(batch, nil)
}

func (c *customID) batchPutReservedCustomIDs(reservedCustomIDs []string, batch *leveldb.Batch) error {
	// initialize cache.
	if len(c.reservedCustomIDs) == 0 {
		existedCustomIDs, err := c.getReservedCustomIDsFromDB()
		if err != nil {
			return err
		}
		c.reservedCustomIDs = existedCustomIDs
	}
	// add new reserved custom ID into cache.
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

func (c *customID) batchPutReceivedCustomIDs(receivedCustomIDs []string,
	did common.Uint168, batch *leveldb.Batch) error {
	// initialize cache.
	if len(c.receivedCustomIDs) == 0 {
		receivedCustomIDs, err := c.getReceivedCustomIDsFromDB()
		if err != nil {
			return err
		}
		c.receivedCustomIDs = receivedCustomIDs
	}
	// add new received custom ID into cache.
	for _, id := range receivedCustomIDs {
		c.receivedCustomIDs[id] = did
	}

	// store received custom ID.
	w := new(bytes.Buffer)
	err := common.WriteUint32(w, uint32(len(c.receivedCustomIDs)+len(receivedCustomIDs)))
	if err != nil {
		return err
	}
	for k, v := range c.receivedCustomIDs {
		if err := common.WriteVarString(w, k); err != nil {
			return err
		}
		if err := v.Serialize(w); err != nil {
			return err
		}
	}
	for _, id := range receivedCustomIDs {
		if err := common.WriteVarString(w, id); err != nil {
			return err
		}
		if err := did.Serialize(w); err != nil {
			return err
		}
	}
	batch.Put(BKTReceivedCustomID, w.Bytes())
	return nil
}

func (c *customID) BatchPutReservedCustomIDs(reservedCustomIDs []string, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPutReservedCustomIDs(reservedCustomIDs, batch)
}

func (c *customID) BatchPutReceivedCustomIDs(receeivedCustomIDs []string,
	did common.Uint168, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPutReceivedCustomIDs(receeivedCustomIDs, did, batch)
}

func (c *customID) GetReservedCustomIDs() (map[string]struct{}, error) {
	c.RLock()
	defer c.RUnlock()
	return c.getReservedCustomIDs()
}

func (c *customID) GetReceivedCustomIDs() (map[string]common.Uint168, error) {
	c.RLock()
	defer c.RUnlock()
	return c.getReceivedCustomIDs()
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

func (c *customID) getReceivedCustomIDsFromDB() (map[string]common.Uint168, error) {
	var val []byte
	val, err := c.db.Get(BKTReceivedCustomID, nil)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(val)
	count, err := common.ReadUint32(r)
	if err != nil {
		return nil, err
	}
	receiedCustomIDs := make(map[string]common.Uint168, 0)
	for i := uint32(0); i < count; i++ {
		id, err := common.ReadVarString(r)
		if err != nil {
			return nil, err
		}
		var did common.Uint168
		if err = did.Deserialize(r); err != nil {
			return nil, err
		}
		receiedCustomIDs[id] = did
	}
	return receiedCustomIDs, nil
}

func (c *customID) getReceivedCustomIDs() (map[string]common.Uint168, error) {
	if len(c.receivedCustomIDs) != 0 {
		return c.receivedCustomIDs, nil
	}

	ids, err := c.getReceivedCustomIDsFromDB()
	if err != nil {
		return nil, err
	}
	// refresh the cache.
	c.receivedCustomIDs = ids
	return ids, nil
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
