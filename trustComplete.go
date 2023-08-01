package main

import (
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"time"
)

/*
	Consensus Characterization: evaluates the decisions made on access control requests, i.e., agreement, negation and uncertainty. It is used to characterize the consensus behavior of the consensus node to be evaluated in the consensus process, i.e., honesty or malice.
	Network Characteristics: Used to characterize the performance efficiency of the consensus node to be evaluated in the consensus process, e.g., network latency or network bandwidth. A parameter K=block size/transmission speed is set, and k1, k2 are two intermediate values set, when K<k1, i.e., network latency is high or unresponsive; network latency is considered to be average when K∈[k1,k2]; and network latency is considered to be low when K>k2.
	Machine characteristics: the machine characteristics of the node in the consensus process of the consensus node to be evaluated, such as the occupancy rate of the processor of the edge node and the memory occupancy rate, etc.
*/
func localTrustComplete(dv DecisionVector,nodePub [32]byte,nodeCtx *NodeCtx) (float64,float64) {

	r := make([]int,3)
	b := make([]float64,3)
	u := 0.0
	b_yt := make([]float64,3)
	u_yt := 0.0
	b_ytDR := make([]float64,3)
	u_ytDR := 0.0
	r_t := make([]float64,3)
	s_t := make([]float64,3)
	E_L := make([]float64,3)

	for i := 0 ;i <3 ;i++ {
		r[i] = 0
		b[i] = 0.0
		b_yt[i] = 0.0
		b_ytDR[i] = 0.0
		r_t[i] = 0.0
		s_t[i] = 0.0
		E_L[i] = 0.0

	}
	//Statistically observed evidence
	if nodeCtx.committee.Members[nodeCtx.committee.CurrentLeader.Bytes].IsHonest {
		for _,txDecision  := range dv.Decision {
			if txDecision.BlockDecision == "N" || (txDecision.Latency >= 0.7) || txDecision.CpuMachine >= 0.8 {
				r[0]++
			}else if txDecision.BlockDecision == "Y" && ( txDecision.Latency > 0.3  ||  txDecision.CpuMachine > 0.4 ) {
				r[1]++
			}else if txDecision.BlockDecision == "Y" && (txDecision.Latency <= 0.3) && txDecision.CpuMachine <= 0.4 {
				r[2]++
			}else {
				fmt.Println("error")
			}
		}
	}else{
		for _,txDecision  := range dv.Decision {
			if txDecision.BlockDecision == "Y" || (txDecision.Latency >= 0.7) || txDecision.CpuMachine >= 0.8 {
				r[0]++
			}else if txDecision.BlockDecision == "N" && ( txDecision.Latency > 0.3  ||  txDecision.CpuMachine > 0.4 ) {
				r[1]++
			}else if txDecision.BlockDecision == "N" && (txDecision.Latency <= 0.3) && txDecision.CpuMachine <= 0.4 {
				r[2]++
			}else {
				fmt.Println("error")
			}
		}
	}


	C := 2
	sum := 0
	for j := 0 ;j< 3 ;j++ {
		sum += r[j]
	}

	for i := 0 ;i < 3; i++ {
		b[i] = float64(r[i]) /float64(C +sum)

	}
	u = float64(C) /float64(C +sum)
	// jsang subjective evaluation of nodes (perception space) based on decisions about transactions in the block (opinion space)
	af := nodeCtx.nodeTrust.Af
	// Evaluate R_y,t including b, d, u
	nowTime := time.Now()
	//Local trust L computation Involves time decay
	nodeCtx.nodeJs.mux.Lock()
	for i :=0 ;i< 3 ;i++ {
		b_yt[i] += b[i]
		
	}
	u_yt += u
	var l uint
	if nodeCtx.i.i < 10 {
		l =0
	}else{
		l = nodeCtx.i.i - 9
	}
	for ;l < nodeCtx.i.i;l++{
		index := - float64(nowTime.Sub(nodeCtx.nodeJs.Jsang[l][nodePub].Time).Milliseconds())
		for i :=0 ;i< 3 ;i++ {
			b_yt[i] += math.Pow(math.E,index) * nodeCtx.nodeJs.Jsang[l][nodePub].R_b[i]
		}
		u_yt += math.Pow(math.E,index) * nodeCtx.nodeJs.Jsang[l][nodePub].R_u
	}

	nodeCtx.nodeJs.mux.Unlock()
	// rating credibility
	D_R := 0.6*nodeCtx.nodeTrust.Trust + 0.4 * nodeCtx.nodeTrust.ETrust
	L_R := 0.0
	for i := 0 ;i< 3 ;i++{
		b_ytDR[i] = b_yt[i] * D_R
		u_ytDR = u_yt * D_R
		r_t[i] = 2*b_ytDR[i]/u_ytDR
	}

	nodeJsangs := Jsang{nowTime,b_ytDR,u_ytDR}
	nodeCtx.nodeJs.add(nodePub,nodeCtx.i.i,nodeJsangs)

	for i := 0 ;i< 3 ;i++{
		numerator := r_t[i]+float64(C)*af[i]
		denominator := float64(C)
		for j := 0 ;j< 3 ;j++{
			denominator += r_t[j]
		}
		E_L[i] = numerator/denominator
		L_R += (float64(i)/2.0) * E_L[i]
	}

	//Calculate localized expectations for positive evaluations, localized expectations for negative evaluations
	L_S := E_L[0]
	return L_R,L_S
}
func localTrustComplete_ra(dv DecisionVector,nodePub [32]byte,nodeCtx *NodeCtx) (float64,float64) {

	r := make([]int,3)
	b := make([]float64,3)
	u := 0.0
	b_yt := make([]float64,3)
	u_yt := 0.0
	b_ytDR := make([]float64,3)
	u_ytDR := 0.0
	r_t := make([]float64,3)
	s_t := make([]float64,3)
	E_L := make([]float64,3)

	for i := 0 ;i <3 ;i++ {
		r[i] = 0
		b[i] = 0.0
		b_yt[i] = 0.0
		b_ytDR[i] = 0.0
		r_t[i] = 0.0
		s_t[i] = 0.0
		E_L[i] = 0.0

	}
	//Statistically observed evidence
		for _,txDecision  := range dv.Decision {
			if txDecision.BlockDecision == "N" || (txDecision.Latency >= 0.7) || txDecision.CpuMachine >= 0.8 {
				r[0]++
			}else if txDecision.BlockDecision == "Y" && ( txDecision.Latency > 0.3  ||  txDecision.CpuMachine > 0.4 ) {
				r[1]++
			}else if txDecision.BlockDecision == "Y" && (txDecision.Latency <= 0.3) && txDecision.CpuMachine <= 0.4 {
				r[2]++
			}else {
				fmt.Println("!23")
			}
	}


	C := 2
	sum := 0
	for j := 0 ;j< 3 ;j++ {
		sum += r[j]
	}

	for i := 0 ;i < 3; i++ {
		b[i] = float64(r[i]) /float64(C +sum)

	}
	u = float64(C) /float64(C +sum)
	// jsang subjective evaluation of nodes (perception space) based on decisions about transactions in the block (opinion space)
	af := nodeCtx.nodeTrust.Af
	// Evaluate R_y,t including b, d, u

	nowTime := time.Now()
	nodeJsang := Jsang{nowTime,b,u}
	nodeCtx.nodeJs.add(nodePub,nodeCtx.i.i,nodeJsang)
	//Local trust L computation Involves time decay
	nodeCtx.nodeJs.mux.Lock()
	for i :=0 ;i< 3 ;i++ {
		b_yt[i] += b[i]
	}
	u_yt += u
	var l uint
	if nodeCtx.i.i < 10 {
		l =0
	}else{
		l = nodeCtx.i.i - 9
	}
	for ;l < nodeCtx.i.i;l++{

		index := - float64(nowTime.Sub(nodeCtx.nodeJs.Jsang[l][nodePub].Time).Milliseconds())
		for i :=0 ;i< 3 ;i++ {
			b_yt[i] += math.Pow(math.E,index) * nodeCtx.nodeJs.Jsang[l][nodePub].R_b[i]

		}
		u_yt += math.Pow(math.E,index) * nodeCtx.nodeJs.Jsang[l][nodePub].R_u

	}

	nodeCtx.nodeJs.mux.Unlock()
	// rating credibility
	D_R := 0.6*nodeCtx.nodeTrust.Trust + 0.4 * nodeCtx.nodeTrust.ETrust
	L_R := 0.0
	for i := 0 ;i< 3 ;i++{
		b_ytDR[i] = b_yt[i] * D_R
		u_ytDR = u_yt * D_R
		r_t[i] = 2*b_ytDR[i]/u_ytDR
	}

	nodeJsangs := Jsang{nowTime,b_ytDR,u_ytDR}
	nodeCtx.nodeJs.add(nodePub,nodeCtx.i.i,nodeJsangs)

	for i := 0 ;i< 3 ;i++{
		numerator := r_t[i]+float64(C)*af[i]
		denominator := float64(C)
		for j := 0 ;j< 3 ;j++{
			denominator += r_t[j]
		}
		E_L[i] = numerator/denominator
		L_R += (float64(i)/2.0) * E_L[i]
	}

	//Calculate localized expectations for positive evaluations, localized expectations for negative evaluations
	L_S := E_L[0]

	return L_R,L_S
}



