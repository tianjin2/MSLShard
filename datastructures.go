package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/kimborgen/go-merkletree"
)

// only data structures that are common in multiple, disjoint files, should belong here

type Node_InitialMessageToCoordinator struct {
	Pub  *PubKey
	Port int
}

// Trust management

// rate rank
type indicators struct {
	BlockDecision string
	Latency   float64
	CpuMachine   float64
}

type DecisionVector struct {
	Decision   map[[32]byte]indicators   //TxHash -> Y,N,U
	//	Af 		   float64
	Round      uint
	NodePub    [32]byte    //The public key of the node making the decision

}
// initialization
func (dv *DecisionVector) init()  {
	dv.Decision = make(map[[32]byte]indicators )
}

//add decision
func (dv *DecisionVector) add(txHash [32]byte,decision indicators) {
	dv.Decision[txHash] = decision
}


func (dv *DecisionVector) delete () {
	dv.Decision = make(map[[32]byte]indicators )
}

// Output
func (dv *DecisionVector)Tostring() string {
	var s string
	for tx,in := range dv.Decision {
		s += bytesToString(tx[:])+": "+in.BlockDecision+","+strconv.FormatFloat(in.Latency,'f',4,64)+","+strconv.FormatFloat(in.CpuMachine,'f',4,64)+"\n"
	}
	return s
}

type sendCoordToNodeLocalTrust struct {
	NodePub   [32]byte
	Af		  []float64
	ETrust    float64
	Trust  map[[32]byte]sendToNodeTrust
}
// Trust value
type TrustVector struct{
	Round        uint
	NodePub    	 [32]byte
	CommitteeID  [32]byte
	Af           []float64
	Ptrust    	 map[[32]byte]float64  //nodePub -> TxHash -> Y,N,U
	Jsang 	     map[[32]byte]*Jsang
	Signature    *Sig

}

// trust vector set
func (tv *TrustVector) init() {
	tv.Af = make([]float64,3)
	tv.Ptrust = make(map[[32]byte]float64)
	tv.Jsang = make(map[[32]byte]*Jsang)
	//tv.Strust = make(map[[32]byte]float64)
}
func (tv *TrustVector) add(nodePub [32]byte, trustValue float64,strustValue float64) {

	tv.Ptrust[nodePub]= trustValue
	//	tv.Strust[nodePub]= strustValue
}

func (tv *TrustVector) delete() {

	tv.Ptrust =  make(map[[32]byte]float64)
	tv.Jsang = make(map[[32]byte]*Jsang)
	//	tv.Strust =  make(map[[32]byte]float64)
}
type sendToNodeTrust struct {
	trust   float64
	A_R     float64
}
type selfTrust struct {
	Trust    float64  //可信度
	ETrust    float64
	Af        []float64
	nodeTrust map[[32]byte]sendToNodeTrust
	nodeEL    map[uint]map[[32]byte]float64
}
func (selfT *selfTrust) init(nodeCtx *NodeCtx,isInait bool){

	/*for i:=0 ;i< 3 ;i++ {
		selfT.Af[i] = 0.5
	}*/
	if isInait{
		selfT.Af = make([]float64,3)
		selfT.Af[0] = 0.5
		selfT.Af[1] = 0.25
		selfT.Af[2] = 0.25
		selfT.ETrust = 0
	}
	selfT.nodeEL = make(map[uint]map[[32]byte]float64)
	selfT.nodeTrust = make(map[[32]byte]sendToNodeTrust)
	selfT.nodeEL[0] = make(map[[32]byte]float64)
	for node :=  range nodeCtx.committee.Members {
		selfT.nodeEL[0][node] = 0.5
	}

}
func (selfT *selfTrust) add(round uint,nodePub [32]byte , trust float64){
	selfT.nodeEL[round][nodePub] = trust

}


func (selfT *selfTrust) delect(round uint){
	delete(selfT.nodeEL,round)
}

type nodeJsang struct {
	Jsang     map[uint]map[[32]byte]*Jsang //Nodal Subjective Trust Evaluation: round -> nodePub -> Jsang
	mux       sync.Mutex
}
type Jsang struct {
	Time  time.Time
	R_b   []float64
	R_u   float64
}
type Qos struct {
	latency  float64
	TransmitEnergy float64
	PacketLossRate float64
	Reliability    float64
}
type QoSec struct {
	Confidentiality float64
	Integrity float64
	Usability float64
	AccessControlLevel int64
}

type InitValue struct {
	qos   Qos
	qosSec QoSec
	C      float64 //Recommendation of trust
	cpu    float64
}
type Coordinate struct {
	x string
	y string
}
type nodeTs struct {
	node *PubKey
	trust float64
}
func completeDistance(a,b Coordinate) float64 {
	x1, _ := strconv.ParseFloat(a.x, 64)
	x2, _ := strconv.ParseFloat(b.x, 64)
	y1, _ := strconv.ParseFloat(a.y, 64)
	y2, _ := strconv.ParseFloat(b.y, 64)
	return math.Sqrt(math.Pow(x1-x2,2)+math.Pow(y1-y2,2))
}


func (InitV *InitValue) string() string{
	var s string
	s += "latency:"+strconv.FormatFloat(InitV.qos.latency,'f',4,64)+"\n"
	s += "ransmitEnergy:"+strconv.FormatFloat(InitV.qos.TransmitEnergy,'f',4,64)+"\n"
	s += "PacketLossRate:"+strconv.FormatFloat(InitV.qos.PacketLossRate,'f',4,64)+"\n"
	s += "Reliability:"+strconv.FormatFloat(InitV.qos.Reliability,'f',4,64)+"\n"
	s += "Integrity:"+strconv.FormatFloat(InitV.qosSec.Integrity,'f',4,64)+"\n"
	s += "Usability:"+strconv.FormatFloat(InitV.qosSec.Usability,'f',4,64)+"\n"
	s += "Confidentiality,:"+strconv.FormatFloat(InitV.qosSec.Confidentiality,'f',4,64)+"\n"
	s += "AccessControlLevel:"+strconv.FormatFloat(float64(InitV.qosSec.AccessControlLevel)/5,'f',4,64)+"\n"
	s += "C:"+strconv.FormatFloat(InitV.C,'f',4,64)+"\n"
	return s
}
func (InitV *InitValue) init(nodeCtx *NodeCtx, coord map[[32]byte]Coordinate){
	// r.Seed(100)
	//rnd := r.Uint32()
	rnd := RandInt64(1,100)
	time.Sleep(10*time.Millisecond)
	sum := 0.0

	for node := range nodeCtx.committee.Members {
		sum += completeDistance(nodeCtx.self.coordinate,coord[node])
	}
	latency := sum/float64(len(nodeCtx.committee.Members)+1)/100
	if rnd >=90 {

		transmitE := RandFloat64(0.7,0.9)
		PacketLossrate := RandFloat64(0.6,0.8)
		reliability := RandFloat64(0.2,0.5)
		qos := Qos{latency,transmitE,PacketLossrate,reliability}
		confidentiality := RandFloat64(0.2,0.5)
		integrity := RandFloat64(0.3,0.5)
		usability := RandFloat64(0.2,0.55)
		accessControlLevel := RandInt64(0,2)
		qosSec := QoSec{confidentiality,integrity,usability,accessControlLevel}
		C := RandFloat64(0.2,0.5)
		InitV.qos = qos
		InitV.qosSec = qosSec
		InitV.C = C
		InitV.cpu = RandFloat64(0.4,0.6)
	}else if rnd >=0 && rnd < 40 {


		transmitE := RandFloat64(0.2,0.4)
		PacketLossrate := RandFloat64(0,0.3)
		reliability := RandFloat64(0.7,0.85)
		qos := Qos{latency,transmitE,PacketLossrate,reliability}
		confidentiality := RandFloat64(0.7,0.9)
		integrity := RandFloat64(0.8,0.9)
		usability := RandFloat64(0.65,0.9)
		accessControlLevel := RandInt64(3,5)
		qosSec := QoSec{confidentiality,integrity,usability,accessControlLevel}
		C := RandFloat64(0.7,0.95)
		InitV.qos = qos
		InitV.qosSec = qosSec
		InitV.C = C
		InitV.cpu = RandFloat64(0,0.2)
	}else if rnd >=40 && rnd < 90{

		transmitE := RandFloat64(0.4,0.7)
		PacketLossrate := RandFloat64(0.3,0.6)
		reliability := RandFloat64(0.5,0.7)
		qos := Qos{latency,transmitE,PacketLossrate,reliability}
		confidentiality := RandFloat64(0.5,0.7)
		integrity := RandFloat64(0.6,0.8)
		usability := RandFloat64(0.5,0.65)
		accessControlLevel := RandInt64(2,3)
		qosSec := QoSec{confidentiality,integrity,usability,accessControlLevel}
		C := RandFloat64(0.6,0.7)
		InitV.qos = qos
		InitV.qosSec = qosSec
		InitV.C = C
		InitV.cpu = RandFloat64(0.2,0.3)
	}

}


