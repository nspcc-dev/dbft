package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/block"
	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/dbft/payload"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/spaolacci/murmur3"
	"go.uber.org/zap"
)

type (
	simNode struct {
		id       int
		d        *dbft.DBFT[util.Uint256, util.Uint160]
		messages chan payload.ConsensusPayload[util.Uint256, util.Uint160]
		key      crypto.PrivateKey
		pub      crypto.PublicKey
		pool     *memPool
		cluster  []*simNode
		log      *zap.Logger

		height     uint32
		lastHash   util.Uint256
		validators []crypto.PublicKey
	}
)

const (
	defaultChanSize = 100
)

var (
	nodebug    = flag.Bool("nodebug", false, "disable debug logging")
	count      = flag.Int("count", 7, "node count")
	watchers   = flag.Int("watchers", 7, "watch-only node count")
	blocked    = flag.Int("blocked", -1, "blocked validator (payloads from him/her are dropped)")
	txPerBlock = flag.Int("txblock", 1, "transactions per block")
	txCount    = flag.Int("txcount", 100000, "transactions on every node")
	duration   = flag.Duration("duration", time.Second*20, "duration of simulation (infinite by default)")
)

func main() {
	flag.Parse()

	initDebugger()

	logger := initLogger()
	clusterSize := *count
	watchOnly := *watchers
	nodes := make([]*simNode, clusterSize+watchOnly)

	initNodes(nodes, logger)
	updatePublicKeys(nodes, clusterSize)

	ctx, cancel := initContext(*duration)
	defer cancel()

	wg := new(sync.WaitGroup)
	wg.Add(len(nodes))

	for i := range nodes {
		go func(i int) {
			defer wg.Done()

			nodes[i].Run(ctx)
		}(i)
	}

	wg.Wait()
}

// Run implements simple event loop.
func (n *simNode) Run(ctx context.Context) {
	n.d.Start(0)

	for {
		select {
		case <-ctx.Done():
			n.log.Info("context cancelled")
			return
		case <-n.d.Timer.C():
			n.d.OnTimeout(n.d.Timer.HV())
		case msg := <-n.messages:
			n.d.OnReceive(msg)
		}
	}
}

func initNodes(nodes []*simNode, log *zap.Logger) {
	for i := range nodes {
		if err := initSimNode(nodes, i, log); err != nil {
			panic(err)
		}
	}
}

// defaultNewConsensusPayload is default function for creating
// consensus payload of specific type.
func defaultNewConsensusPayload(c *dbft.Context[util.Uint256, util.Uint160], t payload.MessageType, msg any) payload.ConsensusPayload[util.Uint256, util.Uint160] {
	cp := payload.NewConsensusPayload()
	cp.SetHeight(c.BlockIndex)
	cp.SetValidatorIndex(uint16(c.MyIndex))
	cp.SetViewNumber(c.ViewNumber)
	cp.SetType(t)
	cp.SetPayload(msg)

	return cp
}

func initSimNode(nodes []*simNode, i int, log *zap.Logger) error {
	key, pub := crypto.Generate(rand.Reader)
	nodes[i] = &simNode{
		id:       i,
		messages: make(chan payload.ConsensusPayload[util.Uint256, util.Uint160], defaultChanSize),
		key:      key,
		pub:      pub,
		pool:     newMemoryPool(),
		log:      log.With(zap.Int("id", i)),
		cluster:  nodes,
	}

	nodes[i].d = dbft.New[util.Uint256, util.Uint160](
		dbft.WithLogger[util.Uint256, util.Uint160](nodes[i].log),
		dbft.WithSecondsPerBlock[util.Uint256, util.Uint160](time.Second*5),
		dbft.WithKeyPair[util.Uint256, util.Uint160](key, pub),
		dbft.WithGetTx[util.Uint256, util.Uint160](nodes[i].pool.Get),
		dbft.WithGetVerified[util.Uint256, util.Uint160](nodes[i].pool.GetVerified),
		dbft.WithBroadcast[util.Uint256, util.Uint160](nodes[i].Broadcast),
		dbft.WithProcessBlock[util.Uint256, util.Uint160](nodes[i].ProcessBlock),
		dbft.WithCurrentHeight[util.Uint256, util.Uint160](nodes[i].CurrentHeight),
		dbft.WithCurrentBlockHash[util.Uint256, util.Uint160](nodes[i].CurrentBlockHash),
		dbft.WithGetValidators[util.Uint256, util.Uint160](nodes[i].GetValidators),
		dbft.WithVerifyPrepareRequest[util.Uint256, util.Uint160](nodes[i].VerifyPayload),
		dbft.WithVerifyPrepareResponse[util.Uint256, util.Uint160](nodes[i].VerifyPayload),

		dbft.WithNewBlockFromContext[util.Uint256, util.Uint160](dbft.NewBlockFromContext),
		dbft.WithGetConsensusAddress[util.Uint256, util.Uint160](func(...crypto.PublicKey) util.Uint160 { return util.Uint160{} }),
		dbft.WithNewConsensusPayload[util.Uint256, util.Uint160](defaultNewConsensusPayload),
		dbft.WithNewPrepareRequest[util.Uint256, util.Uint160](payload.NewPrepareRequest),
		dbft.WithNewPrepareResponse[util.Uint256, util.Uint160](payload.NewPrepareResponse),
		dbft.WithNewChangeView[util.Uint256, util.Uint160](payload.NewChangeView),
		dbft.WithNewCommit[util.Uint256, util.Uint160](payload.NewCommit),
		dbft.WithNewRecoveryMessage[util.Uint256, util.Uint160](payload.NewRecoveryMessage),
		dbft.WithNewRecoveryRequest[util.Uint256, util.Uint160](payload.NewRecoveryRequest),
	)

	if nodes[i].d == nil {
		return errors.New("can't initialize dBFT")
	}

	nodes[i].addTx(*txCount)

	return nil
}