func completeGlobalTrust(nodeLocalTrust map[[32]byte]*TrustVector,nodemsg map[[32]byte]*CommitteeMember) (map[[32]byte]sendToNodeTrust,map[[32]byte][]float64) {
	nodeT  := make(map[[32]byte]sendToNodeTrust)
	jsang := Jsang{}
	EA := make([]float64,3)

	Af := make(map[[32]byte][]float64)
	for _,comm := range nodemsg{
		i := 0
		ssim := 0.0
		b:= make([]float64,3)
		u:= 0.0
		for j := 0 ;j <3 ;j++{
			b[j] = 0.0

		}

		Af[comm.Pub.Bytes] = make([]float64,3)
		for _,sendLocaltrust := range nodeLocalTrust{
			com := comm.Pub.Bytes
			ssimi,err :=  CosineSimilarity(nodeLocalTrust[com].Ptrust,sendLocaltrust.Ptrust)
			ssim += ssimi
			if err != nil{
				errFatal(err,"complete ssim failed")
			}
			if i == 0 {
				jsang.R_b = sendLocaltrust.Jsang[com].R_b
				jsang.R_u = sendLocaltrust.Jsang[com].R_u
				i++
			}else{
				b = jsang.R_b
				u = jsang.R_u
				for j :=0 ; j <3 ;j++ {
					if jsang.R_u !=0 || sendLocaltrust.Jsang[com].R_u !=0 {
						jsang.R_b[j] = (b[j]*sendLocaltrust.Jsang[com].R_u + u * sendLocaltrust.Jsang[com].R_b[j]) / (u +sendLocaltrust.Jsang[com].R_u )
						b[j] = jsang.R_b[j]
					}else{
						jsang.R_b[j] = b[j]*0.7 + sendLocaltrust.Jsang[com].R_b[j] * 0.3

						b[j] = jsang.R_b[j]

					}
				}
				if jsang.R_u !=0 || sendLocaltrust.Jsang[com].R_u !=0 {
					jsang.R_u = (2*u*sendLocaltrust.Jsang[com].R_u) / (u +sendLocaltrust.Jsang[com].R_u )
					u = jsang.R_u
				}else{
					jsang.R_u = 0
					u = jsang.R_u
				}
				i++
			}

		}
		//Global weighting of expectations
		W_RA := ssim / float64(len(nodemsg))

		r_t := make([]float64,3)
		af := nodeLocalTrust[comm.Pub.Bytes].Af
		sum := 0.0
		for j := 0; j<3 ; j++ {

			r_t[j] = 2*jsang.R_b[j]/jsang.R_u
			sum += r_t[j]
		}

		A_R := 0.0
		for j := 0 ;j <3 ;j++  {
			//Global Evaluation Expectations
			EA[j] = (r_t[j]+float64(2)*af[j])/(sum+float64(2))
		    Af[comm.Pub.Bytes][j] = EA[j]
			//Global Trust Calculation
			A_R += float64(j)/float64(2)*EA[j]
		}
		A_S := EA[0]
		//Reputation value calculation
		Re :=  W_RA * A_R
		//Calculation of value at risk
		Ri :=(1-  W_RA) * A_S + (1-af[0]) * jsang.R_u

		//Computing node trustworthiness
		var T float64
		if Ri >= 0.8 && Ri <= 1  {
			T =  0.7 * Re - 0.3 * Ri
		}else if Ri < 0.8 && Ri >=0 {
			T =  0.7 * Re + 0.3 * (0.8 -Ri)
		}else {
			T = 0
		}
		nodeT[comm.Pub.Bytes]= sendToNodeTrust{T,A_R}

	}
	return nodeT,Af
}