func (trust *nodeJsang) init(nodeCtx *NodeCtx){
	trust.mux.Lock()
	defer trust.mux.Unlock()
	trust.Jsang = make(map[uint]map[[32]byte]*Jsang)
	times := time.Now()
	trust.Jsang[0] = make(map[[32]byte]*Jsang)
	self := nodeCtx.self.Pub.Bytes
	trust.Jsang[0][self] = new(Jsang)
	trust.Jsang[0][self].Time = times
	trust.Jsang[0][self].R_b = make([]float64,3)
	trust.Jsang[0][self].R_b[0] = 0.1
	trust.Jsang[0][self].R_b[1] = 0.1
	trust.Jsang[0][self].R_b[2] = 0.1
	trust.Jsang[0][self].R_u = 0.7
	for node := range nodeCtx.committee.Members{

		trust.Jsang[0][node] = new(Jsang)
		trust.Jsang[0][node].Time = times
		trust.Jsang[0][node].R_b = make([]float64,3)
		trust.Jsang[0][node].R_b[0] = 0.1
		trust.Jsang[0][node].R_b[1] = 0.1
		trust.Jsang[0][node].R_b[2] = 0.1
		trust.Jsang[0][node].R_u= 0.7
	}

}
func (trust *nodeJsang) add(nodePub [32]byte,r uint,jsang Jsang){
	trust.mux.Lock()
	defer trust.mux.Unlock()
	if len(trust.Jsang[r]) ==0 {
		trust.Jsang[r] = make(map[[32]byte]*Jsang)
	}

	trust.Jsang[r][nodePub] = &jsang

}

type nodeTrust struct {
	nodeT sendToNodeTrust
	Af []float64
}






type SelfInfo struct {
	Priv        *PrivKey
	Pub         *PubKey
	CommitteeID [32]byte
	IP          string
	IsHonest    bool
	Debug       bool
	coordinate  Coordinate
	clusterID    int
	trust        float64
}

type NodeAllInfo struct {
	Pub         *PubKey
	CommitteeID [32]byte
	IP          string
	IsHonest    bool
	coordinate  Coordinate
	clusterID    int
	nodetrust    float64
}

type PlaceHolder struct {
	t uint
}

type ResponseToNodes struct {
	Nodes                []NodeAllInfo
	GensisisBlocks       []*FinalBlock
	DebugNode            [32]byte
	ReconfigurationBlock *ReconfigurationBlock
}

type ByteArrayAndTimestamp struct {
	B []byte
	T time.Time
}

// Representation of a member beloning to the current committee of a node
type CommitteeMember struct {
	Pub *PubKey
	IP  string
	IsHonest bool
}

// Representation of a committee from the point of view of a node
type Committee struct {
	ID            [32]byte
	BigIntID      *big.Int
	CurrentLeader *PubKey
	Members       map[[32]byte]*CommitteeMember
	mux           sync.Mutex
	trust         float64
}

func (c *Committee) init(ID [32]byte) {
	c.ID = ID
	c.BigIntID = new(big.Int).SetBytes(ID[:])
	c.Members = make(map[[32]byte]*CommitteeMember)
}

func (c *Committee) addMember(m *CommitteeMember) {
	c.Members[m.Pub.Bytes] = m
}

func (c *Committee) safeAddMember(m *CommitteeMember) bool {
	if _, ok := c.Members[m.Pub.Bytes]; !ok {
		return false
	}
	c.addMember(m)
	return true
}

func (c *Committee) getMemberIDsAsSortedList() [][32]byte {
	newList := make([][32]byte, len(c.Members))
	i := 0
	for _, v := range c.Members {
		newList[i] = v.Pub.Bytes
		i++
	}

	sort.Slice(newList, func(i, j int) bool {
		return toBigInt(newList[i]).Cmp(toBigInt(newList[j])) < 0
	})
	//fmt.Println("Sorted list: ", newList)
	return newList
}

// Recived transactions that have not been included in a block yet
type TxPool struct {
	pool map[[32]byte]*Transaction // TxHash -> Transaction
	mux  sync.Mutex
}

func (t *TxPool) len() uint {
	t.mux.Lock()
	defer t.mux.Unlock()
	return uint(len(t.pool))
}

func (t *TxPool) init() {
	t.pool = make(map[[32]byte]*Transaction)
}

func (t *TxPool) _add(tx *Transaction) {
	t.pool[tx.id()] = tx
}

func (t *TxPool) add(tx *Transaction) {
	t.mux.Lock()
	t._add(tx)
	t.mux.Unlock()
}

func (t *TxPool) safeAdd(tx *Transaction) bool {
	// only add if there is no transaction with same has
	t.mux.Lock()
	defer t.mux.Unlock()
	if _, ok := t.pool[tx.id()]; ok {
		errFatal(nil, "tx allready in tx pool")
		return false
	}
	t.pool[tx.id()] = tx
	return true
}

func (t *TxPool) get(txHash [32]byte) *Transaction {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.pool[txHash]
}

func (t *TxPool) getAll() []*Transaction {
	t.mux.Lock()
	defer t.mux.Unlock()
	txes := make([]*Transaction, len(t.pool))
	i := 0
	for _, tx := range t.pool {
		txes[i] = tx
		i++
	}
	return txes
}

func (t *TxPool) getEnoughToFillblock(blockSize uint) []*Transaction {
	t.mux.Lock()
	defer t.mux.Unlock()
	txes := []*Transaction{}
	size := uint(0)
	for _, tx := range t.pool {
		txes = append(txes, tx)
		tmp := unsafe.Sizeof(*tx)
		size += uint(tmp)
		if size >= default_B {
			break
		}
	}
	log.Printf("Got %d bytes from txpool", size)
	return txes
}