func updatePublicKeys(nodes []*simNode, n int) {
	pubs := make([]crypto.PublicKey, n)
	for i := range pubs {
		pubs[i] = nodes[i].pub
	}

	sortValidators(pubs)

	for i := range nodes {
		nodes[i].validators = pubs
	}
}

func sortValidators(pubs []crypto.PublicKey) {
	sort.Slice(pubs, func(i, j int) bool {
		p1, _ := pubs[i].MarshalBinary()
		p2, _ := pubs[j].MarshalBinary()
		return murmur3.Sum64(p1) < murmur3.Sum64(p2)
	})
}

func (n *simNode) Broadcast(m payload.ConsensusPayload[util.Uint256, util.Uint160]) {
	for i, node := range n.cluster {
		if i != n.id {
			select {
			case node.messages <- m:
			default:
				n.log.Warn("can't broadcast message: channel is full")
			}
		}
	}
}

func (n *simNode) CurrentHeight() uint32          { return n.height }
func (n *simNode) CurrentBlockHash() util.Uint256 { return n.lastHash }

// GetValidators always returns the same list of validators.
func (n *simNode) GetValidators(...block.Transaction[util.Uint256]) []crypto.PublicKey {
	return n.validators
}

func (n *simNode) ProcessBlock(b block.Block[util.Uint256, util.Uint160]) {
	n.d.Logger.Debug("received block", zap.Uint32("height", b.Index()))

	for _, tx := range b.Transactions() {
		n.pool.Delete(tx.Hash())
	}

	n.height = b.Index()
	n.lastHash = b.Hash()
}

// VerifyPrepareRequest verifies that payload was received from a good validator.
func (n *simNode) VerifyPayload(p payload.ConsensusPayload[util.Uint256, util.Uint160]) error {
	if *blocked != -1 && p.ValidatorIndex() == uint16(*blocked) {
		return fmt.Errorf("message from blocked validator: %d", *blocked)
	}
	return nil
}

func (n *simNode) addTx(count int) {
	for i := 0; i < count; i++ {
		tx := tx64(uint64(i))
		n.pool.Add(&tx)
	}
}

// =============================
// Small transaction.
// =============================

type tx64 uint64

var _ block.Transaction[util.Uint256] = (*tx64)(nil)

func (t *tx64) Hash() (h util.Uint256) {
	binary.LittleEndian.PutUint64(h[:], uint64(*t))
	return
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (t *tx64) MarshalBinary() ([]byte, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(*t))

	return b, nil
}

// UnmarshalBinary implements encoding.BinaryUnarshaler interface.
func (t *tx64) UnmarshalBinary(data []byte) error {
	if len(data) != 8 {
		return errors.New("length must equal 8 bytes")
	}

	*t = tx64(binary.LittleEndian.Uint64(data))

	return nil
}

// =============================
// Memory pool for transactions.
// =============================

type memPool struct {
	mtx   *sync.RWMutex
	store map[util.Uint256]block.Transaction[util.Uint256]
}

func newMemoryPool() *memPool {
	return &memPool{
		mtx:   new(sync.RWMutex),
		store: make(map[util.Uint256]block.Transaction[util.Uint256]),
	}
}

func (p *memPool) Add(tx block.Transaction[util.Uint256]) {
	p.mtx.Lock()

	h := tx.Hash()
	if _, ok := p.store[h]; !ok {
		p.store[h] = tx
	}

	p.mtx.Unlock()
}

func (p *memPool) Get(h util.Uint256) (tx block.Transaction[util.Uint256]) {
	p.mtx.RLock()
	tx = p.store[h]
	p.mtx.RUnlock()

	return
}

func (p *memPool) Delete(h util.Uint256) {
	p.mtx.Lock()
	delete(p.store, h)
	p.mtx.Unlock()
}

func (p *memPool) GetVerified() (txx []block.Transaction[util.Uint256]) {
	n := *txPerBlock
	if n == 0 {
		return
	}

	txx = make([]block.Transaction[util.Uint256], 0, n)
	for _, tx := range p.store {
		txx = append(txx, tx)

		if n--; n == 0 {
			return
		}
	}

	return
}

// initDebugger initializes pprof debug facilities.
func initDebugger() {
	r := http.NewServeMux()
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go func() {
		err := http.ListenAndServe("localhost:6060", r)
		if err != nil {
			panic(err)
		}
	}()
}

// initLogger initializes new logger.
func initLogger() *zap.Logger {
	if *nodebug {
		return zap.L()
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("can't init logger")
	}

	return logger
}

// initContext creates new context which will be cancelled by Ctrl+C.
func initContext(d time.Duration) (ctx context.Context, cancel func()) {
	// exit by Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-c
		cancel()
	}()

	if d != 0 {
		return context.WithTimeout(context.Background(), *duration)
	}

	return context.WithCancel(context.Background())
}
