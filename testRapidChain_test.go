package main

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"os/exec"
	"sort"
	"strconv"
	"testing"
	"time"
)
type hashPosition struct {
	committeeID  [32]byte
	nodeID       *PubKey
	position     float64
}

func TestRapidchain(t *testing.T){
		var flagArgs FlagArgs
		flagArgs.n = 300
		flagArgs.m = 10
		flagArgs.totalF = 1
		flagArgs.committeeF = 1
		var nodeInfos []*NodeAllInfo
		nodeInfos = make([]*NodeAllInfo,300)
		//var failureType string
		n := 300
		cmd :=exec.Command("python","G:\\PY workplace\\pythonProject1\\main.py")
		_, _ = cmd.Output()
		data := ReadCsv("new_data.csv")
		WriterCSV1("456.csv",data)
		committees := make([][32]byte, flagArgs.m)
		coordinates :=  make(map[[32]byte]Coordinate,n)
		for i := uint(0); i < flagArgs.m; i++ {
			committees[i] = hash(getBytes(rand.Intn(maxId)))
		}
		for i := 0; i < n; i++ {
			privKey := new(PrivKey)
			privKey.gen()
			nodeInfos[i] = &NodeAllInfo{}
			nodeInfos[i].Pub = privKey.Pub
			nodeInfos[i].IP = "127.0.0.1:" + string(rune(2000+i))
			nodeInfos[i].IsHonest = true
			nodeInfos[i].coordinate = Coordinate{data[i][0],data[i][1]}

			coordinates[privKey.Pub.Bytes] = Coordinate{data[i][0],data[i][1]}
			cID, _ := strconv.Atoi(data[i][2])
			nodeInfos[i].clusterID = cID
		}

		rand.Seed(time.Now().UnixNano()) // Set the random number seed, add this line of code to ensure that every random is random
	// Use the rand.Shuffle method to break up the array. The first parameter n indicates the length of the array and the second parameter closure function indicates the data to be exchanged.
		rand.Shuffle(len(nodeInfos), func(i, j int) { nodeInfos[i], nodeInfos[j] = nodeInfos[j], nodeInfos[i] })

		// Create committees with id


		fmt.Println("Committees: ", committees)
		// divide idIdPairs into equal m chunks and assign them to the committees
		npm := int(flagArgs.n / flagArgs.m)
		rest := int(flagArgs.n % flagArgs.m)
		c := 0
		k :=0

		for i := int(0); i < int(flagArgs.n); i++ {
			t_npm := npm

			if i != 0 && i%npm == 0 {
				if i != int(flagArgs.n)-rest { // if there is a rest, it will be put into the last committee
					c++
				} else {
					t_npm += rest
				}
			}
			nodeInfos[i].CommitteeID = committees[nodeInfos[i].clusterID]
			var c_div int
			if c%2 == 0 {
				c_div = 3
			} else {
				c_div = 1
			}
			// TODO: this is not variable with committeeF

			f := (t_npm / (6 / c_div))
			// if the above division created exactly 50% adversaries then we subtract one
			if t_npm%(6/c_div) == 0 {
				f--
			}
			//nodeInfos[i].IsHonest = true
			if t_npm == 0 || i%t_npm <f {
				nodeInfos[i].IsHonest = false
				k++
			} else {
				nodeInfos[i].IsHonest = true
			}
		}

		nodeCtx := make(map[[32]byte]map[[32]byte]*NodeCtx)
		committeelist := make(map[[32]byte][]NodeAllInfo)
		committee := make(map[[32]byte][][32]byte)
		for _, com := range committees {
			nodeCtx[com] = make(map[[32]byte]*NodeCtx)

		}
		for i := 0; i < int(flagArgs.n); i++ {
			for _, com := range committees {
				if nodeInfos[i].CommitteeID == com {
					committeelist[com] = append(committeelist[com], *nodeInfos[i])
					committee[com] = append(committee[com], nodeInfos[i].Pub.Bytes)
					break
				}
			}
		}
		allCommittee := make(map[[32]byte]*Committee)
		for _, committeeinfo := range committees  {
			newCom := new(Committee)
			newCom.init(committeeinfo)
			for _, node := range nodeInfos {
				if node.CommitteeID == newCom.ID {
					tmp := new(CommitteeMember)
					tmp.Pub = node.Pub
					tmp.IP = node.IP
					newCom.addMember(tmp)
				}
			}
			allCommittee[newCom.ID] = newCom
		}



		var currentCommittee Committee
		nodemsgs := make(map[[32]byte][]*CommitteeMember)
		count := make(map[[32]byte]int)
		nodeTrusts := make(map[[32]byte][]*nodeTs)

		for com, node := range committeelist {
			for _, nodectx := range node {
				currentCommittee.init(com)
				nodemsgs[com] = make([]*CommitteeMember, int(flagArgs.n/flagArgs.m))
				var self SelfInfo
				for _, v := range nodeInfos {
					if v.CommitteeID == currentCommittee.ID {
						if v.Pub == nodectx.Pub {
							self.clusterID = v.clusterID
							self.coordinate = v.coordinate

						}
						tmp := new(CommitteeMember)
						tmp.IP = v.IP
						tmp.Pub = v.Pub
						tmp.IsHonest = v.IsHonest
						currentCommittee.Members[v.Pub.Bytes] = tmp

					}
				}

				self.CommitteeID = com
				self.IP = nodectx.IP
				self.IsHonest = currentCommittee.Members[self.Priv.Pub.Bytes].IsHonest
				nodeCtx[com][nodectx.Pub.Bytes] = &NodeCtx{}
				nodeCtx[com][nodectx.Pub.Bytes].committee = currentCommittee
				nodeCtx[com][nodectx.Pub.Bytes].i.i = 1
				nodeCtx[com][nodectx.Pub.Bytes].committee.ID = com
				nodeCtx[com][nodectx.Pub.Bytes].self =self
				nodeCtx[com][nodectx.Pub.Bytes].Initvalue = InitValue{}
				nodeCtx[com][nodectx.Pub.Bytes].Initvalue.init(nodeCtx[com][nodectx.Pub.Bytes],coordinates)

				nodeCtx[com][nodectx.Pub.Bytes].nodeTrust = selfTrust{}
				nodeCtx[com][nodectx.Pub.Bytes].nodeTrust.init(nodeCtx[com][nodectx.Pub.Bytes],true)
				nodeCtx[com][nodectx.Pub.Bytes].nodeJs = nodeJsang{}
				nodeCtx[com][nodectx.Pub.Bytes].nodeJs.init(nodeCtx[com][nodectx.Pub.Bytes])
				nodeCtx[com][nodectx.Pub.Bytes].decision.Round = 0
				nodeCtx[com][nodectx.Pub.Bytes].decision.NodePub = self.Priv.Pub.Bytes
				//nodeCtx[com][nodectx.Pub.Bytes].decision.Decision = make(map[[32]byte]indicators)
				nodeCtx[com][nodectx.Pub.Bytes].failureType = "SA"
				nodeTrusts[com] = append(nodeTrusts[com], &nodeTs{nodectx.Pub,nodeCtx[com][nodectx.Pub.Bytes].nodeTrust.Trust})
				if !self.IsHonest {
					count[currentCommittee.ID]++
				}
			}
		}



		var nodeDv map[[32]byte]DecisionVector
		nodeDv = make(map[[32]byte]DecisionVector)
		trustVs := make(map[[32]byte]map[[32]byte]TrustVector)
		r := 1
		dataT := make([][]string,n+1)
		dataT[0] = []string{"x","y","label","trust"}
		epoch := 0
		TN := 0
		TP := 0
		FN := 0
		FP := 0
		for epoch < 11 {
			if r  == 2 {

				fmt.Println("TN",TN,"TP",TP,"FN",FN,"FP",FP,"failure:",float64(FN+FP)/float64(FN+FP+TN+TP))
				committee,count,nodeTrusts =  reconfigurationRapid(committees,allCommittee,nodeInfos,uint(n),uint(flagArgs.m),nodeCtx)
				r = 1
				epoch ++
				trustVs = make(map[[32]byte]map[[32]byte]TrustVector)
				continue
			}

			for com, nodectxs := range nodeCtx {
				var txhash [][32]byte
				for i := 0; i < 1; i++ {
					tmpp := make([]byte, 32)
					rand.Read(tmpp)
					tmp := hash(tmpp)
					txhash = append(txhash, tmp)
				}
				leaderelection(nodectxs,nodeTrusts[com])
				for pub,nodectx := range nodectxs {
					leader := nodectx.committee.CurrentLeader.Bytes
					if nodectx.amILeader() {
						if !nodectx.self.IsHonest {
							if count[com] < len(nodectx.committee.Members)/2{
								TP++
							}else if  count[com] >= len(nodectx.committee.Members)/2 {
								FP++
							}
						}else {
							if count[com] <= len(nodectx.committee.Members)/2 {
								TN++
							}else if  count[com] > len(nodectx.committee.Members)/2 {
								FN++
							}
						}
					}

					dv := DecisionVector{}
					dv.init()
					if nodectx.committee.Members[leader].IsHonest {
						if nodectx.failureType == "SA" || nodectx.failureType == "CRA" {
							for _,tx := range txhash {
								rnd := RandInt64(1,100)
								it := indicators{}
								if rnd>=0 && rnd <=80 {
									it.BlockDecision = "Y"
								}else{
									it.BlockDecision = "N"
								}
								if nodectx.self.IsHonest {
									it.BlockDecision = "Y"
								}else {
									it.BlockDecision = "N"
								}
								it.Latency = nodectx.Initvalue.qos.latency
								it.CpuMachine = nodectx.Initvalue.cpu
								dv.Decision[tx] = it
								dv.NodePub = pub
								dv.Round = uint(r)
								nodectx.decision.Decision[tx] = it
							}
						}else if nodectx.failureType == "CBA" {
							for _,tx := range txhash {
								it := indicators{}
								it.BlockDecision = "Y"
								it.Latency = nodectx.Initvalue.qos.latency
								it.CpuMachine = nodectx.Initvalue.cpu
								dv.Decision[tx] = it
								dv.NodePub = pub
								dv.Round = uint(r)
								nodectx.decision.Decision[tx] = it
							}
						}
					}else{
						//Leaders presenting error blocks
						for _, tx := range txhash {
							it := indicators{}
							if !nodectx.self.IsHonest {
								it.BlockDecision = "Y"
							} else {
								it.BlockDecision = "N"
							}
							it.Latency = nodectx.Initvalue.qos.latency
							it.CpuMachine = nodectx.Initvalue.cpu
							dv.Decision[tx] = it
							dv.NodePub = pub
							dv.Round = uint(r)
							nodectx.decision.Decision[tx] = it
						}
					}

					nodeDv[pub] = dv
					nodectx.decision.Round = uint(r)
					nodectx.decision.NodePub = pub

				}

				trustVs[com] = make(map[[32]byte]TrustVector)

				for _, self := range nodectxs {
					dvs := make(map[[32]byte]DecisionVector)
					for nodepub,ctx := range nodectxs {
						dvs[nodepub] = ctx.decision
					}
					trustV := receiveDec(self, dvs)
					trustVs[self.committee.ID][self.self.Priv.Pub.Bytes] = trustV
				}

			}

			nodesT := make(map[[32]byte]map[[32]byte]nodeTrust)
			for id := range committee {
				nodesT[id] = make(map[[32]byte]nodeTrust)
			}
			trustLocalGroup := make(map[uint]map[[32]byte]map[[32]byte]*TrustVector)
			for comID, trusts := range trustVs {
				for nodeID, trust := range trusts {
					processLocalTrust(trust, trustLocalGroup, nodeCtx[comID][nodeID].committee.Members, nodesT,committee)
				}
			}
			sum := 0.0
			for committeeID, nodeT := range nodesT {
				for node, trust := range nodeT {
					nodeCtx[committeeID][node].i.i++
					nodeCtx[committeeID][node].nodeTrust.Trust = trust.nodeT.trust
					nodeCtx[committeeID][node].nodeTrust.Af = trust.Af
					nodeCtx[committeeID][node].nodeTrust.ETrust = trust.nodeT.A_R
					sum += trust.nodeT.trust
				}
				allCommittee[committeeID].trust  = sum
			}
			r++
		}

}
func reconfigurationRapid(committees [][32]byte,allCommittee map[[32]byte]*Committee,nodeInfos []*NodeAllInfo,n,m uint,nodeCtx map[[32]byte]map[[32]byte]*NodeCtx) (map[[32]byte][][32]byte,map[[32]byte]int,map[[32]byte][]*nodeTs){
	nids := cuckoo(m,n,allCommittee)
	for _,com := range allCommittee {
		for nid := range com.Members {
			if nid == nids {
				delete(com.Members,nid)
				break
			}
		}
	}
	for _,com := range allCommittee {
		for id := range com.Members {
			if _,ok := nodeCtx[com.ID];!ok {
				nodeCtx[com.ID] = make(map[[32]byte]*NodeCtx)
			}
			if _,ok := nodeCtx[com.ID][id];!ok {
				flag := false
				for _,ncx := range nodeCtx {
					for nid,node := range ncx {
						if nid == id {
							for _,nodeinfo := range nodeInfos {
								if nodeinfo.Pub.Bytes == nid {
									nodeinfo.CommitteeID = com.ID
									break
								}
							}
							node.committee.ID = com.ID
							node.self.CommitteeID = com.ID
							node.committee.Members = allCommittee[com.ID].Members
							nodectx := &NodeCtx{}
							nodectx = node
							//delete(node.committee.Members,nid)
							nodeCtx[com.ID][id] = nodectx
							delete(ncx,nid)
							flag = true
							break
						}
					}
					if flag {
						break
					}
				}
			}else{
				nodeCtx[com.ID][id].committee.Members = allCommittee[com.ID].Members
			}
		}
	}

	for cid,nodectx := range nodeCtx {
		if len(nodectx) == 0  {
			delete(nodeCtx,cid)
		}
	}
	for cid,nodectxs := range nodeCtx {
		for nid := range nodectxs {

			nodeCtx[cid][nid].nodeTrust.init(nodeCtx[cid][nid],false)
			nodeCtx[cid][nid].nodeJs = nodeJsang{}
			nodeCtx[cid][nid].nodeJs.init(nodeCtx[cid][nid])
			nodeCtx[cid][nid].decision.Round = 0
			nodeCtx[cid][nid].decision.NodePub = nid
		//	nodeCtx[cid][nid].decision.Decision = make(map[[32]byte]indicators)
			nodeCtx[cid][nid].i.i = 1
		}
	}
	count := make(map[[32]byte]int)
	committee := make(map[[32]byte][][32]byte)
	for i := 0; i < int(n); i++ {
		for _, com := range committees {
			if nodeInfos[i].CommitteeID == com {
				committee[com] = append(committee[com], nodeInfos[i].Pub.Bytes)
				if !nodeInfos[i].IsHonest {
					count[com]++
				}
				break
			}
		}
	}
	nodeTrusts := make(map[[32]byte][]*nodeTs)
	for com,nodectxs := range nodeCtx {
		for _,node := range nodectxs {
			nodeTrusts[com] = append(nodeTrusts[com], &nodeTs{node.self.Priv.Pub,node.nodeTrust.Trust})
		}
	}
	return committee,count,nodeTrusts
}
func cuckoo(m,n uint,allCommittee map[[32]byte]*Committee) [32]byte  {
	pri := new(PrivKey)
	pri.gen()
	newNode := &CommitteeMember{pri.Pub,"123",true}
	k := float64(m)/float64(n)
	committeePosition := make(map[float64]hashPosition)
	var committees [][32]byte
	var committeeT []committeeTrust
	newMember := make(map[[32]byte]int)
	for committee,committeeMember := range allCommittee {

		committeeT= append(committeeT, committeeTrust{committee,committeeMember.trust})
	}
	sort.Slice(committeeT,func(i, j int) bool { return committeeT[i].sumTrust-committeeT[j].sumTrust < 0 })
	l := int(math.Ceil(float64(len(committeeT))/float64(2)))
	//hash the node ID to between [0,1)
	for i:=0 ;i< l ;i++ {
		committee := committeeT[i].committeeID
		committees = append(committees, committee)
		for _,mem := range allCommittee[committeeT[i].committeeID].Members {
			hashPosit := HashRatio(mem.Pub.Bytes[:])
			committeePosition[hashPosit] = hashPosition{committee,mem.Pub,hashPosit}
		}
	}
	newNodeHash := HashRatio(newNode.Pub.Bytes[:])
	mk :=int(math.Ceil(newNodeHash * float64(l)))
	fmt.Println(mk)
	var newCommitteeID [32]byte
	newCommitteeID = committees[mk-1]
	newMember[newCommitteeID]++
	allCommittee[newCommitteeID].Members[newNode.Pub.Bytes] = newNode
	committeePosition[newNodeHash] = hashPosition{newCommitteeID,newNode.Pub,newNodeHash}
	intervalLeft := newNodeHash - k
	intervalRight := newNodeHash + k
	for hashp,position := range committeePosition{
		if hashp >= intervalLeft && hashp <= intervalRight {
			mkk := rand.Intn(int(m)-l)+l
			newCommitteeID = committeeT[mkk].committeeID
			if len(allCommittee[committeePosition[hashp].committeeID].Members) <= 4{
				continue
			}
			newMember[newCommitteeID]++

			allCommittee[newCommitteeID].Members[position.nodeID.Bytes] = allCommittee[position.committeeID].Members[position.nodeID.Bytes]
			// Delete the original element
			delete(allCommittee[position.committeeID].Members,position.nodeID.Bytes)
		}
	}
	return pri.Pub.Bytes
}
func HashRatio(nodeID []byte) float64 {
	t := &big.Int{}
	t.SetBytes(nodeID[:])
	precision := uint(8 * (len(nodeID) + 1))
	max, b, err := big.ParseFloat("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0, precision, big.ToNearestEven)
	if b != 16 || err != nil {
		log.Fatal("failed to parse big float constant for sortition")
	}
	//hash value as int expression.
	//hval, _ := h.Float64() to get the value
	h := big.Float{}
	h.SetPrec(precision)
	h.SetInt(t)
	ratio := big.Float{}
	cratio, _ := ratio.Quo(&h, max).Float64()
	return cratio
}