func (t *TxPool) _remove(txHash [32]byte) {
	delete(t.pool, txHash)
}

func (t *TxPool) remove(txHash [32]byte) {
	t.mux.Lock()
	defer t.mux.Unlock()
	t._remove(txHash)
}

func (t *TxPool) pop(txHash [32]byte) (*Transaction, bool) {
	t.mux.Lock()
	defer t.mux.Unlock()
	tx, ok := t.pool[txHash]
	if ok {
		delete(t.pool, txHash)
	}
	return tx, ok
}

func (t *TxPool) _popAll() []*Transaction {
	txes := make([]*Transaction, len(t.pool))
	i := 0
	for _, tx := range t.pool {
		txes[i] = tx
		i++
	}
	t.pool = make(map[[32]byte]*Transaction)
	return txes
}

func (t *TxPool) popAll() []*Transaction {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t._popAll()
}

func (t *TxPool) processBlock(transactions []*Transaction) {
	t.mux.Lock()
	defer t.mux.Unlock()
	for _, tx := range transactions {
		// Remove originaltx, incomming-cross-tx
		t._remove(tx.OrigTxHash)
		t._remove(tx.Hash)
	}
}

// type CrossTxMap struct {
// 	InputTxID         [32]byte
// 	Finished          bool     // indicated wheter or not CrossTxResponseID is filled
// 	CrossTxResponseID [32]byte // Points to UTXO set because the outputs should be there
// 	Nonce             uint
// }

// Keeps track of all cross-txes for an originaltx. Is not needed anymore, but may be usefull for debug purposes, so should be deleted after a while (with processIncommingCrossTxResponse changes).
type CrossTxPool struct {
	// set      map[[32]byte]map[[32]byte]CrossTxMap // OrigTxID -> InputTxID (in other committe) -> CrossTxMap
	original map[[32]byte]*Transaction // hold the original transaction untill final transaction is complete
	mux      sync.Mutex
}

func (ctp *CrossTxPool) init() {
	// ctp.set = make(map[[32]byte]map[[32]byte]CrossTxMap)
	ctp.original = make(map[[32]byte]*Transaction)
}

// adds a transaction to pool
func (ctp *CrossTxPool) addOriginalTx(nodeCtx *NodeCtx, tx *Transaction) {
	// assuming transaction.whatAmI == originaltx
	ctp.mux.Lock()
	defer ctp.mux.Unlock()
	// ctp.set[tx.OrigTxHash] = make(map[[32]byte]CrossTxMap)
	if tx.Hash != [32]byte{} || tx.OrigTxHash == [32]byte{} {
		errFatal(nil, "origtx wrong hashes")
	}
	ctp.original[tx.OrigTxHash] = tx
}


func (ctp *CrossTxPool) getOriginal(origTxID [32]byte) *Transaction {
	ctp.mux.Lock()
	defer ctp.mux.Unlock()
	if t, ok := ctp.original[origTxID]; ok {
		return t
	}
	return nil
}

func (ctp *CrossTxPool) removeOriginal(origTxID [32]byte) {
	ctp.mux.Lock()
	defer ctp.mux.Unlock()
	delete(ctp.original, origTxID)
}


type UTXOSet struct {
	set map[[32]byte]map[uint]*OutTx // TxID -> Nonce -> OutTx
	mux sync.Mutex
}

func (s *UTXOSet) String() string {
	var str string = "[UTXOSet] \n"
	for txID, t := range s.set {
		str += fmt.Sprintf("\t TxID: %s, Len of set: %d\n", bytes32ToString(txID), len(t))
		for n, out := range t {
			str += fmt.Sprintf("\tN: %d, out %s\n", n, out)
		}
	}
	return str
}

func (s *UTXOSet) init() {
	s.set = make(map[[32]byte]map[uint]*OutTx)
}

func (s *UTXOSet) _add(k [32]byte, oTx *OutTx) {
	if len(s.set[k]) == 0 {
		s.set[k] = make(map[uint]*OutTx)
	}
	s.set[k][oTx.N] = oTx
}

func (s *UTXOSet) add(k [32]byte, oTx *OutTx) {
	s.mux.Lock()
	s._add(k, oTx)
	s.mux.Unlock()
}

func (s *UTXOSet) _removeOutput(k [32]byte, N uint) {
	delete(s.set[k], N)
	if len(s.set[k]) == 0 {
		delete(s.set, k)
	}
}

func (s *UTXOSet) removeOutput(k [32]byte, N uint) {
	s.mux.Lock()
	s._removeOutput(k, N)
	s.mux.Unlock()
}

func (s *UTXOSet) _get(k [32]byte, N uint) *OutTx {
	if len(s.set[k]) == 0 {
		// fmt.Println("no key")
		return nil
	}
	v, ok := s.set[k][N]
	if !ok {
		// fmt.Println("no out")
		return nil
	}
	if N != v.N {
		// fmt.Println("Pubkey ", bytesToString(v.PubKey.Bytes[:]))
		errFatal(nil, fmt.Sprintf("map nonce %d not equal to tx nonce %d", N, v.N))
	}
	return v
}

func (s *UTXOSet) get(k [32]byte, N uint) *OutTx {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s._get(k, N)
}

func (s *UTXOSet) verifyNonces() {
	s.mux.Lock()
	defer s.mux.Unlock()

	// fmt.Printf("Total len of UTXO set %d\n", len(s.set))

	for _, t := range s.set {
		// fmt.Printf("Number of output in tx: %d", len(t))
		for k, o := range t {
			if k != o.N {
				fmt.Printf("\nTxID: %s map nonce N %d and outtx N %d\n", bytesToString(o.PubKey.Bytes[:]), k, o.N)
				errFatal(nil, "nonces verifyNonces()")
			}
		}
	}
}

func (s *UTXOSet) _getAndRemove(k [32]byte, N uint) *OutTx {
	ret := s._get(k, N)
	s._removeOutput(k, N)
	return ret
}

func (s *UTXOSet) getAndRemove(k [32]byte, N uint) *OutTx {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s._getAndRemove(k, N)
}

func (s *UTXOSet) getTxOutputsAsList(k [32]byte) *[]*OutTx {
	s.mux.Lock()
	defer s.mux.Unlock()
	if len(s.set[k]) == 0 {
		return nil
	}
	a := make([]*OutTx, len(s.set[k]))
	i := 0
	for _, v := range s.set[k] {
		a[i] = v
	}
	return &a
}

func (s *UTXOSet) getLenOfEntireSet() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	l := 0
	for k := range s.set {
		l += len(s.set[k])
	}
	return l
}

func (s *UTXOSet) _totalValue() uint {
	// finds the total value that is in the UTXO set, usefull for each user to know their balance
	var tot uint = 0
	for _, txid := range s.set {
		for _, nonce := range txid {
			tot += nonce.Value
		}
	}
	return tot
}

func (s *UTXOSet) totalValue() uint {
	// finds the total value that is in the UTXO set, usefull for each user to know their balance
	s.mux.Lock()
	defer s.mux.Unlock()

	var tot uint = 0
	for _, txid := range s.set {
		for _, nonce := range txid {
			tot += nonce.Value
		}
	}
	return tot
}

