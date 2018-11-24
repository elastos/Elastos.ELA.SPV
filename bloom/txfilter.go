package bloom

import (
	"bytes"
	"fmt"

	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/filter"

	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

type txFilter struct {
	filter *Filter
}

func (f *txFilter) Load(filter []byte) error {
	var fl msg.FilterLoad
	err := fl.Deserialize(bytes.NewReader(filter))
	if err != nil {
		return err
	}

	f.filter = LoadFilter(&fl)

	return nil
}

func (f *txFilter) Add(filter []byte) error {
	if f.filter == nil || !f.filter.IsLoaded() {
		return fmt.Errorf("filter not loaded")
	}

	f.filter.Add(filter)

	return nil
}

func (f *txFilter) Match(tx *core.Transaction) bool {
	return f.filter.MatchElaTxAndUpdate(tx)
}

func (f *txFilter) ToMsg() *msg.TxFilterLoad {
	bFilter := f.filter.GetFilterLoadMsg()
	buf := new(bytes.Buffer)
	bFilter.Serialize(buf)
	return &msg.TxFilterLoad{
		Type: filter.FTBloom,
		Data: buf.Bytes(),
	}
}

func NewTxFilter(filter *Filter) filter.TxFilter {
	return &txFilter{filter: filter}
}
