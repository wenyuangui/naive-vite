package chain

import (
	"sync"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/store"
)

type BlockChain interface {
	face.SnapshotReader
	face.SnapshotWriter
	face.AccountReader
	face.AccountWriter
	SetChainListener(listener face.ChainListener)
}

type blockchain struct {
	ac       map[string]*accountChain
	sc       *snapshotChain
	store    store.BlockStore
	listener face.ChainListener

	mu sync.Mutex // account chain init
}

func NewChain() BlockChain {
	self := &blockchain{}
	self.store = store.NewMemoryStore(GetGenesisSnapshot())
	self.sc = newSnapshotChain(self.store)
	self.listener = &defaultChainListener{}
	self.ac = make(map[string]*accountChain)
	return self
}
func (self *blockchain) selfAc(addr string) *accountChain {
	chain, ok := self.ac[addr]
	if !ok {
		c := newAccountChain(addr, self.listener, self.store)
		self.mu.Lock()
		defer self.mu.Unlock()
		self.ac[addr] = c
		return self.ac[addr]
	}
	return chain
}

// query received block by send block
func (self *blockchain) GetAccountBySourceHash(address string, source string) *common.AccountStateBlock {
	return self.selfAc(address).getBySourceBlock(source)
}

func (self *blockchain) NextAccountSnapshot() (common.HashHeight, []*common.AccountHashH, error) {
	head := self.sc.head
	//common.SnapshotBlock{}
	var accounts []*common.AccountHashH
	for k, v := range self.ac {
		i, s := v.NextSnapshotPoint()
		if i < 0 {
			continue
		}
		accounts = append(accounts, common.NewAccountHashH(k, s, i))
	}
	if len(accounts) == 0 {
		accounts = nil
	}

	return common.HashHeight{Hash: head.Hash(), Height: head.Height()}, accounts, nil
}

func (self *blockchain) FindAccountAboveSnapshotHeight(address string, snapshotHeight int) *common.AccountStateBlock {
	return self.selfAc(address).findAccountAboveSnapshotHeight(snapshotHeight)
}

func (self *blockchain) SetChainListener(listener face.ChainListener) {
	if listener == nil {
		return
	}
	self.listener = listener
}

func (self *blockchain) GenesisSnapshot() (*common.SnapshotBlock, error) {
	return GetGenesisSnapshot(), nil
}

func (self *blockchain) HeadSnapshot() (*common.SnapshotBlock, error) {
	return self.sc.Head(), nil
}

func (self *blockchain) GetSnapshotByHashH(hashH common.HashHeight) *common.SnapshotBlock {
	return self.sc.GetBlockByHashH(hashH)
}

func (self *blockchain) GetSnapshotByHash(hash string) *common.SnapshotBlock {
	return self.sc.getBlockByHash(hash)
}

func (self *blockchain) GetSnapshotByHeight(height int) *common.SnapshotBlock {
	return self.sc.GetBlockHeight(height)
}

func (self *blockchain) InsertSnapshotBlock(block *common.SnapshotBlock) error {
	err := self.sc.insertChain(block)
	if err == nil {
		// update next snapshot index
		for _, account := range block.Accounts {
			err := self.selfAc(account.Addr).SnapshotPoint(block.Height(), block.Hash(), account)
			if err != nil {
				log.Error("update snapshot point fail.")
				return err
			}
		}
	}
	return err
}

func (self *blockchain) RemoveSnapshotHead(block *common.SnapshotBlock) error {
	return self.sc.removeChain(block)
}

func (self *blockchain) HeadAccount(address string) (*common.AccountStateBlock, error) {
	return self.selfAc(address).Head(), nil
}

func (self *blockchain) GetAccountByHashH(address string, hashH common.HashHeight) *common.AccountStateBlock {
	return self.selfAc(address).GetBlockByHashH(hashH)
}

func (self *blockchain) GetAccountByHash(address string, hash string) *common.AccountStateBlock {
	return self.selfAc(address).GetBlockByHash(hash)
}

func (self *blockchain) GetAccountByHeight(address string, height int) *common.AccountStateBlock {
	return self.selfAc(address).GetBlockByHeight(height)
}

func (self *blockchain) InsertAccountBlock(address string, block *common.AccountStateBlock) error {
	return self.selfAc(address).insertChain(block)
}

func (self *blockchain) RemoveAccountHead(address string, block *common.AccountStateBlock) error {
	return self.selfAc(address).removeChain(block)
}
func (self *blockchain) RollbackSnapshotPoint(address string, start *common.SnapshotPoint, end *common.SnapshotPoint) error {
	return self.selfAc(address).RollbackSnapshotPoint(start, end)
}

type defaultChainListener struct {
}

func (*defaultChainListener) SnapshotInsertCallback(block *common.SnapshotBlock) {

}

func (*defaultChainListener) SnapshotRemoveCallback(block *common.SnapshotBlock) {

}

func (*defaultChainListener) AccountInsertCallback(address string, block *common.AccountStateBlock) {

}

func (*defaultChainListener) AccountRemoveCallback(address string, block *common.AccountStateBlock) {
}