// only to be used if you own all UTXO's
func (s *UTXOSet) getOutputsToFillValue(value uint) ([]*txIDNonceTuple, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	res := []*txIDNonceTuple{}
	var remV int = int(value)
	for txID := range s.set {
		for nonce := range s.set[txID] {
			v := s.set[txID][nonce].Value
			remV -= int(v)
			tnp := new(txIDNonceTuple)
			tnp.txID = txID
			tnp.n = nonce
			if nonce != s.set[txID][nonce].N {
				errFatal(nil, fmt.Sprintf("nonces not equal %d %d", nonce, s.set[txID][nonce].N))
			}
			if remV > 0 {
				// take entire output
				res = append(res, tnp)
			} else {
				// take only the required amount
				res = append(res, tnp)
				return res, true
			}
		}
	}
	// did not find enough outputs to fill value
	return res, false
}

type txIDNonceTuple struct {
	txID [32]byte
	n    uint
}

type InTx struct {
	TxHash [32]byte // output in transaction
	N      uint     // nonce in output in transaction
	Sig    *Sig     // Sig of TxHash, OrigTxHash, N, TxHash, OrigTxHash
}

func (iTx *InTx) String() string {
	return fmt.Sprintf("[InTx] TxHash: %s, N: %d, Sig: %s", bytes32ToString(iTx.TxHash), iTx.N, bytesToString(iTx.Sig.bytes()))
}

type OutTx struct {
	Value  uint // value of UTXO
	N      uint // nonce/i in output list of tx
	PubKey *PubKey
}

func (o *OutTx) String() string {
	return fmt.Sprintf("[OutTx] Value: %d, N: %d, PubKey: %s", o.Value, o.N, bytes32ToString(o.PubKey.Bytes))
}

func (o *OutTx) bytes() []byte {
	b1 := make([]byte, 8) //uint64 is 8 bytes
	binary.LittleEndian.PutUint64(b1, uint64(o.Value))
	b2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b2, uint64(o.N))
	b3 := o.PubKey.Bytes
	return byteSliceAppend(b1, b2, b3[:])
}

func (iTx *InTx) bytesWithSig() []byte {
	return byteSliceAppend(iTx.bytesExceptSig(), iTx.Sig.bytes())
}

func (iTx *InTx) bytesExceptSig() []byte {
	return byteSliceAppend(iTx.TxHash[:], uintToByte(iTx.N))
}

func (iTx *InTx) getHash(extra [32]byte) [32]byte {
	return hash(byteSliceAppend(iTx.bytesExceptSig(), extra[:]))
}

func (iTx *InTx) sign(extra [32]byte, priv *PrivKey) {
	iTx.Sig = priv.sign(iTx.getHash(extra))
}

type ProofOfConsensus struct {
	GossipHash       [32]byte
	IntermediateHash [32]byte
	MerkleRoot       [32]byte
	MerkleProof      *merkletree.Proof
	Signatures       []*ConsensusMsg
}

func (poc *ProofOfConsensus) String() string {
	str := fmt.Sprintf("[PoC] GossipHash: %s, IntermediateHash: %s\n\tMerkleRoot: %s, MerkleProof: %s,\n\tLen of signatures: %d", bytes32ToString(poc.GossipHash), bytes32ToString(poc.IntermediateHash), bytes32ToString(poc.MerkleRoot), " merkleproofindex: ", poc.MerkleProof.Index)
	return str
}

type Transaction struct {
	Hash             [32]byte // hash of inputs and outputs
	OrigTxHash       [32]byte // see cross-tx
	Inputs           []*InTx
	Outputs          []*OutTx
	ProofOfConsensus *ProofOfConsensus
}

func (t *Transaction) String() string {
	start := fmt.Sprintf("[TX] Hash: %s, OrigTxHash: %s\n", bytes32ToString(t.Hash), bytes32ToString(t.OrigTxHash))
	for _, inp := range t.Inputs {
		start += fmt.Sprintln("\n\t", inp)
	}
	for _, out := range t.Outputs {
		start += fmt.Sprintln("\n\t", out)
	}
	if t.ProofOfConsensus != nil {
		start += fmt.Sprintf("\n\t%s\n", t.ProofOfConsensus)
	}
	return start
}

func (t *Transaction) whatAmI(nodeCtx *NodeCtx) string {
	if t.Hash != [32]byte{} && t.OrigTxHash == [32]byte{} {
		return "normal"
	} else if t.Hash == [32]byte{} && t.OrigTxHash != [32]byte{} && t.Outputs == nil {
		return "crosstx"
	} else if t.Hash == [32]byte{} && t.OrigTxHash != [32]byte{} && t.Outputs != nil {
		return "originaltx"
	} else if t.Hash != [32]byte{} && t.OrigTxHash != [32]byte{} && txFindClosestCommittee(nodeCtx, t.OrigTxHash) != nodeCtx.self.CommitteeID {
		return "crosstxresponse_C_in"
	} else if t.Hash != [32]byte{} && t.OrigTxHash != [32]byte{} && t.ProofOfConsensus != nil {
		return "crosstxresponse_C_out"
	} else if t.Hash != [32]byte{} && t.OrigTxHash != [32]byte{} && txFindClosestCommittee(nodeCtx, t.OrigTxHash) == nodeCtx.self.CommitteeID {
		return "finaltransaction"
	} else {
		errFatal(nil, "unknown transaction type?")
		return "this will never return but compiler is angry"
	}
}

func (t *Transaction) calculateHash() [32]byte {
	b := []byte{}
	for i := range t.Inputs {
		b = append(b, t.Inputs[i].bytesExceptSig()...)
	}
	for i := range t.Outputs {
		b = append(b, t.Outputs[i].bytes()...)
	}
	return hash(byteSliceAppend(b, t.OrigTxHash[:]))
}

// since Hash or OrigTxHash can be nil, we need an identifier for internal functions that will
// work regardless.
func (t *Transaction) id() [32]byte {
	id := [32]byte{}
	if t.Hash != [32]byte{} {
		id = t.Hash
	} else {
		id = t.OrigTxHash
	}
	return id
}

func (t *Transaction) ifOrigRetOrigIfNotRetHash() [32]byte {
	id := [32]byte{}
	if t.OrigTxHash != [32]byte{} {
		id = t.OrigTxHash
	} else {
		id = t.Hash
	}
	return id
}

func (t *Transaction) setHash() {
	t.Hash = t.calculateHash()
}

// Signs all inputs
func (t *Transaction) signInputs(priv *PrivKey) {
	for i := range t.Inputs {
		// sign InTx.TxID and t.Hash
		t.Inputs[i].sign(t.id(), priv)
	}
}

func (t *Transaction) encode() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(t)
	ifErrFatal(err, "transaction encode")
	return buf.Bytes()
}

func (t *Transaction) decode(b []byte) {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(t)
	ifErrFatal(err, "transaction decode")
}

type ProposedBlock struct {
	GossipHash [32]byte
	// since signatures are not added to block we should use the last seen valdid gossipHash
	// Because of synchronity all nodes should have the same signature set, and therefor the
	// hash of the final block should be equal among all nodes. But since synchronity is not
	// practical, we use the previous gossip hash in this implementation. (the paper does not mention
	// such implementation details)
	PreviousGossipHash [32]byte
	Iteration          uint
	CommitteeID        [32]byte
	LeaderPub          *PubKey
	MerkleRoot         [32]byte       // Merkle tree of transactions
	LeaderSig          *Sig           // sig of gossiphash
	Transactions       []*Transaction // not hashed because it is implicitly in MerkleRoot
}

