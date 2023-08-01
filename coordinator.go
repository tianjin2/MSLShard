package main

import (
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"io/ioutil"
	"path"
)

type InitialMessageToCoordinator struct {
	pub *PubKey
	ip  string
}

type committeeInfo struct {
	id  [32]byte
	npm uint
	f   int
}

type consensusResult struct {
	echos, pending, accepts int
}

// measure routing of tx
type routetxresults struct {
	start      time.Time              // first node recives transaction
	end        time.Time              // first node in target committee recives tx
	committees map[[32]byte]time.Time // first node in intermediary committee recives tx
	mux        sync.Mutex
}

func (r *routetxresults) init() {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.committees == nil {
		r.committees = make(map[[32]byte]time.Time)
	}
}

// adds a committee with timestamp if it does not allready exist
func (r *routetxresults) add(cId [32]byte, tim time.Time) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.committees == nil {
		r.init()
	} else if _, ok := r.committees[cId]; ok {
		return false
	}
	r.committees[cId] = tim
	return true
}

// adds start timestamp only if it has not been added before
func (r *routetxresults) addStart(tim time.Time) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.start.IsZero() {
		r.start = tim
		return true
	}
	return false
}

// adds end timestamp only if it has not been added before
func (r *routetxresults) addEnd(tim time.Time) bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.end.IsZero() {
		r.end = tim
		return true
	}
	return false
}

type routetxmap struct {
	m   map[[32]byte]*routetxresults
	mux sync.Mutex
}

func (r *routetxmap) add(ID [32]byte) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.m[ID] != nil {
		return
	}
	r.m[ID] = new(routetxresults)
}

func (r *routetxmap) get(ID [32]byte) *routetxresults {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.m[ID]
}

type IDAGossipResultsMap struct {
	m   map[[32]byte]*IDAGossipResults
	mux sync.Mutex
}

func (ida *IDAGossipResultsMap) add(ID [32]byte) {
	ida.mux.Lock()
	defer ida.mux.Unlock()
	if ida.m[ID] != nil {
		return
	}
	ida.m[ID] = new(IDAGossipResults)

}

func (ida *IDAGossipResultsMap) get(ID [32]byte) *IDAGossipResults {
	ida.mux.Lock()
	defer ida.mux.Unlock()
	return ida.m[ID]
}

// measure routing of tx
type IDAGossipResults struct {
	start         time.Time   // first node recives transaction
	reconstructed []time.Time // first node in target committee recives tx
	mux           sync.Mutex
}

// adds the timestamp of the reconstructed msg, return true if it is the first msg
func (ida *IDAGossipResults) addReconstructed(tim time.Time) bool {
	ida.mux.Lock()
	defer ida.mux.Unlock()
	if len(ida.reconstructed) == 0 {
		ida.reconstructed = make([]time.Time, 1)
		ida.reconstructed[0] = tim
		return true
	}
	ida.reconstructed = append(ida.reconstructed, tim)
	return false
}

