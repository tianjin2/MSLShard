package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jinzhu/copier"
	"github.com/kimborgen/go-merkletree"
)

/*
Train of thoughts (delete later):

Discussion:
	Value is not know to C_out using standard bitcoin UTXO model

	Cross-tx must indicate that it is not supposed to be spendable in C_in after inclusion in C_in's blockchain
	The entire output should be owned and only spendable in C_out (aka OrigTxId committee)

	After cross-tx is included in the blockchain in the C_in committees (aka it has undergone consensus) it is included in a block that has votes
	C_out must be able to verify that the transactions sent back from C_in are valid. Otherwise, any leader (and member?) could just forge the transactions and potentially doublespend.
	The only way for C_out to verify the cross-tx is therefor to have mf+1 valid signatures on the block. To known that these signatures are valid. C_out must known the public keys of
	the members of C_in. The reconfiguration block solves this.

	The set of signatures using normal ECDSA is quite huge, but it have to be done (increasing the argument for BLS aggregate signtures), but the set only has to be
	sent with the batched transactions to that committee.

	The signatures only sign the GossipHash of the proposed block. So how would C_out know that the cross-tx belongs to the block
		Merkle proof of transactions?

	Problem statement conclusion: In any cross-tx protocol. The output committee MUST be able to verify that cross-tx was execectued in C_in.
	In other words, C_out must have Proof of Consensus in C_in.

	Reconfiguration block has CommittteID <-> PubKey pairs therefor the signature set is verifiable to anyone who has the reconfiguration block

	What could proof of consensus lock like?
		1. Simplest
			Data
				Entire block
				Signature set that signed the block
			C_out verify process
				Ensure that signatures belong to that committee using reconfiguration block
				Check that we have mf+1 valid signatures
				C_out now knows the validity of cross-tx since entire block is transmitted and therefor any cross-tx is easily found in that block.
		2. harder
			Data
				Block without transactions // transactions are verifiable trough merkle-root so this is ok
				Merkle proof of cross-tx
				Signature set that signed the block
			C_out verify process
				Same as 1. but where merkle-proof is also verified.
		3. hardest
			Data
				GossipHash [gh]
				MerkleRoot [mr]
				MerkleProof of crossTx [mp]
				Hash of everything else [ha]
				Signature set
			Note:
				This requires a hashing function where hash(ha, mr) = gh
				If we change the hasing process this is possible
			C_out verify process
				Same as 2. but where hash(ha,mr) = gh is enough to verify block. and signatures ofc

	Question: How would C_out include the proof for crossTx so that any member can recreate the blockchain from scratch.
		Include all data of 3. in the block? This will significantly expand size.


	If some cross-tx fails, how would we release the cross-tx outputs to owner?
		Maybe they should always be spendable?
			So a user could potentially spend cross_tx1 before cross_tx2 finnished. orig_tx would then be invalid.
		Timeout?
			Spendable after iteration i+10 for example or something.

	Since C_1 does not know the value or the public key/owner of the Input it cannot create an ouput
	Therefor, a cross-tx must have no Outputs.
	This will be the definer for a cross-tx
	The cross-tx is only spendable in C_1 because C_1 is the closest committee to TxID
	And the new cross-tx will not have a different TxID.
		This is still verifyied by the owner of the transaction since the inputs TxID and the transactions TxID is signed by the owner
		Therefor output committee cannot change the outputs or the original Tx because it is recorded in TxID
	You could have a cross-tx imidatly have an output to several or one of the actual outputs
		But this will change the original TxID, and therefor the user has not signed that new txid

	There is one or more(!) inputs in each cross-tx, but no outputs

	NOTE: If OrigTxId != nil, this is where the transaction belongs and not using TxId

	problem: The rest of the committee needs to know if the transaction has been split into cross-tx
	and sent, so next leaders don't do the same
		Possible solutions:
			- Add transaction to block, but add a bool to say this transaction is not valid
			- Add to a second "pending transactions" list that is added to the block
			- Add to a second "pending TxIDs" list that is added to the block
			- Set TxID to nil to indicate that it is not spendable.

	What happens if block is not accepted?
		hmm

	Cross-TXes should be sent by all members of committtee. Not if tx is invalid, but even if block is not accepted.
	This is due to the fact that if one member sends the cross-tx it will be recived by the target committee anyways.
		Cannot a random member that recives the original-tx just send cross-tx to the committee? yes

	So cross-tx and original tx doesnt really need to be included into a block. But it is an effective way to ensure
	That all members know that cross-tx and original-tx has been sent.

	Should not remove valid UTXOs from set in the cross-tx set. They should be validated. But they are always spendable.
	If an user spends those UTXOs before cross-tx can finnish then the final cross-tx will fail.

Example

Orig transacrtions from user_1
	TxID 		a2c			// C_1
	OrigTxId	nil
	Inputs
		0	// value 25
			TxId 			ef2		// C_3
			N				2
			Sig				of ef2 and a2c
		1	// value 20
			TxId			zb4		// C_2
			N 				1
			Sig				of zb4 and a2c
	Outputs
		0
			Value 	30
			N		0
			PubKey	user_2
		1
			Value 15
			N 		1
			PubKey  user_1 // rest back to self
	ProofOfConsensus nil

Leader switches TxID and OrigTxID so TxID is nil and therefor not spendable

cross-tx 2 ommitted from example

Cross-tx 1 to C_3
	TxId 		nil
	OrigTxId	a2c		// C_1
	Inputs
		0	// value 25 this is know to C_3
			TxId 			ef2		// C_3
			N				2
			Sig				of ef2 and a2c
	Outputs			nil
	ProofOfConsensus nil

Response to C_1 from C_3
	Cross-tx1-r
		TxId		skr  // UNSIGNED!
		OrigTxId 	a2c	 // C_1
		Inputs
			0
				TxId 			ef2		// C_3
				N				2
				Sig				Of ef2 and a2c
		Outputs // Problem, hash of tx is not TxID, therefor wee need an identifier to see if it is a cross-tx
			0	// This output is added to UTXOSet with the corresponding Input.TxID
				Value	25 /
				N		0
				PubKey  Owner
		Proof-Of-Consensus
			BlockRepresentation
				GossipHash
				IntermediateHash
				MerkleRoot
				MerkleProof of Cross-tx
			Signature set

cross-tx1-r should be put into the blockchain so it can be used by user if cross-tx2 fails.

When including a cross-tx in a block. A leader (and all nodes) check wheter or not this was the last tx to
fulfill the original Tx. If it was leader should make the final transaction (and if he does not, honest node should not accept the block).

Final transaction given that cross-tx1-r and cross-tx2-r have been recived (and possibly already added to block)

Final transaction improved
	TxID 		qxk		// unsigned
	OrigTxId	a2c
	Inputs
		0	// value 25
			TxId 			ef2
			N				0
			Sig				Sig of ef2 and a2c
		1	// value 20
			TxId			zb4
			N 				0
			Sig				Sig of zb4 and a2c
	Outputs
		0
			Value 	30
			N		0
			PubKey	user_2
		1
			Value 15
			N 		1
			PubKey  user_1 // rest back to self

Added to blockchain and done! abc and skr are validated because they are in the blockchain and its proofOfCosnensus
is validated. And all is good!

QED


user_1 using final transaction Better version?
	Hash 		nro
	OrigTxHash	nil
	Inputs
		0	// value 15
			TxHash 			a2c		// final-cross-tx
			N				1
			Sig				Sig of a2c and nro
	Outputs
		0
			Value 	10
			N		0
			PubKey	user_2
		1
			Value 	5
			N 		1
			PubKey  user_1 // rest back to self

Note: change OrigInpTxID to OrigTxHash



So when leader is creating a block
1. pop all transactions from tx pool
2. all new transactions that require cross-tx
	2.1 Create new transactions (cross-tx) to each input committee
	2.2 Batch these cross-tx together and send to target committees
	2.3 Original transactions TxID moved to OriginalTxID to indicate it is not spendable
3. If there is an incomming cross-tx-r then check if it is the last cross-tx-r to fulfill orig tx
	3.1 If it is, also create final tx and add it to the block after cross-tx-r
4. Add all transactions, original (not-spendable) transactions, cross-tx transactions, and final transactions to block
5. ??
6. Profit! (given an incentive scheme :P)

*/