func (b *ProposedBlock) String() string {
	str := fmt.Sprintf("\n[ProposedBlock] GossipHash: %s, PreviousGossipHash: %s, Iteration %d, ", bytes32ToString(b.GossipHash), bytes32ToString(b.PreviousGossipHash), b.Iteration)
	str += fmt.Sprintf("\n\tCommitteeID: %s, LeaderPub: %s, \n\tmerkleroot: %s, LeaderSig: %s\n", bytes32ToString(b.CommitteeID), bytes32ToString(b.LeaderPub.Bytes), bytes32ToString(b.MerkleRoot), bytesToString(b.LeaderSig.bytes()))
	for _, t := range b.Transactions {
		str += fmt.Sprintf("\t%s\n", t)
	}
	return str
}

func (b *ProposedBlock) calculateHash() [32]byte {
	hashExcMR := b.calculateHashExceptMerkleRoot()
	hashMr := b.calculateHashOfMerkleRoot()

	return hash(byteSliceAppend(hashExcMR[:], hashMr[:]))
}

func (b *ProposedBlock) calculateHashExceptMerkleRoot() [32]byte {
	pgb := b.PreviousGossipHash[:]
	i := uintToByte(b.Iteration)
	c := b.CommitteeID[:]
	lp := getBytes(b.LeaderPub)
	return hash(byteSliceAppend(pgb, i, c, lp))
}

func (b *ProposedBlock) calculateHashOfMerkleRoot() [32]byte {
	return hash(b.MerkleRoot[:])
}

func (b *ProposedBlock) isHashesCorrect() bool {
	rest := b.calculateHashExceptMerkleRoot()
	mr := b.calculateHashOfMerkleRoot()
	together := hash(byteSliceAppend(rest[:], mr[:]))

	h := b.calculateHash()

	// fmt.Println(together, "\n", h, "\n")

	// fmt.Println(bytesToString(together[:]), "\n", bytesToString(h[:]))
	if bytesToString(together[:]) == bytesToString(h[:]) {
		return true
	}
	return false
}

func (b *ProposedBlock) setHash() {
	b.GossipHash = b.calculateHash()
}

func (b *ProposedBlock) encode() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(b)
	ifErrFatal(err, "Proposed block encode")
	return buf.Bytes()
}

func (b *ProposedBlock) decode(bArr []byte) {
	buf := bytes.NewBuffer(bArr)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(b)
	ifErrFatal(err, "Proposed block decode")
}

// The final block recorded by each member.
// Because of synchronity, the signature set is equal among all nodes
type FinalBlock struct {
	ProposedBlock *ProposedBlock
	Signatures    []*ConsensusMsg
}

// processes the final block by changing the UTXO set and remove transaction from tx pool
func (b *FinalBlock) processBlock(nodeCtx *NodeCtx) {
	// assume that signatures and so on are valid because the block has gone trough consensus
	// lock utxoSet because we are going to do a lot of changes that must be atomic with the respect to the block
	nodeCtx.utxoSet.mux.Lock()
	// fmt.Println("Processing block")
	// we do not want to proccess new cross-tx'es, or original tx'es (UTXO's in original TX are still spendable)
	for _, t := range b.ProposedBlock.Transactions {
		what := t.whatAmI(nodeCtx)
		// fmt.Println("processBlock: ", what)
		// fmt.Println(t)
		if what == "normal" {
			var tot uint = 0
			var totOut uint = 0
			// spend inputs
			for _, inp := range t.Inputs {
				UTXO := nodeCtx.utxoSet._getAndRemove(inp.TxHash, inp.N)
				tot += UTXO.Value
			}
			for _, out := range t.Outputs {
				nodeCtx.utxoSet._add(t.Hash, out)
				totOut += out.Value
				// fmt.Printf("%s Added UTXO with N %d, Value %d and Pub %s\n", bytesToString(nodeCtx.self.CommitteeID[:]), out.N, out.Value, bytesToString(out.PubKey.Bytes[:]))
			}
			// if signatures is nil, then this is the genesis block
			// check if total input == total output, except if this is the gensis block (then signatures will be nil)
			if tot != totOut && b.Signatures != nil {
				errFatal(nil, fmt.Sprintf("Spent value %d not equal to new unspent value %d", tot, totOut))
			}
			continue
		} else if what == "crosstx" {
			// do nothing (no inputs in this committee)
			continue
		} else if what == "originaltx" {
			// add inputs to special map so we can keep track of originalTx and its cross-txes
			nodeCtx.crossTxPool.addOriginalTx(nodeCtx, t)
			continue
		} else if what == "crosstxresponse_C_in" {
			// spend inputs, but do not add outputs, because they belong in another committee
			for _, inp := range t.Inputs {
				nodeCtx.utxoSet._removeOutput(inp.TxHash, inp.N)
			}
			continue
		} else if what == "crosstxresponse_C_out" {
			// add outputs, but do not do anything with inputs, because they come from another comittee
			if len(t.Inputs) != len(t.Outputs) {
				errFatal(nil, "input n output length not equal idkrn")
			}
			for i := range t.Outputs {
				nodeCtx.utxoSet._add(t.Inputs[i].TxHash, t.Outputs[i])
			}
			// nodeCtx.crossTxPool.addResponses(t)
			continue
		} else if what == "finaltransaction" {
			// get crossTxMap
			// all inputs in crossTxMap corresponds to finaltransaction and original transaction
			// crossMap := nodeCtx.crossTxPool.getMap(t.OrigTxHash)
			original := nodeCtx.crossTxPool.getOriginal(t.OrigTxHash)
			if original == nil {
				errFatal(nil, fmt.Sprint("original was nil", original))
			}

			// outputs should be exactly the same in final and original
			for _, out := range t.Outputs {
				var found bool = false
				for _, origOut := range original.Outputs {
					if out.Value == origOut.Value && out.N == origOut.N && out.PubKey.Bytes == origOut.PubKey.Bytes {
						found = true
						break
					}
				}
				if !found {
					errFatal(nil, "outputs not equal")
				}
			}

			// var tot uint = 0
			var totOut uint = 0
			// spend inputs
			for _, inp := range t.Inputs {
				nodeCtx.utxoSet._getAndRemove(inp.TxHash, inp.N)
				// tot += UTXO.Value
			}

			if t.OrigTxHash != original.OrigTxHash || t.OrigTxHash == [32]byte{} {
				errFatal(nil, "orighasshes not equal akjb3")
			}

			// add outputs
			for _, out := range t.Outputs {
				nodeCtx.utxoSet._add(t.OrigTxHash, out)
				totOut += out.Value
			}

			// if tot != totOut {
			// 	errFatal(nil, fmt.Sprintf("finaltransaction: Spent value %d not equal to new unspent value %d", tot, totOut))
			// }

			// remove original tx and crosstxmap from crossTxPool
			nodeCtx.crossTxPool.removeOriginal(t.OrigTxHash)
			// nodeCtx.crossTxPool.removeMap(t.OrigTxHash)
			continue
		} else {
			errFatal(nil, "unknow whatAmI ")
		}
	}

	// remove transactions from tx pool
	nodeCtx.txPool.processBlock(b.ProposedBlock.Transactions)

	// print utxo set
	// fmt.Println(nodeCtx.utxoSet)

	nodeCtx.utxoSet.mux.Unlock()
}

