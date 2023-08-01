package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"
)

// Start a completly new iteration. With leader election and if you are leader, perform leader duties.
func startNewIteration(nodeCtx *NodeCtx) {
	nodeCtx.i.flag[nodeCtx.i.i] = true
	leaderElection(nodeCtx)
	// If this node is leader then initate leader protocol
	if nodeCtx.committee.CurrentLeader.Bytes == nodeCtx.self.Priv.Pub.Bytes {
		// wait untill tx pool is large enough
		before := time.Now()
		var dur float64

		for {
			after := time.Now()
			dur = after.Sub(before).Seconds()
			l := nodeCtx.txPool.len()
			if l >= 10 {
				before = time.Now()
				break
			}else if l>=5 {
				before = time.Now()
				break
			}else if dur > 10 && dur <= 20 {
				if l != 0 {
					before = time.Now()
					break
				}
			}else if dur > 20 {
				before = time.Now()
				fmt.Println(l)
			}

			time.Sleep(200 * time.Millisecond)
		}
		leader(nodeCtx)
	}
}

type byte32sortHelper struct {
	original [32]byte
	toSort   [32]byte
}

// finds current leader of comittee, using epoch randomness and iteration number
func leaderElection(nodeCtx *NodeCtx) {
	// The paper doesnt specifically mention any leader election protocols, so we assume that the leader election protocol
	// used in bootstrap is also used in the normal protocol, with the adition of iteration (unless the same leader would
	// be selected).
	// TODO actually add a setup phase where one must publish their hash. This way there will always
	// be a leader even if some nodes are offline. But with the assumption that every node is online
	// this works fine.

	// get current randomness
	recBlock := nodeCtx.blockchain.getLastReconfigurationBlock()
	rnd := recBlock.Randomness

	// get current iteration
	_currIteration := nodeCtx.i.getI()
	currI := make([]byte, 8)
	binary.LittleEndian.PutUint64(currI, uint64(_currIteration))

	listOfHashes := make([]byte32sortHelper, len(nodeCtx.committee.Members))
	// calculate hash(id | rnd | currI) for every member
	ii := 0
	for _, m := range nodeCtx.committee.Members {
		connoctated := byteSliceAppend(m.Pub.Bytes[:], rnd[:], currI)
		hsh := hash(connoctated)
		listOfHashes[ii] = byte32sortHelper{m.Pub.Bytes, hsh}
		ii++
	}

	// sort list
	listOfHashes = sortListOfByte32SortHelper(listOfHashes)

	// calculate hash of self
	selfHash := hash(byteSliceAppend(nodeCtx.self.Priv.Pub.Bytes[:], rnd[:], currI))

	// the leader is the lowest in list except if selfHash is lower than that.
	if byte32Operations(selfHash, "<", listOfHashes[0].toSort) {
		nodeCtx.committee.CurrentLeader = nodeCtx.self.Priv.Pub
		//log.Println("I am leader!", nodeCtx.amILeader())
	} else {
		leader := listOfHashes[0].original
		nodeCtx.committee.CurrentLeader = nodeCtx.committee.Members[leader].Pub
	}
}

func shouldISendCrossTX(nodeCtx *NodeCtx) bool {
	// log(m) nodes as defined in thesis

	// get current randomness
	recBlock := nodeCtx.blockchain.getLastReconfigurationBlock()
	rnd := recBlock.Randomness

	// get current iteration
	_currIteration := nodeCtx.i.getI()
	currI := make([]byte, 8)
	binary.LittleEndian.PutUint64(currI, uint64(_currIteration))

	listOfHashes := make([]byte32sortHelper, len(nodeCtx.committee.Members))
	// calculate hash(id | rnd | currI) for every member
	ii := 0
	for _, m := range nodeCtx.committee.Members {
		connoctated := byteSliceAppend(m.Pub.Bytes[:], rnd[:], currI)
		hsh := hash(connoctated)
		listOfHashes[ii] = byte32sortHelper{m.Pub.Bytes, hsh}
		ii++
	}

	// sort list
	listOfHashes = sortListOfByte32SortHelper(listOfHashes)

	// calculate hash of self
	selfHash := hash(byteSliceAppend(nodeCtx.self.Priv.Pub.Bytes[:], rnd[:], currI))

	_cutoff := math.Log2(float64(len(nodeCtx.committee.Members)))
	cutoff := int(math.Abs(_cutoff))

	// the leader is the lowest in list except if selfHash is lower than that.
	if byte32Operations(selfHash, "<", listOfHashes[cutoff].toSort) {
		return true
	}
	return false
}

func basicLeaderElectionasdf(nodeCtx *NodeCtx) *PubKey {
	// Find out who is the leader, returns ID of leader
	// For now just pick the one with the lowest ID.
	// TODO: create an actual leader election protocol based on epoch randomness and nonce, assume every node is online
	// TODO: figure out some complete leader election protocol

	// set yourself to the lowest seen
	var lowestID *PubKey = nodeCtx.self.Priv.Pub
	lowestIDBigInt := toBigInt(lowestID.Bytes)

	for k := range nodeCtx.committee.Members {
		kBI := toBigInt(k)
		if kBI.Cmp(lowestIDBigInt) < 0 {
			lowestID = nodeCtx.committee.Members[k].Pub
			lowestIDBigInt = kBI
		}
	}

	if false {
		// go trough all nodes in system and give the lowest id instead of lowest id in committee
		for k := range nodeCtx.allInfo {
			kBI := toBigInt(k)
			if kBI.Cmp(lowestIDBigInt) < 0 {
				lowestID = nodeCtx.allInfo[k].Pub
				lowestIDBigInt = kBI
			}
		}
	}

	fmt.Println("Leader", lowestID == nodeCtx.self.Priv.Pub)

	nodeCtx.committee.CurrentLeader = lowestID

	return lowestID
}

func leader(nodeCtx *NodeCtx) {
	// initates leader process

	// create a block
	block := createProposeBlock(nodeCtx)

	// ida-gossip the block
	IDAGossip(nodeCtx, block.encode(), "block")

	// wait until we have recivied and recreated IDA message
	for !nodeCtx.blockchain.isProposedBlock(block.GossipHash) {
		time.Sleep(100 * time.Millisecond)
	}

	// sleep a delta before iniation consensus
	time.Sleep(time.Duration(2*nodeCtx.flagArgs.delta) * time.Millisecond)

	// create a propose msg to initate consensus
	cMsg := new(ConsensusMsg)
	cMsg.GossipHash = block.GossipHash
	cMsg.Tag = "propose"
	cMsg.Pub = nodeCtx.self.Priv.Pub
	cMsg.sign(nodeCtx.self.Priv)

	msg := Msg{"consensus", cMsg, nodeCtx.self.Priv.Pub}

	// start consensus rounds.
	log.Printf("Leader starting conseuss in committee %s\n", bytes32ToString(nodeCtx.committee.ID))
	sendMsgToCommitteeAndSelf(msg, nodeCtx)
}
