package ledger

import (
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/ledger/pool"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/verifier"
	"sync"
)

type Ledger interface {
	// from other peer
	AddSnapshotBlock(block *common.SnapshotBlock)
	// from self
	MiningSnapshotBlock(address string, timestamp uint64) error
	// from other peer
	AddAccountBlock(account string, block *common.AccountStateBlock)
	// from self
	MiningAccountBlock(address string, block *common.AccountStateBlock) error
	// create account genesis block
	CreateAccount(address string) error
	HeadAccount(address string) (*common.AccountStateBlock, error)
	HeadSnaphost() (*common.SnapshotBlock, error)
}

type ledger struct {
	ac        map[string]*AccountChain
	sc        *Snapshotchain
	pendingSc *pool.SnapshotPool
	pendingAc map[string]*pool.AccountPool
	reqPool   *reqPool

	snapshotVerifier *verifier.SnapshotVerifier
	accountVerifier  *verifier.AccountVerifier
	syncer           syncer.Syncer
	rwMutex          *sync.RWMutex
}

func (self *ledger) HeadAccount(address string) (*common.AccountStateBlock, error) {
	ac := self.selfAc(address)
	if ac == nil {
		return nil, common.StrError{"account not exist."}
	}
	head := ac.Head()
	if head == nil {
		return nil, common.StrError{"head not exist."}
	}
	block := head.(*common.AccountStateBlock)
	return block, nil
}

func (self *ledger) HeadSnaphost() (*common.SnapshotBlock, error) {
	head := self.sc.Head()
	if head == nil {
		return nil, common.StrError{"head not exist."}
	}
	block := head.(*common.SnapshotBlock)
	return block, nil
}

func (self *ledger) CreateAccount(address string) error {
	head := self.sc.Head()
	if self.ac[address] != nil {
		log.Warn("exist account for %s.", address)
		return common.StrError{"exist account " + address}
	}
	accountChain := NewAccountChain(address, self.reqPool, head.Height(), head.Hash())
	accountPool := pool.NewAccountPool("accountChainPool-" + address)
	accountPool.Init(accountChain.insertChain, accountChain.removeChain, self.accountVerifier, self.syncer, accountChain, self.rwMutex.RLocker(), accountChain)
	self.ac[address] = accountChain
	self.pendingAc[address] = accountPool
	return nil
}

func (self *ledger) AddSnapshotBlock(block *common.SnapshotBlock) {
	self.pendingSc.AddBlock(block)
}

func (self *ledger) MiningSnapshotBlock(address string, timestamp uint64) error {
	//self.pendingSc.AddDirectBlock(block)
	return nil
}

func (self *ledger) AddAccountBlock(account string, block *common.AccountStateBlock) {
	self.selfPendingAc(account).AddBlock(block)
}

func (self *ledger) MiningAccountBlock(account string, block *common.AccountStateBlock) error {
	return self.selfPendingAc(account).AddDirectBlock(block)
}

func (self *ledger) selfAc(addr string) *AccountChain {
	return self.ac[addr]
}

func (self *ledger) selfPendingAc(addr string) *pool.AccountPool {
	return self.pendingAc[addr]
}

func (self *ledger) ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error {
	for _, v := range self.pendingAc {
		err := v.RollbackAndForkAccount(nil, forkPoint)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (self *ledger) ForkAccountTo(h *common.AccountHashH) error {
	return self.selfPendingAc(h.Addr).ForkAccount(h)
}

func NewLedger(syncer syncer.Syncer) *ledger {
	ledger := &ledger{}
	ledger.rwMutex = new(sync.RWMutex)

	sc := NewSnapshotChain()
	ledger.snapshotVerifier = verifier.NewSnapshotVerifier(sc, ledger)
	ledger.accountVerifier = verifier.NewAccountVerifier(sc, ledger)
	ledger.syncer = syncer

	snapshotPool := pool.NewSnapshotPool("snapshotPool")
	snapshotPool.Init(sc.insertChain,
		sc.removeChain,
		ledger.snapshotVerifier,
		ledger.syncer,
		sc,
		ledger.rwMutex,
		ledger)
	ledger.reqPool = newReqPool()

	acPools := make(map[string]*pool.AccountPool)
	acs := make(map[string]*AccountChain)
	accounts := Accounts()
	for _, account := range accounts {
		ac := NewAccountChain(account, ledger.reqPool, sc.head.Height(), sc.head.Hash())
		accountPool := pool.NewAccountPool("accountChainPool-" + account)
		accountPool.Init(ac.insertChain, ac.removeChain, ledger.accountVerifier, ledger.syncer, ac, ledger.rwMutex.RLocker(), ac)
		acs[account] = ac
		acPools[account] = accountPool
	}

	ledger.ac = acs
	ledger.sc = sc
	ledger.pendingAc = acPools
	ledger.pendingSc = snapshotPool
	return ledger
}

func (self *ledger) GetFromChain(account string, hash string) *common.AccountStateBlock {
	b := self.selfAc(account).GetBlockByHash(hash)
	if b == nil {
		return nil
	}
	return b.(*common.AccountStateBlock)
}
func (self *ledger) GetByHFromChain(account string, height int) *common.AccountStateBlock {
	b := self.selfAc(account).GetBlock(height)
	if b == nil {
		return nil
	}
	block := b.(*common.AccountStateBlock)
	return block
}
func (self *ledger) GetReferred(account string, sourceHash string) *common.AccountStateBlock {
	self.selfAc(account).GetBySourceBlock(sourceHash)
	return nil
}
func (self *ledger) Start() {
	self.pendingSc.Start()
	for _, pending := range self.pendingAc {
		pending.Start()
	}
}

func Accounts() []string {
	return []string{"viteshan1", "viteshan2", "viteshan3"}
}