// forces the processing of a block without checking for valid UTXOs. (This is valid only if signature set is valid)
func (b *FinalBlock) forceProcessBlock(nodeCtx *NodeCtx) {
	// assume that signatures and so on are valid because the block has gone trough consensus
	// lock utxoSet because we are going to do a lot of changes that must be atomic with the respect to the block
	nodeCtx.utxoSet.mux.Lock()
	// fmt.Println("Processing block")
	// we do not want to proccess new cross-tx'es, or original tx'es (UTXO's in original TX are still spendable)
	for _, t := range b.ProposedBlock.Transactions {
		what := t.whatAmI(nodeCtx)
		// fmt.Println("processBlock: ", what)
		// fmt.Println(t)
		if what == "normal" {
			// spend inputs
			for _, inp := range t.Inputs {
				nodeCtx.utxoSet._removeOutput(inp.TxHash, inp.N)
			}
			for _, out := range t.Outputs {
				nodeCtx.utxoSet._add(t.Hash, out)
				// fmt.Printf("%s Added UTXO with N %d, Value %d and Pub %s\n", bytesToString(nodeCtx.self.CommitteeID[:]), out.N, out.Value, bytesToString(out.PubKey.Bytes[:]))
			}
			// if signatures is nil, then this is the genesis block
			// check if total input == total output, except if this is the gensis block (then signatures will be nil)
			continue
		} else if what == "crosstx" {
			// do nothing (no inputs in this committee)
			continue
		} else if what == "originaltx" {
			// add inputs to special map so we can keep track of originalTx and its cross-txes
			nodeCtx.crossTxPool.addOriginalTx(nodeCtx, t)
			continue
		} else if what == "crosstxresponse_C_in" {
			// spend inputs, but do not add outputs, because they belong in another committee
			for _, inp := range t.Inputs {
				nodeCtx.utxoSet._removeOutput(inp.TxHash, inp.N)
			}
			continue
		} else if what == "crosstxresponse_C_out" {
			// add outputs, but do not do anything with inputs, because they come from another comittee
			if len(t.Inputs) != len(t.Outputs) {
				errFatal(nil, "input n output length not equal idkrn")
			}
			for i := range t.Outputs {
				nodeCtx.utxoSet._add(t.Inputs[i].TxHash, t.Outputs[i])
			}
			// nodeCtx.crossTxPool.addResponses(t)
			continue
		} else if what == "finaltransaction" {

			for _, inp := range t.Inputs {
				nodeCtx.utxoSet._removeOutput(inp.TxHash, inp.N)
			}

			// add outputs
			for _, out := range t.Outputs {
				nodeCtx.utxoSet._add(t.OrigTxHash, out)
			}

			// remove original tx and crosstxmap from crossTxPool
			nodeCtx.crossTxPool.removeOriginal(t.OrigTxHash)
			continue
		} else {
			errFatal(nil, "unknow whatAmI ")
		}
	}

	// remove transactions from tx pool
	nodeCtx.txPool.processBlock(b.ProposedBlock.Transactions)

	// print utxo set
	// fmt.Println(nodeCtx.utxoSet)

	nodeCtx.utxoSet.mux.Unlock()
}

type ConsensusMsg struct {
	GossipHash [32]byte
	Tag        string // propose, echo, accept or pending
	Pub        *PubKey
	Sig        *Sig // Sig of the hash of the above
}

func (cMsg *ConsensusMsg) String() string {
	return fmt.Sprintf("[cMsg] GossipHash: %s, Tag: %s, Pubkey: %s, Sig: %s\n ", bytes32ToString(cMsg.GossipHash), cMsg.Tag, bytes32ToString(cMsg.Pub.Bytes), bytesToString(cMsg.Sig.bytes()))
}

func (cMsg *ConsensusMsg) calculateHash() [32]byte {
	b := byteSliceAppend(cMsg.GossipHash[:], []byte(cMsg.Tag), cMsg.Pub.Bytes[:])
	return hash(b)
}

func (cMsg *ConsensusMsg) sign(pk *PrivKey) {
	cMsg.Sig = pk.sign(cMsg.calculateHash())
}

type ConsensusMsgs struct {
	m   map[[32]byte]map[[32]byte]*ConsensusMsg // GossipHash -> Pub.Bytes -> msg
	mux sync.Mutex
}

func (cMsgs *ConsensusMsgs) init() {
	cMsgs.m = make(map[[32]byte]map[[32]byte]*ConsensusMsg)
}

func (cMsgs *ConsensusMsgs) _initGossipHash(gh [32]byte) {
	cMsgs.m[gh] = make(map[[32]byte]*ConsensusMsg)
}

func (cMsgs *ConsensusMsgs) initGossipHash(gh [32]byte) {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()
	cMsgs._initGossipHash(gh)
}

func (cMsgs *ConsensusMsgs) _exists(gh [32]byte) bool {
	_, ok := cMsgs.m[gh]
	return ok
}
func (cMsgs *ConsensusMsgs) exists(gh [32]byte) bool {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()
	return cMsgs._exists(gh)
}

func (cMsgs *ConsensusMsgs) _hasMsgFrom(gh [32]byte, ID [32]byte) bool {
	_, ok := cMsgs.m[gh][ID]
	return ok
}
func (cMsgs *ConsensusMsgs) hasMsgFrom(gh [32]byte, ID [32]byte) bool {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()
	return cMsgs._hasMsgFrom(gh, ID)
}

func (cMsgs *ConsensusMsgs) _add(gh [32]byte, ID [32]byte, cMsg *ConsensusMsg) {
	if !cMsgs._exists(gh) {
		cMsgs._initGossipHash(gh)
	}
	cMsgs.m[gh][ID] = cMsg
}

func (cMsgs *ConsensusMsgs) add(gh [32]byte, ID [32]byte, cMsg *ConsensusMsg) {
	cMsgs.mux.Lock()
	if !cMsgs._exists(gh) {
		cMsgs._initGossipHash(gh)
	}
	cMsgs.m[gh][ID] = cMsg
	cMsgs.mux.Unlock()
}

func (cMsgs *ConsensusMsgs) _safeAdd(gh [32]byte, ID [32]byte, cMsg *ConsensusMsg) bool {
	if cMsgs._hasMsgFrom(gh, ID) {
		return false
	}
	cMsgs._add(gh, ID, cMsg)
	return true
}

func (cMsgs *ConsensusMsgs) safeAdd(gh [32]byte, ID [32]byte, cMsg *ConsensusMsg) bool {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()
	return cMsgs._safeAdd(gh, ID, cMsg)
}

func (cMsgs *ConsensusMsgs) getConsensusMsg(gh [32]byte, ID [32]byte) *ConsensusMsg {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()
	return cMsgs.m[gh][ID]
}

func (cMsgs *ConsensusMsgs) _getAllConsensusMsgs(gh [32]byte) []*ConsensusMsg {
	ret := make([]*ConsensusMsg, len(cMsgs.m[gh]))
	iter := 0
	for _, v := range cMsgs.m[gh] {
		ret[iter] = v
		iter++
	}
	return ret
}

func (cMsgs *ConsensusMsgs) getAllConsensusMsgs(gh [32]byte) []*ConsensusMsg {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()
	return cMsgs._getAllConsensusMsgs(gh)
}

func (cMsgs *ConsensusMsgs) pop(gh [32]byte) []*ConsensusMsg {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()
	ret := cMsgs._getAllConsensusMsgs(gh)
	// reset
	cMsgs._initGossipHash(gh)
	return ret
}

