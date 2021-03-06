package store

import (
	"sync"

	"strconv"

	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/tools"
)

type BlockStore interface {
	PutSnapshot(block *common.SnapshotBlock)
	PutAccount(address string, block *common.AccountStateBlock)
	DeleteSnapshot(hashH common.HashHeight)
	DeleteAccount(address string, hashH common.HashHeight)

	SetSnapshotHead(hashH *common.HashHeight)
	SetAccountHead(address string, hashH *common.HashHeight)

	GetSnapshotHead() *common.HashHeight
	GetAccountHead(address string) *common.HashHeight

	GetSnapshotByHash(hash string) *common.SnapshotBlock
	GetSnapshotByHeight(height int) *common.SnapshotBlock

	GetAccountByHash(hash string) *common.AccountStateBlock
	GetAccountByHeight(address string, height int) *common.AccountStateBlock

	GetAccountBySourceHash(hash string) *common.AccountStateBlock
	PutSourceHash(hash string, block *common.AccountStateBlock)
	DeleteSourceHash(hash string)
}

func NewMemoryStore(snapshotGenesis *common.SnapshotBlock) BlockStore {
	self := &blockMemoryStore{}
	for _, a := range genesisAccounts {
		genesis := common.NewAccountBlock(0, "", "", a, time.Unix(0, 0),
			200, 0, 0, "", common.GENESIS, a, a, "", -1)
		genesis.SetHash(tools.CalculateAccountHash(genesis))
		self.PutAccount(a, genesis)
		self.SetAccountHead(a, &common.HashHeight{Hash: genesis.Hash(), Height: genesis.Height()})
	}

	return self
}

var genesisAccounts = []string{"viteshan", "jie"}

// thread safe block memory store
type blockMemoryStore struct {
	snapshotHeight sync.Map
	snapshotHash   sync.Map
	accountHeight  sync.Map
	accountHash    sync.Map
	// key: source hash val: received
	sourceHash sync.Map

	head sync.Map

	sMu sync.Mutex
	aMu sync.Mutex
}

var snapshotHeadKey = "s_head_key"

func (self *blockMemoryStore) GetSnapshotHead() *common.HashHeight {
	value, ok := self.head.Load(snapshotHeadKey)
	if !ok {
		return nil
	}
	return value.(*common.HashHeight)
}

func (self *blockMemoryStore) GetAccountHead(address string) *common.HashHeight {
	value, ok := self.head.Load(address)
	if !ok {
		return nil
	}
	return value.(*common.HashHeight)
}

func (self *blockMemoryStore) SetSnapshotHead(hashH *common.HashHeight) {
	if hashH == nil {
		self.head.Delete(snapshotHeadKey)
	} else {
		self.head.Store(snapshotHeadKey, hashH)
	}
}

func (self *blockMemoryStore) SetAccountHead(address string, hashH *common.HashHeight) {
	if hashH == nil {
		self.head.Delete(address)
	} else {
		self.head.Store(address, hashH)
	}

}

func (self *blockMemoryStore) DeleteSnapshot(hashH common.HashHeight) {
	self.snapshotHeight.Delete(hashH.Height)
	self.snapshotHash.Delete(hashH.Hash)
}

func (self *blockMemoryStore) DeleteAccount(address string, hashH common.HashHeight) {
	self.accountHash.Delete(hashH.Hash)
	self.accountHeight.Delete(self.genKey(address, hashH.Height))
}

func (self *blockMemoryStore) PutSnapshot(block *common.SnapshotBlock) {
	self.snapshotHash.Store(block.Hash(), block)
	self.snapshotHeight.Store(block.Height(), block)
}

func (self *blockMemoryStore) PutAccount(address string, block *common.AccountStateBlock) {
	self.accountHash.Store(block.Hash(), block)
	self.accountHeight.Store(self.genKey(address, block.Height()), block)
}

func (self *blockMemoryStore) GetSnapshotByHash(hash string) *common.SnapshotBlock {
	value, ok := self.snapshotHash.Load(hash)
	if !ok {
		return nil
	}
	return value.(*common.SnapshotBlock)
}

func (self *blockMemoryStore) GetSnapshotByHeight(height int) *common.SnapshotBlock {
	value, ok := self.snapshotHeight.Load(height)
	if !ok {
		return nil
	}
	return value.(*common.SnapshotBlock)
}

func (self *blockMemoryStore) GetAccountByHash(hash string) *common.AccountStateBlock {
	value, ok := self.accountHash.Load(hash)
	if !ok {
		return nil
	}
	return value.(*common.AccountStateBlock)
}

func (self *blockMemoryStore) GetAccountBySourceHash(hash string) *common.AccountStateBlock {
	value, ok := self.sourceHash.Load(hash)
	if !ok {
		return nil
	}
	return value.(*common.AccountStateBlock)
}

func (self *blockMemoryStore) PutSourceHash(hash string, block *common.AccountStateBlock) {
	self.sourceHash.Store(hash, block)
}
func (self *blockMemoryStore) DeleteSourceHash(hash string) {
	self.sourceHash.Delete(hash)
}

func (self *blockMemoryStore) GetAccountByHeight(address string, height int) *common.AccountStateBlock {
	value, ok := self.accountHeight.Load(self.genKey(address, height))
	if !ok {
		return nil
	}
	return value.(*common.AccountStateBlock)

}

func (self *blockMemoryStore) genKey(address string, height int) string {
	return address + "_" + strconv.Itoa(height)
}