func launchCoordinator(flagArgs *FlagArgs) {
	/*
		The coordinator should listen to incoming connections untill it has recived n different ids
		Then it should create:
			a map of identity <-> ip
			a map of committee <-> identity
			a map of identity <-> honest or malicious.
			an initial randomness
		These variables should then be sent to every node.
	*/

	// To be used to send ID and IP from node connection to coordinator
	chanToCoordinator := make(chan InitialMessageToCoordinator, flagArgs.n)

	// To be used to send result back to node connection
	chanToNodes := make([]chan ResponseToNodes, flagArgs.n)
	for i := uint(0); i < flagArgs.n; i++ {
		chanToNodes[i] = make(chan ResponseToNodes)
	}
	allCommittee := make(map[[32]byte][]*nodeMsg)
	// waitgroup for all node connections to have recived an ID
	var wg sync.WaitGroup
	wg.Add(int(flagArgs.n))
	committeeT := make(chan committeeTrust,flagArgs.m*2)
	// waitgroup for when coordinator is done and sent all data to connections
	var wg_done sync.WaitGroup
	wg_done.Add(int(flagArgs.n))

	rand.Seed(1337)

	finalBlockChan := make(chan FinalBlock, flagArgs.m*2)

	var err error
	files := make([]*os.File, 6)
	_, err = os.Stat("results")
	if err == nil {
		dir, _ := ioutil.ReadDir("results")
		for _, d := range dir {
			fmt.Println(d.Name())
			os.RemoveAll(path.Join([]string{"results", d.Name()}...))
		}
	}
	files[0], err = os.Create("results/tx" + time.Now().Format("-20060102150405") + ".csv") //time.Now().String()
	ifErrFatal(err, "txresfile")
	files[1], err = os.Create("results/pocverify" + time.Now().Format("-20060102150405") + ".csv")
	ifErrFatal(err, "pocverifyfile")
	files[2], err = os.Create("results/pocadd" + time.Now().Format("-20060102150405") + ".csv")
	ifErrFatal(err, "pocaddfile")
	files[3], err = os.Create("results/routing" + time.Now().Format("-20060102150405") + ".csv")
	ifErrFatal(err, "routing")
	files[4], err = os.Create("results/ida" + time.Now().Format("-20060102150405") + ".csv")
	ifErrFatal(err, "ida")
	files[5], err = os.Create("results/consensusacceptfail" + time.Now().Format("-20060102150405") + ".csv")
	ifErrFatal(err, "consensusacceptfail")
	for _, f := range files {
		defer f.Close()
	}
	nodeT := make(map[[32]byte]sendToNodeTrust)
	isSendTx := make(chan bool, 1)
	go coordinator(chanToCoordinator, chanToNodes, &wg, flagArgs,finalBlockChan,  allCommittee,files,committeeT,isSendTx,nodeT )

	listener, err := net.Listen("tcp", ":8080")
	ifErrFatal(err, "tcp listen on port 8080")

	var i uint = 0
	// block main and listen to all incoming connections
	for i < flagArgs.n {

		// accept new connection
		conn, err := listener.Accept()
		ifErrFatal(err, "tcp accept")

		// spawn off goroutine to able to accept new connections
		go coordinatorHandleConnection(conn, chanToCoordinator, chanToNodes[i], &wg, &wg_done)

		if flagArgs.n > 20 && i%(flagArgs.n/10) == 0 {
			fmt.Printf("#connections: %d", i)
		}
		i += 1
	}
	wg_done.Wait()
	log.Println("Coordination executed")
	trustVectorGroup := new(trustLocalGroup)
	trustVectorGroup.trust = make(map[uint]map[[32]byte]map[[32]byte]*TrustVector)
	// merkleroot -> number of nodes succesfully recreated it

	// routetx map
	// txid -> committeeid ->
	routetxmap := new(routetxmap)
	routetxmap.m = make(map[[32]byte]*routetxresults)

	idaresults := new(IDAGossipResultsMap)
	idaresults.m = make(map[[32]byte]*IDAGossipResults)


	// start listening for debug/stats
	for {
		// accept new connection
		conn, err := listener.Accept()
		ifErrFatal(err, "tcp accept")
		// spawn off goroutine to able to accept new connections
		go coordinatorDebugStatsHandleConnection(conn,trustVectorGroup, committeeT, finalBlockChan, files,routetxmap , idaresults, allCommittee, nodeT)
	}
}

func coordinatorHandleConnection(conn net.Conn,
	chanToCoordinator chan<- InitialMessageToCoordinator,
	chanFromCoordinator <-chan ResponseToNodes,
	wg, wg_done *sync.WaitGroup) {

	dec := gob.NewDecoder(conn)
	rec_msg := new(Node_InitialMessageToCoordinator)
	err := dec.Decode(rec_msg)
	ifErrFatal(err, "decoding")

	// get the remote address of the client
	clientAddr := conn.RemoteAddr().String()
	// remove port number and add rec_msg.Port instead
	fmt.Println("1: ", clientAddr)
	clientAddr = fmt.Sprintf("%s:%d", clientAddr[:strings.IndexByte(clientAddr, ':')], rec_msg.Port)
	//fmt.Println("2: ", clientAddr)

	chanToCoordinator <- InitialMessageToCoordinator{rec_msg.Pub, clientAddr}

	// signalize to waitgroup that this connection has recived an ID
	wg.Done()

	fmt.Println("waiting for returnMessage")
	returnMessage := <-chanFromCoordinator
	enc := gob.NewEncoder(conn)
	err = enc.Encode(returnMessage)
	ifErrFatal(err, "encoding")
	wg_done.Done()
}