// createBlock pops transaction from transaction pool and creates a new block
func createProposeBlock(nodeCtx *NodeCtx) *ProposedBlock {

	block := new(ProposedBlock)

	// block all blockchain operations while creating new block
	nodeCtx.blockchain.mux.Lock()

	// set previous block hash
	block.PreviousGossipHash = nodeCtx.blockchain.LatestBlock

	// block tx pool operations

	// get all transactions
	// TODO limit the amout of transactions to be included
	txes := nodeCtx.txPool.getEnoughToFillblock(default_B)

	txes = processTransactions(nodeCtx, txes)

	block.Transactions = txes

	block.Iteration = nodeCtx.i.getI()
	block.CommitteeID = nodeCtx.self.CommitteeID
	block.LeaderPub = nodeCtx.self.Priv.Pub

	tree := createMerkleTree(nodeCtx, txes)

	block.MerkleRoot = toByte32(tree.Root())

	// set hash
	block.setHash()

	// sign hash
	block.LeaderSig = nodeCtx.self.Priv.sign(block.GossipHash)

	// unlock locked mutexes
	nodeCtx.blockchain.mux.Unlock()
	return block
}

// processTransacitons proccesses a list of transactions from txpool and returns
// a list of final transactions ready to be included into a block.
// It handles cross-txes and sends them as well.
func processTransactions(nodeCtx *NodeCtx, txes []*Transaction) []*Transaction {
	/* Question: How do we order transactions?
	Easiest would be to rank them as,
		1. Independant transactions where all inputs belongs to this committe
		2. Incoming cross-tx-r
		3. Final transactions
		4. Incomming cross-tx and resposne creation
		5. New cross-tx transactions
	*/

	/* // transform txes to map for easy deletion
	remainingTxes := make(map[[32]byte]*Transaction)
	for _, t := range txes {
		remainingTxes[t.id()] = t
	} */

	// create output tx list of finished and processesd txes
	processedTxes := []*Transaction{}

	// create a temporary UTXO sets to record spent UTXOs. (Have to wait to delete things untill after block is valid
	spentUTXOSet := new(UTXOSet)
	spentUTXOSet.init()

	addedUTXOSet := new(UTXOSet)
	addedUTXOSet.init()

	tmpCrossTxPool := new(CrossTxPool)
	tmpCrossTxPool.init()

	//normalTxes := make(map[[32]byte]*Transaction)
	//toCrossTxes := make(map[[32]byte]*Transaction)
	//incommingCrossTxes := make(map[[32]byte]*Transaction)
	//crossTxResponses := make(map[[32]byte]*Transaction)

	// sort txes to their corresponding assigments
	for _, trans := range txes {

		t := new(Transaction)
		copier.Copy(t, trans)

		if t.OrigTxHash == [32]byte{} && t.Hash != [32]byte{} {
			// this is a normal transaction
			normal := true
			for _, inp := range t.Inputs {
				closestCommittee := txFindClosestCommittee(nodeCtx, inp.TxHash)
				if closestCommittee != nodeCtx.self.CommitteeID {
					// the input belongs to another committee
					normal = false
					break
				}
			}
			if normal {
				// all inputs belonged to this committee
				// log.Println("Normal transaction, all inputs in this committee")
				// todo add rest of sets here
				res := processNormalTransaction(nodeCtx, t, spentUTXOSet, addedUTXOSet)
				if res {
					processedTxes = append(processedTxes, t)
				}
			} else {
				// some inputs did not belong to this committee
				//toCrossTxes[t.Hash] = t
				// log.Println("Normal transaction, some inputs not in this committee")
				newCrossTxes := processTransactionWithUnknowInputs(nodeCtx, t, spentUTXOSet, addedUTXOSet)
				if len(newCrossTxes) == 0 {
					errFatal(nil, "len of new cross-txes was 0")
				}
				if newCrossTxes != nil {
					if t.OrigTxHash == [32]byte{} || t.Hash != [32]byte{} {
						errFatal(nil, "originaltx hashes was not changed")
					}
					processedTxes = append(processedTxes, t)

					// add original to tmpCrossTxPool
					tmpCrossTxPool.addOriginalTx(nodeCtx, t)
					processedTxes = append(processedTxes, newCrossTxes...)
				}
			}
		} else if t.OrigTxHash != [32]byte{} && t.Hash == [32]byte{} {
			// cross tx
			//incommingCrossTxes[t.OrigTxHash] = t
			// log.Println("Incomming cross-tx")
			ok := processIncommingCrossTx(nodeCtx, t, spentUTXOSet, addedUTXOSet)
			if ok {
				// log.Println("Cross-tx accepted")
				processedTxes = append(processedTxes, t)
			}
		} else if t.OrigTxHash != [32]byte{} && t.Hash != [32]byte{} {
			// A cross-tx-r
			// crossTxResponses[t.id()] = t
			// log.Println("Incomming cross-tx response")
			newTx, ok := proccessCrossTxResponse(nodeCtx, t, spentUTXOSet, addedUTXOSet, tmpCrossTxPool)
			if ok {
				// log.Println("cross-tx-response ok")
				processedTxes = append(processedTxes, t)
				if newTx != nil {
					// log.Println("Final transaction added")
					processedTxes = append(processedTxes, newTx)
				}
			}
		} else {
			errFatal(nil, "this shouldnt be reached?")
		}
	}

	return processedTxes
}

