package examples

import (
	"bufio"
	"github.com/asaskevich/EventBus"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/consensus"
	"github.com/viteshan/naive-vite/ledger"
	"github.com/viteshan/naive-vite/miner"
	"github.com/viteshan/naive-vite/syncer"
	"github.com/viteshan/naive-vite/tools"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type V0_12 struct {
	nodes      map[string]string
	ledger     ledger.Ledger
	chainmutex *sync.Mutex
}

type TestSyncer struct {
}

func (self *TestSyncer) Fetch(hash syncer.BlockHash, prevCnt int) {

}
func newV0_12() *V0_12 {
	self := &V0_12{}
	self.nodes = make(map[string]string)
	self.chainmutex = &sync.Mutex{}
	self.ledger = ledger.NewLedger(&TestSyncer{})

	self.ledger.Start()
	genesisTime := ledger.GetGenesisSnapshot().Timestamp()
	committee := consensus.NewCommittee(genesisTime, 1, int32(len(consensus.DefaultMembers)))

	bus := EventBus.New()
	coinbase := common.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	miner := miner.NewMiner(self.ledger, bus, coinbase, committee)

	committee.Init()
	miner.Init()
	committee.Start()
	miner.Start()
	select {
	case <-time.After(1 * time.Second):
		println("downloader finish.")
		//miner.downloaderRegisterCh <- 0
		bus.Publish(common.DwlDone)
	}
	return self
}

func Run_0_12() {
	self := newV0_12()
	httpPort := strconv.Itoa(9000)

	// start TCP and serve TCP server
	server, err := net.Listen("tcp", ":"+httpPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Tcp Server Listening on port :", httpPort)
	defer server.Close()

	//go func() {
	//	for {
	//		time.Sleep(10 * time.Second)
	//		output := printAccountBlockChain(accountStateBlockChain)
	//		log.Printf("%v", output)
	//		snapshot := printSnapshotBlockChain(snapshotBlockChain)
	//		log.Printf("%v", snapshot)
	//	}
	//}()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go self.handleConn(conn)
	}
}
func (self *V0_12) handleConn(conn net.Conn) {
	defer conn.Close()

	var closedChan = make(chan bool)

	// validator address
	var address string

	// allow user to allocate number of tokens to stake
	// the greater the number of tokens, the greater chance to forging a new block
	io.WriteString(conn, "Enter node address:")

	scanAddress := bufio.NewScanner(conn)
	if scanAddress.Scan() {
		address = scanAddress.Text()
	}

	self.ledger.CreateAccount(address)

	node := self.initNode(address)
	defer self.destoryNode(node)

	for {
		io.WriteString(conn, "+++++++++++++++++++++++++++++++++++++++++++++\n"+address+
			", Enter role:\n 1:get balance\n 2:send tx\n 3:snapshot height\n 4:list request\n 5:response for request[from,hash]\n\n> ")

		scanRole := bufio.NewScanner(conn)
		if scanRole.Scan() {
			role, err := strconv.Atoi(scanRole.Text())
			if err != nil {
				log.Printf("%v not a number: %v", scanRole.Text(), err)
				return
			}

			if role == 1 {
				self.printBalance(conn, address)
			} else if role == 2 {
				self.sendTx(conn, address)
			} else if role == 3 {
				self.printSnapshot(conn)
			} else if role == 4 {
				self.printReceivedTxs(conn, address)
			} else if role == 5 {
				self.receiveTx(conn, address)
			}
		}
	}

	closedChan <- true

}

func (self *V0_12) receiveTx(conn net.Conn, address string) {
	io.WriteString(conn, "Enter Req Hash:")
	scanTx := bufio.NewScanner(conn)
	if scanTx.Scan() {
		input := scanTx.Text()
		s := strings.Split(input, ",")

		err := self.ledger.ResponseAccountBlock(s[0], address, s[1])
		if err != nil {
			io.WriteString(conn, "request error for:"+input+"\n")
		} else {
			io.WriteString(conn, "request done for:"+input+"\n")
		}
	}
}

func (self *V0_12) destoryNode(node string) {
	self.chainmutex.Lock()
	defer self.chainmutex.Unlock()

	delete(self.nodes, node)
}
func (self *V0_12) initNode(node string) string {
	self.chainmutex.Lock()
	defer self.chainmutex.Unlock()

	self.nodes[node] = node
	return node
}

func (self *V0_12) printBalance(conn net.Conn, address string) {
	currentBalance := self.ledger.GetAccountBalance(address)
	io.WriteString(conn, "current balance is :"+strconv.Itoa(currentBalance)+"\n")
}

func (self *V0_12) sendTx(conn net.Conn, address string) {
	io.WriteString(conn, "Enter to address:")
	scanTx := bufio.NewScanner(conn)
	var toAddress string
	if scanTx.Scan() {
		toAddress = scanTx.Text()
		exists := self.ledger.ExistAccount(toAddress)
		if !exists {
			io.WriteString(conn, "address:"+toAddress+" not exists\n")
			return
		}
	}
	io.WriteString(conn, address+", Enter to amount:")

	if scanTx.Scan() {
		toAmount, err := strconv.Atoi(scanTx.Text())
		if err != nil {
			log.Printf("%v not a number: %v", scanTx.Text(), err)
			return
		}
		self.submitTx(address, toAddress, -toAmount)
	}

}

func (self *V0_12) submitTx(from string, to string, amount int) {
	headAccount, _ := self.ledger.HeadAccount(from)
	headSnaphost, _ := self.ledger.HeadSnapshost()
	newBlock := common.NewAccountBlock(headAccount.Height()+1, "", headAccount.Hash(), from, time.Now(), headAccount.Amount+amount, amount, headSnaphost.Height(), headSnaphost.Hash(),
		common.SEND, from, to, "")
	newBlock.SetHash(tools.CalculateAccountHash(newBlock))
	err := self.ledger.RequestAccountBlock(from, newBlock)

	if err == nil {
		log.Printf("submit send Tx success[" + from + "].\n")
	} else {
		log.Printf("submit send Tx failed["+from+"].\n", err)
	}
}
func (self *V0_12) printSnapshot(conn net.Conn) {
	headSnaphost, _ := self.ledger.HeadSnapshost()
	io.WriteString(conn, "current snapshot height is:"+strconv.Itoa(headSnaphost.Height())+"\n")
}
func (self *V0_12) printReceivedTxs(conn net.Conn, address string) {
	reqs := self.ledger.ListRequest(address)

	var lines string
	for _, v := range reqs {
		req := "From:\t" + v.From + "\tAmount:\t" + strconv.Itoa(v.Amount) + "\tReqHash:" + v.ReqHash + "\n"
		lines = lines + req
	}
	io.WriteString(conn, "current request list is:\n"+lines)
}