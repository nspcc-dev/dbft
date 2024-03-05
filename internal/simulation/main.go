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
	"github.com/nspcc-dev/dbft/internal/block"
	"github.com/nspcc-dev/dbft/internal/crypto"
	"github.com/nspcc-dev/dbft/internal/payload"
	"github.com/twmb/murmur3"
	"go.uber.org/zap"
)

type (
	simNode struct {
		id       int
		d        *dbft.DBFT[crypto.Uint256, crypto.Uint160]
		messages chan dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160]
		key      dbft.PrivateKey
		pub      dbft.PublicKey
		pool     *memPool
		cluster  []*simNode
		log      *zap.Logger

		height     uint32
		lastHash   crypto.Uint256
		validators []dbft.PublicKey
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

func newBlockFromContext(ctx *dbft.Context[crypto.Uint256, crypto.Uint160]) dbft.Block[crypto.Uint256, crypto.Uint160] {
	if ctx.TransactionHashes == nil {
		return nil
	}
	block := block.NewBlock(ctx.Timestamp, ctx.BlockIndex, ctx.NextConsensus, ctx.PrevHash, ctx.Version, ctx.Nonce, ctx.TransactionHashes)
	return block
}

// defaultNewConsensusPayload is default function for creating
// consensus payload of specific type.
func defaultNewConsensusPayload(c *dbft.Context[crypto.Uint256, crypto.Uint160], t dbft.MessageType, msg any) dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160] {
	return payload.NewConsensusPayload(t, c.BlockIndex, uint16(c.MyIndex), c.ViewNumber, msg)
}

func initSimNode(nodes []*simNode, i int, log *zap.Logger) error {
	key, pub := crypto.Generate(rand.Reader)
	nodes[i] = &simNode{
		id:       i,
		messages: make(chan dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160], defaultChanSize),
		key:      key,
		pub:      pub,
		pool:     newMemoryPool(),
		log:      log.With(zap.Int("id", i)),
		cluster:  nodes,
	}

	nodes[i].d = dbft.New[crypto.Uint256, crypto.Uint160](
		dbft.WithLogger[crypto.Uint256, crypto.Uint160](nodes[i].log),
		dbft.WithSecondsPerBlock[crypto.Uint256, crypto.Uint160](time.Second*5),
		dbft.WithKeyPair[crypto.Uint256, crypto.Uint160](key, pub),
		dbft.WithGetTx[crypto.Uint256, crypto.Uint160](nodes[i].pool.Get),
		dbft.WithGetVerified[crypto.Uint256, crypto.Uint160](nodes[i].pool.GetVerified),
		dbft.WithBroadcast[crypto.Uint256, crypto.Uint160](nodes[i].Broadcast),
		dbft.WithProcessBlock[crypto.Uint256, crypto.Uint160](nodes[i].ProcessBlock),
		dbft.WithCurrentHeight[crypto.Uint256, crypto.Uint160](nodes[i].CurrentHeight),
		dbft.WithCurrentBlockHash[crypto.Uint256, crypto.Uint160](nodes[i].CurrentBlockHash),
		dbft.WithGetValidators[crypto.Uint256, crypto.Uint160](nodes[i].GetValidators),
		dbft.WithVerifyPrepareRequest[crypto.Uint256, crypto.Uint160](nodes[i].VerifyPayload),
		dbft.WithVerifyPrepareResponse[crypto.Uint256, crypto.Uint160](nodes[i].VerifyPayload),

		dbft.WithNewBlockFromContext[crypto.Uint256, crypto.Uint160](newBlockFromContext),
		dbft.WithGetConsensusAddress[crypto.Uint256, crypto.Uint160](func(...dbft.PublicKey) crypto.Uint160 { return crypto.Uint160{} }),
		dbft.WithNewConsensusPayload[crypto.Uint256, crypto.Uint160](defaultNewConsensusPayload),
		dbft.WithNewPrepareRequest[crypto.Uint256, crypto.Uint160](payload.NewPrepareRequest),
		dbft.WithNewPrepareResponse[crypto.Uint256, crypto.Uint160](payload.NewPrepareResponse),
		dbft.WithNewChangeView[crypto.Uint256, crypto.Uint160](payload.NewChangeView),
		dbft.WithNewCommit[crypto.Uint256, crypto.Uint160](payload.NewCommit),
		dbft.WithNewRecoveryMessage[crypto.Uint256, crypto.Uint160](func() dbft.RecoveryMessage[crypto.Uint256, crypto.Uint160] {
			return payload.NewRecoveryMessage(nil)
		}),
		dbft.WithNewRecoveryRequest[crypto.Uint256, crypto.Uint160](payload.NewRecoveryRequest),
	)

	if nodes[i].d == nil {
		return errors.New("can't initialize dBFT")
	}

	nodes[i].addTx(*txCount)

	return nil
}

func updatePublicKeys(nodes []*simNode, n int) {
	pubs := make([]dbft.PublicKey, n)
	for i := range pubs {
		pubs[i] = nodes[i].pub
	}

	sortValidators(pubs)

	for i := range nodes {
		nodes[i].validators = pubs
	}
}

func sortValidators(pubs []dbft.PublicKey) {
	sort.Slice(pubs, func(i, j int) bool {
		p1, _ := pubs[i].MarshalBinary()
		p2, _ := pubs[j].MarshalBinary()
		return murmur3.Sum64(p1) < murmur3.Sum64(p2)
	})
}

func (n *simNode) Broadcast(m dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160]) {
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

func (n *simNode) CurrentHeight() uint32            { return n.height }
func (n *simNode) CurrentBlockHash() crypto.Uint256 { return n.lastHash }

// GetValidators always returns the same list of validators.
func (n *simNode) GetValidators(...dbft.Transaction[crypto.Uint256]) []dbft.PublicKey {
	return n.validators
}

func (n *simNode) ProcessBlock(b dbft.Block[crypto.Uint256, crypto.Uint160]) {
	n.d.Logger.Debug("received block", zap.Uint32("height", b.Index()))

	for _, tx := range b.Transactions() {
		n.pool.Delete(tx.Hash())
	}

	n.height = b.Index()
	n.lastHash = b.Hash()
}

// VerifyPayload verifies that payload was received from a good validator.
func (n *simNode) VerifyPayload(p dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160]) error {
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

var _ dbft.Transaction[crypto.Uint256] = (*tx64)(nil)

func (t *tx64) Hash() (h crypto.Uint256) {
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
	store map[crypto.Uint256]dbft.Transaction[crypto.Uint256]
}

func newMemoryPool() *memPool {
	return &memPool{
		mtx:   new(sync.RWMutex),
		store: make(map[crypto.Uint256]dbft.Transaction[crypto.Uint256]),
	}
}

func (p *memPool) Add(tx dbft.Transaction[crypto.Uint256]) {
	p.mtx.Lock()

	h := tx.Hash()
	if _, ok := p.store[h]; !ok {
		p.store[h] = tx
	}

	p.mtx.Unlock()
}

func (p *memPool) Get(h crypto.Uint256) (tx dbft.Transaction[crypto.Uint256]) {
	p.mtx.RLock()
	tx = p.store[h]
	p.mtx.RUnlock()

	return
}

func (p *memPool) Delete(h crypto.Uint256) {
	p.mtx.Lock()
	delete(p.store, h)
	p.mtx.Unlock()
}

func (p *memPool) GetVerified() (txx []dbft.Transaction[crypto.Uint256]) {
	n := *txPerBlock
	if n == 0 {
		return
	}

	txx = make([]dbft.Transaction[crypto.Uint256], 0, n)
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
