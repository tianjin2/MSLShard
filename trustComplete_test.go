package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"sort"
	"strconv"
	"testing"
	"time"
)


type nodeTss []*nodeTs

func (n nodeTss) Len() int {
	return len(n)
}
func (n nodeTss) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}
func (n nodeTss) Less(i, j int) bool {
	return n[i].trust < n[j].trust

}
func TestTrust(test *testing.T) {
	var flagArgs FlagArgs
	flagArgs.n = 10
	flagArgs.m = 2
	flagArgs.totalF = 1
	flagArgs.committeeF = 1
	var nodeInfos []*NodeAllInfo
	nodeInfos = make([]*NodeAllInfo,10)
	//var failureType string
	n := 10
	cmd :=exec.Command("python","G:\\PY workplace\\pythonProject1\\main.py")
	_, _ = cmd.Output()
	data := ReadCsv("new_data.csv")

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

	rand.Seed(time.Now().UnixNano()) //设置随机数种子，加上这行代码，可以保证每次随机都是随机的
	//使用rand.Shuffle方法打乱数组。第一个参数n表示数组的长度，第二个参数闭包函数表示交换数据。
	rand.Shuffle(len(nodeInfos), func(i, j int) { nodeInfos[i], nodeInfos[j] = nodeInfos[j], nodeInfos[i] })

	// Create committees with id


	fmt.Println("Committees: ", committees)
	// divide idIdPairs into equal m chunks and assign them to the committees
	npm := int(flagArgs.n / flagArgs.m)
	rest := int(flagArgs.n % flagArgs.m)
	c := 0
	k :=0
	//c_div := []int{11,11,12,11,12,11,12,12,11,11}
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

		// amount of adversaries in this committee
		f := (t_npm / (6 / c_div))
		// if the above division created exactly 50% adversaries then we subtract one
		if t_npm%(6/c_div) == 0 {
			f--
		}
		/*if c%3 == 0 {
			c_div = 5
		} else {
			c_div = 5
		}
*/
		// TODO: this is not variable with committeeF



		// if the above division created exactly 50% adversaries then we subtract one
		/*if t_npm%(30/c_div[c]) == 0 {
			f--
		}*/
		//nodeInfos[i].IsHonest = true
		if t_npm == 0 || i%t_npm <f {
			nodeInfos[i].IsHonest = true
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
						self.Pub = nodectx.Pub
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
			self.IsHonest = currentCommittee.Members[self.Pub.Bytes].IsHonest
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
			nodeCtx[com][nodectx.Pub.Bytes].decision.NodePub = self.Pub.Bytes
			nodeCtx[com][nodectx.Pub.Bytes].decision.Decision = make(map[[32]byte]indicators)
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
	for epoch < 1 {

		if r  == 11 {
			fmt.Println("TN",TN,"TP",TP,"FN",FN,"FP",FP,"failure:",float64(FN+FP)/float64(FN+FP+TN+TP))
			WriterCSV1("new_data.csv",dataT)
			c1 := exec.Command("sh","run.sh")
			if err := c1.Run(); err != nil {
				fmt.Println("Error: ", err)
			}
			committee,count,nodeTrusts =  reconfiguration(committees,nodeInfos,n,int(flagArgs.m),nodeCtx)
			r = 1
			epoch ++
			trustVs = make(map[[32]byte]map[[32]byte]TrustVector)
			fmt.Println(len(committee))
			//continue
			for _,num := range count {
				fmt.Println(num)
			}
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
						for _, tx := range txhash {
							it := indicators{}
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
						for _, tx := range txhash {
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
					//领导者提出错误块
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

			//time.Sleep(1000*time.Second)
			trustVs[com] = make(map[[32]byte]TrustVector)

			for _, self := range nodectxs {
				time.Sleep(1 * time.Millisecond)
				dvs := make(map[[32]byte]DecisionVector)
				for nodepub,ctx := range nodectxs {
					dvs[nodepub] = ctx.decision
				}
				trustV := receiveDec(self, dvs)
				trustVs[self.committee.ID][self.self.Pub.Bytes] = trustV
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
		data1 := []string{"trust","nengli"}
		WriterCSV("result.csv",data1)
		WriterCSV("result.csv",[]string{strconv.Itoa(r)})
		l := 1
		if r == 10  {
			dataT = make([][]string,n+1)
			dataT[0] = []string{"x","y","trust","label"}
		}
		for committeeID, nodeT := range nodesT {
			fmt.Println("委员会：", bytesToString(committeeID[:]),"   ",r)
			for node, trust := range nodeT {
				fmt.Println("node", bytesToString(node[:4]), ",可信度,", trust.nodeT.trust, "  ")
				nodeCtx[committeeID][node].i.i++
				nodeCtx[committeeID][node].nodeTrust.Trust = trust.nodeT.trust
				nodeCtx[committeeID][node].nodeTrust.Af = trust.Af
				nodeCtx[committeeID][node].nodeTrust.ETrust = trust.nodeT.A_R
				cc := 0.0
				if nodeCtx[committeeID][node].self.IsHonest {
					cc += 0.8 * 0.5 + (1-nodeCtx[committeeID][node].Initvalue.cpu) * 0.25 +(1-nodeCtx[committeeID][node].Initvalue.qos.latency) * 0.25
				}else{
					cc += 0.2 * 0.5 + (1-nodeCtx[committeeID][node].Initvalue.cpu) * 0.25 +(1-nodeCtx[committeeID][node].Initvalue.qos.latency) * 0.25
				}
				data2 := []string{strconv.FormatFloat(trust.nodeT.trust,'f',3,64),strconv.FormatFloat(cc,'f',3,64)}
				if r  == 10 {
					dataT[l] = []string{nodeCtx[committeeID][node].self.coordinate.x,nodeCtx[committeeID][node].self.coordinate.y,strconv.FormatFloat(nodeCtx[committeeID][node].nodeTrust.Trust,'f',17,64),strconv.Itoa(nodeCtx[committeeID][node].self.clusterID)}
					l++
				}
				WriterCSV("result.csv",data2)
			}
		}

		r++
	time.Sleep(1*time.Millisecond)
	}
}


func leaderelection(nodectxs map[[32]byte]*NodeCtx,nodeTrusts []*nodeTs )   {
	sort.Sort(sort.Reverse(nodeTss(nodeTrusts)))
	rnd := RandInt64(1,int64(len(nodeTrusts)-1))
	leader := nodeTrusts[rnd].node
	for _,nodectx := range nodectxs {
		nodectx.committee.CurrentLeader = leader
	}

}