func coordinator(
	chanToCoordinator chan InitialMessageToCoordinator,
	chanToNodes []chan ResponseToNodes,
	wg *sync.WaitGroup,
	flagArgs *FlagArgs,
	finalBlockChan chan FinalBlock,
	allCommittee map[[32]byte][]*nodeMsg,
	files []*os.File,
	trustChan chan committeeTrust ,
	isSendTx chan bool,
	nodeT map[[32]byte]sendToNodeTrust) {

	// wait untill all node connections have pushed an ID/IP to chan
	wg.Wait()
	close(chanToCoordinator)

	// create array of structs that has all info about a node and assign it id/ip
	nodeInfos := make([]NodeAllInfo, flagArgs.n)
	i := 0
	for elem := range chanToCoordinator {
		nodeInfos[i].Pub = elem.pub
		nodeInfos[i].IP = elem.ip
		i += 1
	}
	fmt.Println("9:44")

	// shuffle the list
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(nodeInfos), func(i, j int) { nodeInfos[i], nodeInfos[j] = nodeInfos[j], nodeInfos[i] })

	// Create committees with id
	committees := make([][32]byte, flagArgs.m)
	for i := uint(0); i < flagArgs.m; i++ {
		committees[i] = hash(getBytes(rand.Intn(maxId)))
	}

	fmt.Println("Committees: ", committees)

	var committee  map[[32]byte]*Committee
	committee = make(map[[32]byte]*Committee,flagArgs.m)
	for i :=0 ;i<int(flagArgs.m) ;i++ {
		committee[committees[i]] = new(Committee)
		committee[committees[i]].init(committees[i])
		committee[committees[i]].ID =  committees[i]
	}
	npm,rest := shard(nodeInfos,committees,int(flagArgs.n),int(flagArgs.m))

	// double check amount of nodes in each committee and their adversaries
	lastCommittee := committees[0]
	committeeInfos := make([]committeeInfo, flagArgs.m)
	iCommittee := 0
	for i := int(0); i < int(flagArgs.n); i++ {
		if nodeInfos[i].CommitteeID != lastCommittee {
			lastCommittee = nodeInfos[i].CommitteeID
			iCommittee += 1
		}
		committeeInfos[iCommittee].id = nodeInfos[i].CommitteeID
		committeeInfos[iCommittee].npm += 1
		if !nodeInfos[i].IsHonest {
			committeeInfos[iCommittee].f += 1
		}
	}

	// check that invariants are held
	checkTotalF := 0
	for i := 0; i < len(committeeInfos); i++ {
		if committeeInfos[i].npm != uint(npm) && committeeInfos[i].npm != uint(npm+rest) {
			log.Fatal("Number of nodes in committee not right", npm, npm+rest, committeeInfos[i].npm)
		}

		if committeeInfos[i].f >= int(math.Ceil(float64(committeeInfos[i].npm)/float64(flagArgs.committeeF))) {
			log.Fatalf("Comitte %d has too many adversaries %d", committeeInfos[i].id, committeeInfos[i].f)
		}

		checkTotalF += committeeInfos[i].f
	}

	if flagArgs.n/flagArgs.m != 1 && int(flagArgs.n)/checkTotalF < 1/int(flagArgs.totalF) {
		log.Fatalf("There was too many adversaries in total %d", checkTotalF)
	}

	fmt.Println("Total adversary percentage: ", float64(checkTotalF)/float64(flagArgs.n))

	// gen set of idenetites
	users := genUsers(flagArgs)
	genesisBlocks := genGenesisBlock(flagArgs, committeeInfos, users)

	// create reconfiguration block
	rBlock := new(ReconfigurationBlock)
	rBlock.init()
	count := 0
	for _, committeeinfo := range committeeInfos {
		count = 0
		for _, node := range nodeInfos {
			if node.CommitteeID == committee[committeeinfo.id].ID  {
				nodeM := &nodeMsg{node.Pub,node.IP}
				allCommittee[committeeinfo.id] = append(allCommittee[committeeinfo.id], nodeM )
				tmp := new(CommitteeMember)
				tmp.Pub = node.Pub
				tmp.IP = node.IP
				tmp.IsHonest= node.IsHonest

				committee[committeeinfo.id].addMember(tmp)
				// set BLS key

				count++
			}
		}

	}


	for _, committeeInfo := range committeeInfos {
		newCom := new(Committee)
		newCom.init(committeeInfo.id)
		for _, node := range nodeInfos {
			if node.CommitteeID == newCom.ID {

				tmp := new(CommitteeMember)
				tmp.Pub = node.Pub
				tmp.IP = node.IP
				newCom.addMember(tmp)
			}
		}
		rBlock.Committees[newCom.ID] = newCom
	}
	// create initial randomness
	rnd := make([]byte, 32)
	rand.Read(rnd)
	rBlock.Randomness = hash(rnd)
	rBlock.setHash()

	msg := ResponseToNodes{nodeInfos, genesisBlocks, nodeInfos[0].Pub.Bytes, rBlock}

	for _, c := range chanToNodes {
		c <- msg
	}
	go competleSumTrust(trustChan,committee)
	txGenerator(flagArgs, nodeInfos, users, genesisBlocks, finalBlockChan, files, isSendTx,committee, nodeT)
}
func competleSumTrust(committeeT chan committeeTrust,committee map[[32]byte]*Committee){
	for  {
		before := time.Now()
		l := len(committeeT)
		for i:=0 ;i<l ;i++ {
			trust := <- committeeT
			committee[trust.committeeID].mux.Lock()
			committee[trust.committeeID].trust = trust.sumTrust
			committee[trust.committeeID].mux.Unlock()
		}
		after := time.Now()
		dur := (time.Second / time.Duration(len(committee))) - after.Sub(before)
		if dur > 0 {
			time.Sleep(dur)
		}
	}
}
func prepareResultString(s string) string {
	tmp := strconv.FormatInt(time.Now().Unix(), 10)
	tmp += ","
	tmp += s
	tmp += "\n"
	return tmp
}