func CosineSimilarity(selfTrust map[[32]byte]float64 , nodeTrust map[[32]byte]float64)(ssim float64,err error){
	sumA := 0.0
	s1 := 0.0
	//s2 := math.Pow(nodeTrust[selfPub],2)
	s2 := 0.0
	for node,nodeT := range selfTrust{
		/*if node == nodePub {
			s1 += math.Pow(nodeT,2)
			continue
		}else {*/
		sumA += nodeT * nodeTrust[node]
		s1 += math.Pow(nodeT,2)
		s2 += math.Pow(nodeTrust[node],2)
		//	}
	}
	if s1 == 0 || s2 == 0 {
		return 0.0, errors.New("Vectors should not be null (all zeros)")
	}
	return sumA / (math.Sqrt(s1) * math.Sqrt(s2)), nil
}
func receiveDec_ra(nodeCtx *NodeCtx,dv DecisionVector,nodePub [32]byte) {
	El, _ := localTrustComplete_ra(dv,nodePub,nodeCtx)
	if len(nodeCtx.nodeTrust.nodeEL[nodeCtx.i.i]) == 0 {
		nodeCtx.nodeTrust.nodeEL[nodeCtx.i.i] = make(map[[32]byte]float64)
	}
	nodeCtx.nodeTrust.add(nodeCtx.i.i,nodePub,El)
	if len(nodeCtx.nodeTrust.nodeEL[nodeCtx.i.i]) == len(nodeCtx.committee.Members)+1 {
		trustV := TrustVector{}
		trustV.init()
		trustV.NodePub = nodeCtx.self.Priv.Pub.Bytes
		trustV.Ptrust = nodeCtx.nodeTrust.nodeEL[nodeCtx.i.i]
		//fmt.Println(nodeCtx.nodeTrust.nodeEL[nodeCtx.i.i])
		trustV.CommitteeID = nodeCtx.committee.ID
		trustV.Af = nodeCtx.nodeTrust.Af
		trustV.Round = dv.Round
		trustV.Jsang = nodeCtx.nodeJs.Jsang[trustV.Round]
		h := byteSliceAppend(nodeCtx.self.Priv.Pub.Bytes[:])
		for nodepub,trust := range trustV.Ptrust {
			h = byteSliceAppend(h,nodepub[:], Float64ToByte(trust))
		}
		h = byteSliceAppend(h,trustV.CommitteeID[:])
		var msghash [32]byte
		copy(msghash[:], h)
		trustV.Signature  = nodeCtx.self.Priv.sign(msghash)

		msg := Msg{"trustVector", trustV, nodeCtx.self.Priv.Pub}
		go dialAndSend(coord+":8080", msg)
		nodeCtx.nodeTrust.delect(nodeCtx.i.i)
	}

}
func Float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}
func receiveDec(nodeCtx *NodeCtx,dvs map[[32]byte]DecisionVector) TrustVector  {
	for nodePub , dv := range dvs {
		El, _ := localTrustComplete(dv,nodePub,nodeCtx)

		if len(nodeCtx.nodeTrust.nodeEL[nodeCtx.i.i]) == 0 {
			nodeCtx.nodeTrust.nodeEL[nodeCtx.i.i] = make(map[[32]byte]float64)
		}
		nodeCtx.nodeTrust.add(nodeCtx.i.i,nodePub,El)
	}
	round := dvs[nodeCtx.self.Pub.Bytes].Round
	trustV := TrustVector{}
	trustV.init()
	trustV.NodePub = nodeCtx.self.Pub.Bytes
	trustV.Ptrust = nodeCtx.nodeTrust.nodeEL[nodeCtx.i.i]
	trustV.CommitteeID = nodeCtx.committee.ID
	trustV.Af = nodeCtx.nodeTrust.Af
	trustV.Round = round
	trustV.Jsang = nodeCtx.nodeJs.Jsang[trustV.Round]
	nodeCtx.nodeTrust.delect(nodeCtx.i.i)
	return trustV
}