// Counts valid votes
// Votes are valid if the iteration is I and Tag is echo or accepted
// Votes are valid if the iteration is below I and Tag is accepted
func (cMsgs *ConsensusMsgs) countValidVotes(gh [32]byte) int {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()

	votes := 0
	for _, v := range cMsgs.m[gh] {
		if v.Tag == "echo" || v.Tag == "accept" {
			votes++
		}
	}
	return votes
}

// Counts valid accepts
// Votes are valid if the Tag is accepted
func (cMsgs *ConsensusMsgs) countValidAccepts(gh [32]byte) int {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()

	votes := 0
	for _, v := range cMsgs.m[gh] {
		if v.Tag == "accept" {
			votes++
		}
	}
	return votes
}

// add a getAllConsensusMsg that are votes in this iteration?

func (cMsgs *ConsensusMsgs) getLen(gh [32]byte) uint {
	cMsgs.mux.Lock()
	defer cMsgs.mux.Unlock()
	return uint(len(cMsgs.m[gh]))
}

// Todo define, extend and create reconfiguration block
type ReconfigurationBlock struct {
	Hash       [32]byte
	Committees map[[32]byte]*Committee
	Randomness [32]byte
}

func (rb *ReconfigurationBlock) init() {
	rb.Committees = make(map[[32]byte]*Committee)
}

func (rb *ReconfigurationBlock) calculateHash() [32]byte {
	// calculate hash of all committee ids, all comittee members public key and randomness
	var toHash []byte = rb.Randomness[:]
	for _, committee := range rb.Committees {
		toHash = append(toHash, committee.ID[:]...)
		for _, member := range committee.Members {
			toHash = append(toHash, member.Pub.Bytes[:]...)
		}
	}
	return hash(toHash)
}

func (rb *ReconfigurationBlock) setHash() {
	rb.Hash = rb.calculateHash()
}

type Blockchain struct {
	CommitteeID           [32]byte
	Blocks                map[[32]byte]*FinalBlock    // GossipHash -> FinalBlock
	LatestBlock           [32]byte                    // GossipHash
	ProposedBlocks        map[[32]byte]*ProposedBlock // GossipHash -> ProposedBlock
	ReconfigurationBlocks []*ReconfigurationBlock
	mux                   sync.Mutex
}

func (b *Blockchain) init(committeeID [32]byte) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.CommitteeID = committeeID
	b.Blocks = make(map[[32]byte]*FinalBlock)
	b.ProposedBlocks = make(map[[32]byte]*ProposedBlock)
	b.ReconfigurationBlocks = []*ReconfigurationBlock{}
}

func (b *Blockchain) _getLatest() *FinalBlock {
	return b.Blocks[b.LatestBlock]
}

func (b *Blockchain) getLatest() *FinalBlock {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b._getLatest()
}

func (b *Blockchain) getByIteration(iteration uint64) *FinalBlock {
	b.mux.Lock()
	defer b.mux.Unlock()

	currentBlock := b.LatestBlock
	for {
		block := b.Blocks[currentBlock]
		if uint64(block.ProposedBlock.Iteration) <= iteration {
			return block
		} else if block.ProposedBlock.PreviousGossipHash == [32]byte{} {
			// we are at the genisis block
			return block
		}
		currentBlock = block.ProposedBlock.PreviousGossipHash
	}

}

func (b *Blockchain) _add(block *FinalBlock) {
	b.Blocks[block.ProposedBlock.GossipHash] = block
	b.LatestBlock = block.ProposedBlock.GossipHash
}
func (b *Blockchain) add(block *FinalBlock) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b._add(block)
}

func (b *Blockchain) _isSafe(block *FinalBlock) bool {
	_, ok := b.Blocks[block.ProposedBlock.GossipHash]
	return ok
}

func (b *Blockchain) _safeAdd(block *FinalBlock) bool {
	if !b._isSafe(block) {
		return false
	}
	b._add(block)
	return true
}

func (b *Blockchain) safeAdd(block *FinalBlock) bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b._safeAdd(block)
}

func (b *Blockchain) _isBlock(gh [32]byte) bool {
	_, ok := b.Blocks[gh]
	return ok
}

func (b *Blockchain) isBlock(gh [32]byte) bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b._isBlock(gh)
}

func (b *Blockchain) _getProposedBlock(gh [32]byte) *ProposedBlock {
	if !b._isProposedBlock(gh) {
		return nil
	}
	return b.ProposedBlocks[gh]
}

func (b *Blockchain) getProposedBlock(gh [32]byte) *ProposedBlock {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b._getProposedBlock(gh)
}

func (b *Blockchain) popProposedBlock(gh [32]byte) *ProposedBlock {
	b.mux.Lock()
	defer b.mux.Unlock()
	ret := b._getProposedBlock(gh)
	delete(b.ProposedBlocks, ret.GossipHash)
	return ret
}

func (b *Blockchain) _addProposedBlock(block *ProposedBlock) {
	b.ProposedBlocks[block.GossipHash] = block
}

func (b *Blockchain) _isProposedBlock(gossipHash [32]byte) bool {
	_, ok := b.ProposedBlocks[gossipHash]
	return ok
}

func (b *Blockchain) isProposedBlock(gossipHash [32]byte) bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b._isProposedBlock(gossipHash)
}

func (b *Blockchain) _safeAddProposedBlock(block *ProposedBlock) bool {
	if !b._isProposedBlock(block.GossipHash) {
		return false
	}
	b._addProposedBlock(block)
	return true
}

func (b *Blockchain) addProposedBlock(block *ProposedBlock) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b._addProposedBlock(block)
}

func (b *Blockchain) safeAddProposedBlock(block *ProposedBlock) bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b._safeAddProposedBlock(block)
}

func (b *Blockchain) _addRecBlock(block *ReconfigurationBlock) {
	b.ReconfigurationBlocks = append(b.ReconfigurationBlocks, block)
}

func (b *Blockchain) addRecBlock(block *ReconfigurationBlock) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b._addRecBlock(block)
}

// ensures that you do not add same block twice
func (b *Blockchain) safeAddRecBlock(block *ReconfigurationBlock) bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	if b.ReconfigurationBlocks[len(b.ReconfigurationBlocks)-1].Hash == block.Hash {
		return false
	}
	b._addRecBlock(block)
	return true
}

func (b *Blockchain) _getLastReconfigurationBlock() *ReconfigurationBlock {
	return b.ReconfigurationBlocks[len(b.ReconfigurationBlocks)-1]
}

func (b *Blockchain) getLastReconfigurationBlock() *ReconfigurationBlock {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b._getLastReconfigurationBlock()
}

// routing table
type RoutingTable struct {
	l   []Committee // Your known commites, sorted by distance
	mux sync.Mutex
}

func (r *RoutingTable) init(length int) {
	r.mux.Lock()
	r.l = make([]Committee, length)
	r.mux.Unlock()
}

func (r *RoutingTable) addCommittee(i uint, ID [32]byte) {
	r.mux.Lock()
	r.l[i] = Committee{}
	r.l[i].init(ID)
	r.mux.Unlock()
}

func (r *RoutingTable) addMember(i uint, cm *CommitteeMember) {
	r.mux.Lock()
	r.l[i].addMember(cm)
	r.mux.Unlock()
}

func (r *RoutingTable) get() []Committee {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.l
}

type KademliaFindNodeMsg struct {
	ID [32]byte
}