func proccessCrossTxResponse(nodeCtx *NodeCtx,
	t *Transaction,
	spentUTXOSet *UTXOSet,
	addedUTXOSet *UTXOSet,
	tmpCrossTxPool *CrossTxPool) (*Transaction, bool) {

	// calculate time it took to validate
	before := time.Now()

	// verify proof of consensus (PoC):
	if t.ProofOfConsensus == nil {
		errFatal(nil, "Proof of consensus was nil")
	}

	// PoC: validate merkle proof
	tmpid := t.ifOrigRetOrigIfNotRetHash()
	verified, err := merkletree.VerifyProof(tmpid[:], false, (*merkletree.Proof)(t.ProofOfConsensus.MerkleProof), [][]byte{t.ProofOfConsensus.MerkleRoot[:]})
	fail := ifErr(err, "merkletree.Verifyproof")
	if fail || !verified {
		errFatal(err, "proof could not be verified")
	}

	// PoC: validate hashes
	mrHash := hash(t.ProofOfConsensus.MerkleRoot[:])
	valHash := hash(byteSliceAppend(t.ProofOfConsensus.IntermediateHash[:], mrHash[:]))
	if valHash != t.ProofOfConsensus.GossipHash {
		errFatal(nil, fmt.Sprintf("ProofOfConsensus hashes invalid. calculatedhash: %s, GossipHash: %s", bytes32ToString(valHash), bytes32ToString(t.ProofOfConsensus.GossipHash)))
	}

	// PoC: validate signatures:

	if uint(len(t.ProofOfConsensus.Signatures)) < (nodeCtx.flagArgs.n/nodeCtx.flagArgs.m)/nodeCtx.flagArgs.committeeF {
		errFatal(nil, fmt.Sprintf("Len of signatures: %d was lower than required: %d ", len(t.ProofOfConsensus.Signatures), (nodeCtx.flagArgs.n/nodeCtx.flagArgs.m)/nodeCtx.flagArgs.committeeF))
	}
	rBlock := nodeCtx.blockchain._getLastReconfigurationBlock()
	for _, cMsg := range t.ProofOfConsensus.Signatures {
		// verify signature

		ok := cMsg.Pub.verify(cMsg.calculateHash(), cMsg.Sig)
		notOkErr(ok, "signature verify")

		// verify that pub exists in that committee
		// the id of the committee that sent cross-tx-response is the same as the committee that the input belongs to
		comitteeID := txFindClosestCommittee(nodeCtx, t.Inputs[0].TxHash)
		if comitteeID == nodeCtx.self.CommitteeID {
			errFatal(nil, "committee of cross-tx-response was this committee?")
		}
		if _, ok := rBlock.Committees[comitteeID].Members[cMsg.Pub.Bytes]; !ok {
			errFatal(nil, "signature pub did not exist in that committee")
		}
	}

	dur := time.Now().Sub(before)
	go dialAndSendToCoordinator("pocverify", dur)

	// add output to temp
	if len(t.Inputs) != len(t.Outputs) {
		errFatal(nil, "length of crossTxResponse inputs was not equal to len of outputs")
	}

	// get original tx, throw error if the original tx is not found
	original := nodeCtx.crossTxPool.getOriginal(t.OrigTxHash)
	if original == nil {
		original = tmpCrossTxPool.getOriginal(t.OrigTxHash)
		if original == nil {
			//fmt.Println(t)
			errFatal(nil, "no original tx")
		}
	}

	if original.OrigTxHash != t.OrigTxHash {
		errFatal(nil, "orighash not equal")
	}

	// add outputs to addedUTXOSet
	for i := range t.Outputs {
		addedUTXOSet.add(t.Inputs[i].TxHash, t.Outputs[i])
	}

	// add all inputs to crossTxPool map such that we can check if all inputs are satisifed in originalTx
	// tmpCrossTxPool.addResponses(t) // this should be moved?

	// we can concur with CrossTxPool for all inputs that have outputs
	// we can concur with addedUTXO for any inputs that have outputs added in this iteration
	// this should be enoguh to check if all inputs are covered in this transaction.

	// crossTxs := nodeCtx.crossTxPool.getMap(t.OrigTxHash)

	allInputsCovered := true
	for _, inp := range original.Inputs {

		// Check if all inputs from original transaction now exists in the UTXOSet
		outTx := nodeCtx.utxoSet.get(inp.TxHash, inp.N)
		if outTx == nil {
			outTx = addedUTXOSet.get(inp.TxHash, inp.N)
			if outTx == nil {
				allInputsCovered = false
				break
			}
		}

		if spentUTXOSet.get(inp.TxHash, inp.N) != nil {
			allInputsCovered = false
			break
		}

		// if nodeCtx.crossTxPool.getCrossTxMap(t.OrigTxHash, inp.TxHash) == nil && tmpCrossTxPool.getCrossTxMap(t.OrigTxHash, inp.TxHash) == nil {
		// 	// no we dont have a corresponding output
		// 	allInputsCovered = false
		// }
	}

	if !allInputsCovered {
		return nil, true
	}

	newTx := new(Transaction)

	newTx.OrigTxHash = original.OrigTxHash
	newTx.Outputs = original.Outputs

	newInputs := []*InTx{}
	for _, inp := range original.Inputs {
		// 	c := nodeCtx.crossTxPool.getCrossTxMap(t.OrigTxHash, inp.TxHash)
		// 	if c == nil {
		// 		c = tmpCrossTxPool.getCrossTxMap(t.OrigTxHash, inp.TxHash)
		// 		if c == nil {
		// 			errFatal(nil, "should not happen")
		// 		}
		// 	}
		outTx := nodeCtx.utxoSet.get(inp.TxHash, inp.N)
		if outTx == nil {
			outTx = addedUTXOSet.get(inp.TxHash, inp.N)
			if outTx == nil {
				fmt.Println(original)
				fmt.Println(inp)
				fmt.Println(outTx)

				fmt.Println(bytes32ToString(txFindClosestCommittee(nodeCtx, inp.TxHash)))
				fmt.Println(bytes32ToString(nodeCtx.self.CommitteeID))

				if spenttx := spentUTXOSet.get(inp.TxHash, inp.N); spenttx != nil {
					log.Println("original UTXO was allready spent :o ")
					log.Println(spenttx)
					errFatal(nil, "spent")
				}

				errFatal(nil, "outTx did not exists hmmmm")
			}
		}

		// fmt.Println(outTx)
		// if outTx.N != c.Nonce {
		// 	errFatal(nil, "nonces not equal")
		// }

		if spentUTXOSet.get(inp.TxHash, inp.N) != nil {
			fmt.Println(inp)
			fmt.Println(outTx)
			// fmt.Println(bytes32ToString(c.CrossTxResponseID))
			// fmt.Println(c.Nonce)

			fmt.Println("UTXOSet ", nodeCtx.utxoSet)
			fmt.Println("addedUTXOSet ", addedUTXOSet)
			fmt.Println("spentUTXOSet ", spentUTXOSet)
			fmt.Println("original: ", original)

			fmt.Println(bytes32ToString(txFindClosestCommittee(nodeCtx, inp.TxHash)))
			fmt.Println(bytes32ToString(nodeCtx.self.CommitteeID))

			log.Println("UTXO allready spent")
			errFatal(nil, "spent")
		}

		newInp := new(InTx)
		newInp.TxHash = inp.TxHash
		// newInp.OrigTxHash = inp.OrigTxHash
		newInp.N = inp.N
		newInp.Sig = inp.Sig

		newInputs = append(newInputs, newInp)

		// spend output
		spentUTXOSet.add(inp.TxHash, outTx)
	}

	newTx.Inputs = newInputs
	newTx.setHash()
	//closest := txFindClosestCommittee(nodeCtx,newTx.Hash)
	//fmt.Println(closest,"   ",nodeCtx.self.CommitteeID,"=====================")
	// finaly add the original outputs, on the original id
	for _, out := range original.Outputs {
		addedUTXOSet.add(original.OrigTxHash, out)
	}

	// remove from crossTxPool
	// tmpCrossTxPool.removeMap(original.OrigTxHash)
	tmpCrossTxPool.removeOriginal(original.OrigTxHash)
	return newTx, true
}