func processLocalTrust_ra(tv TrustVector,trustVectorGroup *trustLocalGroup,allCommittee map[[32]byte][]*nodeMsg,committeeSumT chan committeeTrust, nodeT map[[32]byte]sendToNodeTrust) {
	round := tv.Round
	committeeID := tv.CommitteeID
	nodePub := tv.NodePub

	var Af map[[32]byte][]float64
	trustVectorGroup.mux.Lock()
	if len(trustVectorGroup.trust[round]) == 0 {
		trustVectorGroup.trust[round] = make(map[[32]byte]map[[32]byte]*TrustVector)
	}
	if len(trustVectorGroup.trust[round][committeeID]) == 0 {
		trustVectorGroup.trust[round][committeeID] = make(map[[32]byte]*TrustVector)
	}
	trustVectorGroup.trust[round][committeeID][nodePub] = &tv
	trustVectorGroup.mux.Unlock()
	if len(trustVectorGroup.trust[round][committeeID]) == len(allCommittee[committeeID]) {
		nodeT,Af = completeGlobalTrust_ra(trustVectorGroup.trust[round][committeeID],allCommittee[committeeID])
		sendCorrdToNode := sendCoordToNodeLocalTrust{}
		sendCorrdToNode.Trust = make(map[[32]byte]sendToNodeTrust)
		sumTrust := 0.0
		for _,node := range allCommittee[committeeID] {
			sendCorrdToNode.NodePub = node.nodePub.Bytes
			sendCorrdToNode.Af = Af[node.nodePub.Bytes]
			sendCorrdToNode.ETrust = nodeT[node.nodePub.Bytes].A_R
			sendCorrdToNode.Trust = nodeT
			msg := Msg{"NodeTrust",sendCorrdToNode,node.nodePub}
			fmt.Println("send node")
			go dialAndSend(node.IP, msg)
			sumTrust += nodeT[node.nodePub.Bytes].trust
		}
		committeeSumT <- committeeTrust{committeeID,sumTrust}
		delete(trustVectorGroup.trust,round)
	}

}

