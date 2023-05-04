package sdk

import (
	"sync"
)

/*
This is a helper class to filter interested tx types when synchronize transactions
or get cached tx types list to build a bloom filter instead of load tx types from
database every time.
*/
type TxTypesFilter struct {
	sync.Mutex
	txTypes map[uint8]struct{}
}

// Create a TxTypesFilter instance, you can pass all the tx types through this method
// or pass nil and use AddTxType() method to add interested tx types later.
func NewTxTypesFilter(txTypes []uint8) *TxTypesFilter {
	filter := new(TxTypesFilter)
	filter.LoadTxTypes(txTypes)
	return filter
}

// Load or reload all the interested transaction types into the TxTypesFilter
func (filter *TxTypesFilter) LoadTxTypes(txTypes []uint8) {
	filter.Lock()
	defer filter.Unlock()

	filter.clear()
	for _, txType := range txTypes {
		filter.txTypes[txType] = struct{}{}
	}
}

// Check if transaction types are loaded into this Filter
func (filter *TxTypesFilter) IsLoaded() bool {
	filter.Lock()
	defer filter.Unlock()

	return len(filter.txTypes) > 0
}

// Add a interested transaction type into this Filter
func (filter *TxTypesFilter) AddTxType(txType uint8) {
	filter.Lock()
	defer filter.Unlock()

	filter.txTypes[txType] = struct{}{}
}

// Remove a transaction type from this Filter
func (filter *TxTypesFilter) DeleteTxType(txType uint8) {
	filter.Lock()
	defer filter.Unlock()

	delete(filter.txTypes, txType)
}

// Get transaction types that were added into this Filter
func (filter *TxTypesFilter) GetTxTypes() []uint8 {
	filter.Lock()
	defer filter.Unlock()

	var txTypes = make([]uint8, 0, len(filter.txTypes))
	for t, _ := range filter.txTypes {
		txTypes = append(txTypes, t)
	}

	return txTypes
}

// Check if a transaction type was added into this filter as a interested tx type
func (filter *TxTypesFilter) ContainTxType(txType uint8) bool {
	filter.Lock()
	defer filter.Unlock()

	_, ok := filter.txTypes[txType]
	return ok
}

func (filter *TxTypesFilter) Clear() {
	filter.Lock()
	defer filter.Unlock()

	filter.clear()
}

func (filter *TxTypesFilter) clear() {
	filter.txTypes = make(map[uint8]struct{})
}