func processIncommingCrossTx(nodeCtx *NodeCtx, t *Transaction, spentUTXOSet *UTXOSet, addedUTXOSet *UTXOSet) bool {
	// validate inputs.

	if t.Outputs != nil {
		fmt.Print(t)
		fmt.Print(t.whatAmI(nodeCtx))
		fmt.Print(bytes32ToString(txFindClosestCommittee(nodeCtx, t.OrigTxHash)))
		fmt.Println(bytes32ToString(nodeCtx.self.CommitteeID))
		errFatal(nil, "outputs was not nil in incomming cross-tx")
	}

	// fmt.Println(bytes32ToString(nodeCtx.self.CommitteeID), bytes32ToString(txFindClosestCommittee(nodeCtx, t.Inputs[0].TxHash)))
	if nodeCtx.self.CommitteeID != txFindClosestCommittee(nodeCtx, t.Inputs[0].TxHash) {
		fmt.Println(bytes32ToString(nodeCtx.self.CommitteeID), bytes32ToString(txFindClosestCommittee(nodeCtx, t.Inputs[0].TxHash)))
		errFatal(nil, "incomming cross tx not beloning in thsi committee")
	}

	for _, inp := range t.Inputs {
		if !validateInput(nodeCtx, inp, t.OrigTxHash, spentUTXOSet, addedUTXOSet) {
			log.Println("Incoming cross-tx input not valid")
			fmt.Println(t)
			errFatal(nil, "cross-tx")
			return false
		}
	}
	newOuts := make([]*OutTx, len(t.Inputs))

	for i, inp := range t.Inputs {
		newOuts[i] = spendInputToNewOutput(nodeCtx, inp, spentUTXOSet)
	}

	// create new transaction with outputs
	if t.Outputs != nil {
		errFatal(nil, fmt.Sprintf("t.Outputs was not nil: %v", t.Outputs))
	}

	t.Outputs = newOuts

	if t.Hash != [32]byte{} {
		errFatal(nil, fmt.Sprintf("t.Hash was not nil: %v", t.Hash))
	}

	// set a new hash
	t.setHash()

	// DO NOT add newOuts to addedUTXOSet since these outputs belong in another committtee
	// add newOuts to addedUTXOSet
	// for _, out := range newOuts {
	// 	addedUTXOSet.add(t.Hash, out)
	// }

	return true
}