type KademliaFindNodeResponse struct {
	Committee Committee
}

type IdaMsgs struct {
	m   map[[32]byte][]IDAGossipMsg
	mux sync.Mutex
}

func (ida *IdaMsgs) init() {
	ida.m = make(map[[32]byte][]IDAGossipMsg)
}

func (ida *IdaMsgs) _isArr(root [32]byte) bool {
	_, ok := ida.m[root]
	return ok
}

func (ida *IdaMsgs) isArr(root [32]byte) bool {
	ida.mux.Lock()
	defer ida.mux.Unlock()
	return ida._isArr(root)
}

func (ida *IdaMsgs) _add(root [32]byte, m IDAGossipMsg) {
	if arr, ok := ida.m[root]; !ok {
		ida.m[root] = []IDAGossipMsg{m}
	} else {
		ida.m[root] = append(arr, m)
	}
}

func (ida *IdaMsgs) add(root [32]byte, m IDAGossipMsg) {
	ida.mux.Lock()
	ida._add(root, m)
	ida.mux.Unlock()
}

func (ida *IdaMsgs) _getMsg(root [32]byte, index uint) IDAGossipMsg {
	return ida.m[root][index]
}

func (ida *IdaMsgs) getMsg(root [32]byte, index uint) IDAGossipMsg {
	ida.mux.Lock()
	defer ida.mux.Unlock()
	return ida._getMsg(root, index)
}

func (ida *IdaMsgs) _getMsgs(root [32]byte) []IDAGossipMsg {
	return ida.m[root]
}

func (ida *IdaMsgs) getMsgs(root [32]byte) []IDAGossipMsg {
	ida.mux.Lock()
	defer ida.mux.Unlock()
	return ida._getMsgs(root)
}

func (ida *IdaMsgs) _getLenOfChunks(root [32]byte) int {

	isIndex := make([]bool, default_kappa+default_parity)
	for _, elem := range ida.m[root] {
		for i, _ := range elem.Chunks {
			// gather leaf node position from proof
			index := elem.Proofs[i].Index
			isIndex[index] = true
		}
	}

	tot := 0
	for _, index := range isIndex {
		if index {
			tot++
		}
	}

	return tot
}

func (ida *IdaMsgs) getLenOfChunks(root [32]byte) int {
	ida.mux.Lock()
	defer ida.mux.Unlock()
	return ida._getLenOfChunks(root)
}

func (ida *IdaMsgs) _getLen() uint {
	return uint(len(ida.m))
}

func (ida *IdaMsgs) getLen() uint {
	ida.mux.Lock()
	defer ida.mux.Unlock()
	return ida._getLen()
}

type ReconstructedIdaMsgs struct {
	m   map[[32]byte][][]byte
	mux sync.Mutex
}

func (b *ReconstructedIdaMsgs) init() {
	b.m = make(map[[32]byte][][]byte)
}

func (b *ReconstructedIdaMsgs) keyExists(root [32]byte) bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	_, ok := b.m[root]
	return ok
}

func (b *ReconstructedIdaMsgs) add(root [32]byte, block [][]byte) {
	b.mux.Lock()
	b.m[root] = block
	b.mux.Unlock()
}

func (b *ReconstructedIdaMsgs) safeAdd(root [32]byte, block [][]byte) bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	if _, ok := b.m[root]; ok {
		return false
	}
	b.m[root] = block
	return true
}

func (b *ReconstructedIdaMsgs) get(root [32]byte) [][]byte {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.m[root]
}

func (b *ReconstructedIdaMsgs) pop(root [32]byte) [][]byte {
	b.mux.Lock()
	defer b.mux.Unlock()
	ret := b.m[root]
	delete(b.m, root)
	return ret
}

func (b *ReconstructedIdaMsgs) getLen() uint {
	b.mux.Lock()
	defer b.mux.Unlock()
	return uint(len(b.m))
}

func (b *ReconstructedIdaMsgs) getData(root [32]byte) []byte {
	b.mux.Lock()
	defer b.mux.Unlock()
	// get data, flatten it, and unpadd
	data := b.m[root][:default_kappa]
	bArr := []byte{}
	for _, chunk := range data {
		bArr = append(bArr, chunk...)
	}

	if isPadded(bArr) {
		return unPad(bArr)
	}
	return bArr
}

func (b *ReconstructedIdaMsgs) popData(root [32]byte) []byte {
	b.mux.Lock()
	defer b.mux.Unlock()
	data := b.m[root][:default_kappa]
	bArr := []byte{}
	for _, chunk := range data {
		bArr = append(bArr, chunk...)
	}
	delete(b.m, root)
	if isPadded(bArr) {
		return unPad(bArr)
	}
	return bArr
}

type Channels struct {
	echoChan chan bool
}

func (c *Channels) init(l int) {
	c.echoChan = make(chan bool, l)
}
type trustLocalGroup struct {
	trust map[uint]map[[32]byte]map[[32]byte]*TrustVector
	mux   sync.Mutex
}
type committeeTrust struct {
	committeeID [32]byte
	sumTrust    float64
}
type nodeMsg struct {
	nodePub   *PubKey
	IP      string
}
type CurrentIteration struct {
	i   uint
	flag map[uint]bool
	mux sync.Mutex
}

func (c *CurrentIteration) add() {
	c.mux.Lock()
	c.i++
	c.mux.Unlock()
}

func (c *CurrentIteration) getI() uint {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.i
}

// context for a node, to be passed everywhere, acts like a global var
type NodeCtx struct {
	flagArgs             FlagArgs
	committee            Committee  // current committee
	neighbors            [][32]byte // neighboring nodes
	self                 SelfInfo
	allInfo              map[[32]byte]NodeAllInfo // cheat variable for easy testing
	idaMsgs              IdaMsgs
	reconstructedIdaMsgs ReconstructedIdaMsgs
	consensusMsgs        ConsensusMsgs
	channels             Channels
	i                    CurrentIteration
	routingTable         RoutingTable
	committeeList        [][32]byte //list of all committee ids, to be replaced with reference block?
	txPool               TxPool
	crossTxPool          CrossTxPool
	utxoSet              *UTXOSet
	blockchain           Blockchain
	nodeTrust            selfTrust
	Initvalue            InitValue
	nodeJs               nodeJsang
	decision     		 DecisionVector
	failureType          string
}

func (nc *NodeCtx) amILeader() bool {
	if nc.self.Pub.Bytes == nc.committee.CurrentLeader.Bytes {
		return true
	}
	return false
}

// generic msg. typ indicates which struct to decode msg to.
type Msg struct {
	Typ     string
	Msg     interface{}
	FromPub *PubKey
}
// 生成范围随机float64
func RandFloat64(min, max float64) float64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	minStr := strconv.FormatFloat(min, 'f', -1, 64)
	// 不包含小数点
	if strings.Index(minStr, ".") == -1 {
		return max
	}
	multipleNum := len(minStr) - (strings.Index(minStr, ".") + 1)
	multiple := math.Pow10(multipleNum)
	minMult := min * multiple
	maxMult := max * multiple
	randVal := RandInt64(int64(minMult), int64(maxMult))
	result := float64(randVal) / multiple
	return result
}

//随机整数
func RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	//	time.Sleep(100*time.Millisecond)
	rand.Seed(time.Now().UnixNano())
	return rand.Int63n(max - min + 1) + min
}