func writeIntToFile(integer int64, f *os.File) {

	s := prepareResultString(strconv.FormatInt(integer, 10))

	f.WriteString(s)
	f.Sync()
}

func writeStringToFile(s string, f *os.File) {

	newS := prepareResultString(s)

	f.WriteString(newS)
	f.Sync()
}

func coordinatorDebugStatsHandleConnection(conn net.Conn,
	trustlocalGroup *trustLocalGroup,
	committeeSumT chan committeeTrust,
	finalBlockChan chan FinalBlock,
	files []*os.File,
	rMap *routetxmap,
	idaresults *IDAGossipResultsMap,
	allCommittee map[[32]byte][]*nodeMsg,
	nodeT map[[32]byte]sendToNodeTrust,
	) {
	msg := new(Msg)
	reciveMsg(conn, msg)
	switch msg.Typ {
	case "IDASuccess":
		_, ok := msg.Msg.([32]byte)
		if !ok {
			errFatal(ok, "IDASuccess decoding")
		}
		//coordinatorHandleIDASuccess(idaMsg, successfullGossips)

	case "consensus":
		_, ok := msg.Msg.(string)
		notOkErr(ok, "coordinator consensus cMsg decoding")
		//coordinatorHandleConsensus(cMsg, consensusResults)
	case "finalblock":
		log.Println("Recived: ", msg.Typ)
		block, ok := msg.Msg.(FinalBlock)
		notOkErr(ok, "finalblock")
		finalBlockChan <- block
	case "pocverify":
		dur, ok := msg.Msg.(time.Duration)
		notOkErr(ok, "pocverify")
		writeIntToFile(dur.Nanoseconds(), files[1])
	case "pocadd":
		dur, ok := msg.Msg.(time.Duration)
		notOkErr(ok, "pocadd")
		writeIntToFile(dur.Nanoseconds(), files[2])
	case "routetx":
		tx, ok := msg.Msg.(ByteArrayAndTimestamp)
		notOkErr(ok, "routtx")
		ID := toByte32(tx.B)
		rMap.add(ID)
		r := rMap.get(ID)
		r.addStart(tx.T)
	case "find_node":
		tuple, ok := msg.Msg.(ByteArrayAndTimestamp)
		notOkErr(ok, "find_node")
		txid := toByte32(tuple.B[:32])
		committeeID := toByte32(tuple.B[32:])
		rMap.add(txid)
		r := rMap.get(txid)
		r.add(committeeID, tuple.T)
	case "trustVector":
		tv , ok := msg.Msg.(TrustVector)
		notOkErr(ok, "start trust process")
		fmt.Println("rec trust vector")
		processLocalTrust_ra(tv,trustlocalGroup,allCommittee,committeeSumT, nodeT)
	case "transaction_recieved":
		bat, ok := msg.Msg.(ByteArrayAndTimestamp)
		notOkErr(ok, "transaction recived")
		ID := toByte32(bat.B)
		rMap.add(ID)
		r := rMap.get(ID)
		ok = r.addEnd(bat.T)
		if ok {
			// sleep for a delta to let incomming request be processed
			time.Sleep(default_delta * 3 * time.Millisecond)
			var s string
			if r.start.IsZero() {
				s += "0"
			} else {
				s += strconv.FormatInt(r.start.Unix(), 10)
			}
			s += ","
			s += strconv.FormatInt(r.end.Unix(), 10)
			for cID, tStamp := range r.committees {
				s += ","
				s += bytes32ToString(cID)
				s += ","
				s += strconv.FormatInt(tStamp.Unix(), 10)
			}
			writeStringToFile(s, files[3])
		}
	case "start_ida_gossip":
		bat, ok := msg.Msg.(ByteArrayAndTimestamp)
		notOkErr(ok, "start ida gossip")
		ID := toByte32(bat.B)
		idaresults.add(ID)
		ida := idaresults.get(ID)
		ida.start = bat.T
	case "reconstructed_ida_gossip":
		bat, ok := msg.Msg.(ByteArrayAndTimestamp)
		notOkErr(ok, "reconstructed idagossip")
		ID := toByte32(bat.B)
		idaresults.add(ID)
		ida := idaresults.get(ID)

		ok = ida.addReconstructed(bat.T)

		if ok {
			time.Sleep(default_delta * time.Millisecond * 3)
			var s string
			ida.mux.Lock()

			s += strconv.FormatInt(ida.start.Unix(), 10)
			for _, tStamp := range ida.reconstructed {
				s += ","
				s += strconv.FormatInt(tStamp.Unix(), 10)
			}
			ida.mux.Unlock()
			writeStringToFile(s, files[4])
		}
	case "consensus_accept_fail":
		log.Println("Recived: ", msg.Typ)
		bat, ok := msg.Msg.(ByteArrayAndTimestamp)
		notOkErr(ok, "consensus accept fail")
		if len(bat.B) != 88 {
			errFatal(nil, fmt.Sprintf("length of consensus accept fail msg was not 80: %d ", len(bat.B)))
		}
		// 32 32 8 8
		cID := toByte32(bat.B[:32])
		pub := toByte32(bat.B[32:64])
		iter := binary.LittleEndian.Uint64(bat.B[64:72])
		totalVotes := int64(binary.LittleEndian.Uint64(bat.B[72:80]))
		rec := int64(binary.LittleEndian.Uint64(bat.B[80:88]))
		log.Printf("[ConsensusAcceptFail] cID: %s, pub: %s, iter: %d, totalVotes: %d, rec: %d", bytes32ToString(cID), bytes32ToString(pub), iter, totalVotes, rec)
		s := fmt.Sprintf("%s,%s,%d,%d,%d", bytes32ToString(cID), bytes32ToString(pub), iter, totalVotes, rec)
		writeStringToFile(s, files[5])

	default:
		errFatal(nil, "no known message type (coordinator)")
	}
}


func coordinatorHandleConsensus(tag string, consensusResults *consensusResult) {

	tmp := (*consensusResults)
	switch tag {
	case "echo":
		tmp.echos += 1
	case "pending":
		tmp.pending += 1
	case "accept":
		tmp.accepts += 1
	}
	(*consensusResults) = tmp
	// TODO fix this to handle multiple committees

}