func processTransactionWithUnknowInputs(nodeCtx *NodeCtx, t *Transaction, spentUTXOSet *UTXOSet, addedUTXOSet *UTXOSet) []*Transaction {
	// validate known inputs (but do not spend!)
	// create cross-tx for rest

	newTxs := []*Transaction{}
	newInputs := make(map[[32]byte][]*InTx) // committeeID -> Input
	testtmp := 0
	for _, inp := range t.Inputs {
		closestCommittee := txFindClosestCommittee(nodeCtx, inp.TxHash)
		if closestCommittee != nodeCtx.self.CommitteeID {
			// in another committee
			if len(newInputs[closestCommittee]) == 0 {
				newInputs[closestCommittee] = []*InTx{inp}
			} else {
				newInputs[closestCommittee] = append(newInputs[closestCommittee], inp)
			}

		} else {
			testtmp++
			// in this committe, validate
			if !validateInput(nodeCtx, inp, t.id(), spentUTXOSet, addedUTXOSet) {
				log.Println("input not valid")
				errFatal(nil, "input not valid")
				return nil
			}
			// do not spend inputs
		}
	}

	for _, inps := range newInputs {
		// all
		newTx := new(Transaction)
		newTx.OrigTxHash = t.Hash
		newTx.Inputs = inps
		newTxs = append(newTxs, newTx)
	}

	// set OrigID to TxID, and TxID to nil in original transaction
	t.OrigTxHash = t.Hash
	t.Hash = [32]byte{}

	// if len(t.Inputs) > 1 && testtmp == 0 {
	// 	fmt.Println(t)
	// 	errFatal(nil, "test")
	// }

	return newTxs
}