func completeGlobalTrust_ra(nodeLocalTrust map[[32]byte]*TrustVector,nodemsg []*nodeMsg) (map[[32]byte]sendToNodeTrust,map[[32]byte][]float64) {
	nodeT  := make(map[[32]byte]sendToNodeTrust)
	jsang := Jsang{}
	EA := make([]float64,3)
	Af := make(map[[32]byte][]float64)
	for _,comm := range nodemsg{
		i := 0
		ssim := 0.0
		b:= make([]float64,3)
		u:= 0.0
		for j := 0 ;j <3 ;j++{
			b[j] = 0.0

		}
		Af[comm.nodePub.Bytes] = make([]float64,3)
		for nodeID,sendLocaltrust := range nodeLocalTrust{
			com := comm.nodePub.Bytes
			if com == nodeID {
				//continue
			}
			ssimi,err :=  CosineSimilarity(nodeLocalTrust[com].Ptrust,sendLocaltrust.Ptrust)
			ssim += ssimi
			if err != nil{
				errFatal(err,"complete ssim failed")
			}
			if i == 0 {
				jsang.R_b = sendLocaltrust.Jsang[com].R_b
				jsang.R_u = sendLocaltrust.Jsang[com].R_u
				i++
			}else{
				b = jsang.R_b
				u = jsang.R_u
				for j:=0;j<3;j++{
					if jsang.R_u !=0 || sendLocaltrust.Jsang[com].R_u !=0 {
						jsang.R_b[j] = (b[j]*sendLocaltrust.Jsang[com].R_u + u * sendLocaltrust.Jsang[com].R_b[j]) / (u +sendLocaltrust.Jsang[com].R_u )
						jsang.R_u = (2*u*sendLocaltrust.Jsang[com].R_u ) / (u +sendLocaltrust.Jsang[com].R_u )
						b[j] = jsang.R_b[j]
					}else{
						jsang.R_b[j] = b[j]*0.7 + sendLocaltrust.Jsang[com].R_b[j] * 0.3
						b[j] = jsang.R_b[j]
					}
				}
				if jsang.R_u !=0 || sendLocaltrust.Jsang[com].R_u !=0 {
					jsang.R_u = (2*u*sendLocaltrust.Jsang[com].R_u) / (u +sendLocaltrust.Jsang[com].R_u )
					u = jsang.R_u
				}else{
					jsang.R_u = 0
					u = jsang.R_u
				}
				i++
			}
		}
		//Global weighting of expectations
		W_RA := ssim / float64(len(nodemsg))
		r_t := make([]float64,3)
		sum := 0.0
		af := nodeLocalTrust[comm.nodePub.Bytes].Af
		for j := 0; j<3 ; j++ {

			r_t[j] = 2*jsang.R_b[j]/jsang.R_u
			sum += r_t[j]
		}
		A_R := 0.0
		for j := 0 ;j <3 ;j++  {
			//Global Evaluation Expectations
			EA[j] = (r_t[j]+float64(2)*af[j])/(sum+float64(2))
			Af[comm.nodePub.Bytes][j] = EA[j]
			//Global Trust Calculation
			A_R += float64(j)/float64(2)*EA[j]
		}
		A_S := EA[0]
		//Reputation value calculation
		Re :=  W_RA * A_R

		Ri :=(1-  W_RA) * A_S + (1-af[0]) * jsang.R_u
		//Computing node trustworthiness
		var T float64
		if Ri >= 0.8 && Ri <= 1  {
			T =  0.7 * Re - 0.3 * Ri
		}else if Ri <= 0.8 && Ri >=0 {
			T =  0.7 * Re + 0.3 * (0.8 -Ri)
		}else {
			T = 0
		}
		nodeT[comm.nodePub.Bytes]= sendToNodeTrust{T,A_R}
	}
	return nodeT,Af
}
func processLocalTrust(tv TrustVector,trustLocalGroup map[uint]map[[32]byte]map[[32]byte]*TrustVector,CommitteeMember map[[32]byte]*CommitteeMember,nodesT map[[32]byte]map[[32]byte]nodeTrust,committee map[[32]byte][][32]byte) {
	round := tv.Round
	committeeID := tv.CommitteeID
	nodePub := tv.NodePub
	var nodeT map[[32]byte]sendToNodeTrust
	var Af map[[32]byte][]float64
	if len(trustLocalGroup[round]) == 0 {
		trustLocalGroup[round] = make(map[[32]byte]map[[32]byte]*TrustVector)
	}
	if len(trustLocalGroup[round][committeeID]) == 0 {
		trustLocalGroup[round][committeeID] = make(map[[32]byte]*TrustVector)
	}
	trustLocalGroup[round][committeeID][nodePub] = &tv

	//fmt.Println(nodePub," ",len(trustLocalGroup[round][committeeID]))
	if len(trustLocalGroup[round][committeeID]) == len(committee[committeeID]) {
		//fmt.Println(allCommittee[committeeID])
		nodeT, Af = completeGlobalTrust(trustLocalGroup[round][committeeID], CommitteeMember)
		for id,nodetrust  := range nodeT{
			nodesT[committeeID][id] = nodeTrust{nodetrust,Af[id]}
		}
	}
}
func RecommendTrustFusion(nodeJA,nodeJB Jsang,trustA,trustB float64,nodeA,nodeB [32]byte,afA,afB []float64) float64 {
	K := trustA*nodeJB.R_u + trustB*nodeJA.R_u + (2 - trustA - trustB)*nodeJA.R_u*nodeJB.R_u
	nodeJ := Jsang{}
	nodeAf := make([]float64,3)
	nodeJ.R_b = make([]float64,3)
	if K !=0 {
		for i :=0 ;i <3 ;i++ {
			nodeJ.R_b[i] = (trustA*nodeJA.R_b[i]*nodeJB.R_u+trustB*nodeJB.R_b[i]*nodeJA.R_u) / K
			nodeAf[i] = (trustB*afB[i]*nodeJA.R_u +trustA*afA[i]*nodeJB.R_u - (trustA*afA[i] + trustB * afB[i])*nodeJA.R_u*nodeJB.R_u)/(trustB*nodeJA.R_u +trustA*nodeJB.R_u - (trustA + trustB)*nodeJA.R_u*nodeJB.R_u)
		}
		nodeJ.R_u = (nodeJA.R_u*nodeJB.R_u)/K
	}else{
		rA := 0.5
		rB := 0.5
		for i :=0 ;i <3 ;i++ {
			nodeJ.R_b[i] = rA*nodeJA.R_b[i]+ rB * nodeJB.R_b[i]
			nodeAf[i] = rB * afB[i] + rA * afA[i]
		}
		nodeJ.R_u =  0.0
	}
	C := 0.0
	for i := 0;i<3 ;i++ {
		C += float64(i)/float64(3-1)*(nodeJ.R_b[i] + nodeAf[i] * nodeJ.R_u)
	}
	return C
}

