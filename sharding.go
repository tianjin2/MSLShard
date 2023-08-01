package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

func shard(nodeInfos []NodeAllInfo,committee [][32]byte,n int,m int ) (int,int) {

	fmt.Println("shardingggggg")
	c1 := exec.Command("sh","run.sh")
	if err := c1.Run(); err != nil {
		fmt.Println("Error: ", err)
	}
	data := ReadCsv("new_data.csv")
	npm := n / m
	rest := n % m
	c := 0
	for i := 0; i < n; i++{
		nodeInfos[i].coordinate = Coordinate{data[i][0],data[i][1]}
		nodeInfos[i].nodetrust,_ = strconv.ParseFloat(data[i][2],4)
		cID, _ := strconv.Atoi(data[i][3])
		nodeInfos[i].clusterID = cID
		fmt.Println(cID)
		nodeInfos[i].CommitteeID = committee[cID]
		t_npm := npm
		if i != 0 && i%npm == 0 {
			if i != n-rest { // if there is a rest, it will be put into the last committee
				c++
			} else {
				t_npm += rest
			}
		}
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
		if t_npm == 0 || i%t_npm < f {
			nodeInfos[i].IsHonest = false
		} else {
			nodeInfos[i].IsHonest = true
		}
	}
	return npm,rest
}

func ReadCsv(filepath string) [][]string {
	// open the file (read-only mode), create an instance of the io.read interface
	opencast,err:=os.Open(filepath)
	if err!=nil{
		log.Println("csv文件打开失败！")
	}
	defer opencast.Close()

	// Create a csv reading interface instance
	ReadCsv:=csv.NewReader(opencast)

	// Get a line of content, usually the first line of content
	read,_:=ReadCsv.Read()
	log.Println(read)

	//Read all the contents
	ReadAll,err:=ReadCsv.ReadAll()
	//log.Println(ReadAll)
	return  ReadAll
	/*
	  Description:
	   1, read csv file returns the content of the slice type, you can traverse the way to use or Slicer[0] way to get the specific value.
	   2, the same function or thread, two calls Read () method, the second call to get the value of every two lines of data, and so on.
	   3, the use of large files when reading line by line, small files directly read all and then traversed, the two application scenarios are not the same, you need to pay attention to.
	*/


}
func reconfiguration(committees [][32]byte,nodeInfos []*NodeAllInfo,n,m int,nodeCtx map[[32]byte]map[[32]byte]*NodeCtx) (map[[32]byte][][32]byte,map[[32]byte]int,map[[32]byte][]*nodeTs){
	data := ReadCsv("new_data.csv")
	committeeMem := make(map[[32]byte]map[[32]byte]*CommitteeMember)
	for _,cid := range committees {
		committeeMem[cid] = make(map[[32]byte]*CommitteeMember)
	}
	for _,coor  := range data {
		for _,nodeInfo := range nodeInfos {
			if nodeInfo.coordinate.x[:13] == coor[0][:13] && nodeInfo.coordinate.y[:13] == coor[1][:13] {
				cid ,_ := strconv.Atoi(coor[2])
				nodeInfo.clusterID = cid
				nodeInfo.CommitteeID = committees[cid]
				committeeMem[nodeInfo.CommitteeID][nodeInfo.Pub.Bytes] = &CommitteeMember{nodeInfo.Pub,nodeInfo.IP,nodeInfo.IsHonest}
				break
			}
		}
	}
	for _,nodeInfo := range nodeInfos {
		if _,ok := nodeCtx[nodeInfo.CommitteeID];!ok {
			nodeCtx[nodeInfo.CommitteeID] = make(map[[32]byte]*NodeCtx)
		}
		if _,ok := nodeCtx[nodeInfo.CommitteeID][nodeInfo.Pub.Bytes];!ok {
			flag := false
			for _,ncx := range nodeCtx {
				for nid,node := range ncx {
					if nid == nodeInfo.Pub.Bytes {
						node.committee.ID = nodeInfo.CommitteeID
						node.self.CommitteeID = nodeInfo.CommitteeID
						node.self.clusterID = nodeInfo.clusterID
						node.committee.Members = committeeMem[nodeInfo.CommitteeID]
						nodectx := &NodeCtx{}
						nodectx = node
						//delete(node.committee.Members,nid)
						nodeCtx[nodeInfo.CommitteeID][nodeInfo.Pub.Bytes] = nodectx
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
			nodeCtx[nodeInfo.CommitteeID][nodeInfo.Pub.Bytes].committee.Members = committeeMem[nodeInfo.CommitteeID]
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
			nodeCtx[cid][nid].decision.Decision = make(map[[32]byte]indicators)
			nodeCtx[cid][nid].i.i = 1
		}
	}
	count := make(map[[32]byte]int)
	committee := make(map[[32]byte][][32]byte)
	for i := 0; i < n; i++ {
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
			nodeTrusts[com] = append(nodeTrusts[com], &nodeTs{node.self.Pub,node.nodeTrust.Trust})
		}
	}
	return committee,count,nodeTrusts
}

func reconfiguration_ra(committees map[[32]byte]*Committee,nodeInfos *[]NodeAllInfo,n,m int,nodeT map[[32]byte]sendToNodeTrust) {
	dataT := make([][]string,n+1)
	dataT[0] = []string{"x","y","trust","label"}
	l := 1
	for _,node := range *nodeInfos {
		dataT[l] = []string{node.coordinate.x,node.coordinate.y,strconv.FormatFloat(nodeT[node.Pub.Bytes].trust,'f',3,64)}
		l++
	}
	c1 := exec.Command("sh","run.sh")
	if err := c1.Run(); err != nil {
		fmt.Println("Error: ", err)
	}
	data := ReadCsv("new_data.csv")
	committee := make([][32]byte, m)
	i := 0
	for cid,_ := range committees{
		committee[i] = cid
		i++
	}
	for _,coor  := range data {
		for _,nodeInfo := range *nodeInfos {
			if nodeInfo.coordinate.x[:13] == coor[0][:13] && nodeInfo.coordinate.y[:13] == coor[1][:13] {
				cid ,_ := strconv.Atoi(coor[2])

				if nodeInfo.CommitteeID != committee[cid] {
					delete(committees[nodeInfo.CommitteeID].Members, nodeInfo.Pub.Bytes)
					nodeInfo.clusterID = cid
					nodeInfo.CommitteeID = committee[cid]
					committees[nodeInfo.CommitteeID].Members[nodeInfo.Pub.Bytes] = &CommitteeMember{nodeInfo.Pub,nodeInfo.IP,nodeInfo.IsHonest}
				}
				break
			}
		}
	}
	for cid,comm := range  committees{
		if len(comm.Members) == 0 {
			delete(committees,cid)
		}
	}

}