// processes input. throws error if UTXOset does not have outTx. Returns amount spent
func spendInput(nodeCtx *NodeCtx, iTx *InTx, spentUTXOSet *UTXOSet) uint {
	outTx := nodeCtx.utxoSet.get(iTx.TxHash, iTx.N)
	if outTx == nil {
		errFatal(nil, "outTx was nil")
	}
	spentUTXOSet.add(iTx.TxHash, outTx)
	return outTx.Value
}

// processes input and creates a corresponding output. throws error if UTXOset does not have outTx.
func spendInputToNewOutput(nodeCtx *NodeCtx, iTx *InTx, spentUTXOSet *UTXOSet) *OutTx {
	outTx := nodeCtx.utxoSet.get(iTx.TxHash, iTx.N)
	if outTx == nil {
		errFatal(nil, "outTx was nil")
	}
	spentUTXOSet.add(iTx.TxHash, outTx)
	return outTx
}

func validateInput(nodeCtx *NodeCtx, iTx *InTx, txID [32]byte, spentUTXOSet *UTXOSet, addedUTXOSet *UTXOSet) bool {

	// check if output is not allready spent
	if spentUTXOSet.get(iTx.TxHash, iTx.N) != nil {
		log.Println("UTXO allready spent")
		errFatal(nil, "spent")
		return false
	}

	// check wheter or not inputs have a corresponding UTXO
	outTx := nodeCtx.utxoSet.get(iTx.TxHash, iTx.N)
	if outTx == nil {
		outTx = addedUTXOSet.get(iTx.TxHash, iTx.N)
		if outTx == nil {
			log.Println("No UTXO on this input")
			fmt.Println(iTx)
			errFatal(nil, "utxo")
			return false
		}
	}

	// check signature
	if !verify(outTx.PubKey.Pub, iTx.getHash(txID), iTx.Sig) {
		log.Println("Signature not valid")
		fmt.Println("Pubkey ", bytesToString(outTx.PubKey.Bytes[:]))
		fmt.Println("Out tx N: ", outTx.N, "InTX N: ", iTx.N)
		fmt.Println("OutTx", outTx)
		fmt.Println("Input", iTx)
		fmt.Println("txhash", bytesToString(iTx.TxHash[:]))
		return false
	}

	return true
}