func writeTx(node string,s string){
	var filename = node+".txt"
	var f *os.File
	var _ error
	if checkFileIsExist(filename) { // If the file exists
		f, _ = os.OpenFile(filename,  os.O_APPEND|os.O_RDWR, 0666) //打开文件
	} else {
		f, _ = os.Create(filename) //Creating documents
	}
	io.WriteString(f, s+"\n") //write to file (string)
	_ = f.Close()
}
func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}
func WriterCSV(path string,data []string)  {
	//OpenFile reads a file, creates it if it doesn't exist, uses append mode
	nfs, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err!=nil{
		log.Println("File open failed!")
	}

	defer nfs.Close()
	nfs.Seek(0, io.SeekEnd)

	w := csv.NewWriter(nfs)
	//Setting Properties
	w.Comma = ','
	w.UseCRLF = true
	//row := []string{"1", "2", "3", "4", "5,6"}
	err = w.Write(data)
	if err != nil {
		log.Fatalf("can not write, err is %+v", err)
	}
	// Here it must be refreshed in order to write the data to the file.
	w.Flush()
}

func WriterCSV1(path string,data [][]string)  {

	//OpenFile读取文件，不存在时则创建，使用追加模式
	File,err:=os.Create(path)
	if err!=nil{
		log.Println("文件打开失败！")
	}
	defer File.Close()

	//创建写入接口
	WriterCsv:=csv.NewWriter(File)
	/*for _,str := range data {
		err1:=WriterCsv.Write(str)
		if err1!=nil{
			log.Println("WriterCsv写入文件失败")
		}
		WriterCsv.Flush() //刷新，不刷新是无法写入的
	}*/
	WriterCsv.WriteAll(data)
	//写入一条数据，传入数据为切片(追加模式)


	log.Println("数据写入成功...")
}