// processTransaction removes UTXO's from UTXOSet and creates new ones, and validates that
// all inputs are correct. Returns if the transaction is succesfull or not.
// All inputs in transactions must belong in this committee, otherwise returns false
func processNormalTransaction(nodeCtx *NodeCtx, t *Transaction, spentUTXOSet *UTXOSet, addedUTXOSet *UTXOSet) bool {
	if !validateNormalTransaction(nodeCtx, t, spentUTXOSet, addedUTXOSet) {
		return false
	}
	// now we know that all UTXOs are spendable

	// spend all inputs
	for _, inp := range t.Inputs {
		// spend from either normal utxoSet or addedUTXOSet
		outTx := nodeCtx.utxoSet.get(inp.TxHash, inp.N)
		if outTx != nil {
			spentUTXOSet.add(inp.TxHash, outTx)
		} else {
			// the input was not in normal utxoSet, therefor it is in addedUTXOSet
			outTx = addedUTXOSet.getAndRemove(inp.TxHash, inp.N)
			if outTx != nil {
				spentUTXOSet.add(inp.TxHash, outTx)
			} else {
				errFatal(nil, "OutTx was not present in neither normal utxoset or addedUTXOSet")
			}
		}
	}

	// add outputs to addedUTXOSet
	for _, out := range t.Outputs {
		addedUTXOSet.add(t.Hash, out)
	}

	return true
}

// Validates wheter or not a normal transaction (not cross-tx) is valid. Returns true if valid, false if not.
// All inputs in transaction must be in this committee, returns false if not.
func validateNormalTransaction(nodeCtx *NodeCtx, t *Transaction, spentUTXOSet *UTXOSet, addedUTXOSet *UTXOSet) bool {
	// A transaction is valid when the UTXO set has all inputs.
	// Asuming the UTXO set is verifed.

	var totalUTXOValue uint = 0
	for _, inp := range t.Inputs {
		// check wheter or not inputs have a corresponding UTXO

		// check if output is not allready spent
		if spentUTXOSet.get(inp.TxHash, inp.N) != nil {
			log.Println("UTXO allready spent")
			errFatal(nil, "UTXO allready spent")
			return false
		}

		outTx := nodeCtx.utxoSet.get(inp.TxHash, inp.N)
		if outTx == nil {
			outTx = addedUTXOSet._get(inp.TxHash, inp.N)
			if outTx == nil {
				log.Println("No UTXO on this input")
				fmt.Println(t)
				errFatal(nil, "No UTXO on this input")
				return false
			}
		}

		// check signature
		if !verify(outTx.PubKey.Pub, inp.getHash(t.id()), inp.Sig) {
			log.Println("Signature not valid")

			fmt.Println("Pubkey ", bytesToString(outTx.PubKey.Bytes[:]))

			fmt.Println("Out tx N: ", outTx.N, "InTX N: ", inp.N)
			fmt.Println("OutTx", outTx)
			fmt.Println("Input", inp)
			fmt.Println("Spec: ", inp.N, inp.TxHash)
			errFatal(nil, "signature")
			return false
		}

		totalUTXOValue += outTx.Value
	}

	var totalOutputValue uint = 0
	for _, out := range t.Outputs {
		totalOutputValue += out.Value
	}

	if totalOutputValue != totalUTXOValue {
		log.Printf("Total output value in transaction %d not equal to total UTXO value %d", totalOutputValue, totalUTXOValue)
		errFatal(nil, "value")
		return false
	}

	return true
}
