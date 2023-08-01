# MSLShard
This implementation belong to the current work. (todo insert link here once it becomes available)

A proof of concept derivation of the implementation presented in Rapidchain.

A good place to start would be main.go.

testComplete_test.go is a simulation test of the trust management model in MSLShard and testRapidChain_test.go is a simulation test of adding trust management to Rapidchain.

For the overall run, it is realized by the following code:

Run reference committee (cloud blockchain):
```
. /ra -function coordinator
```
Run the nodes in each slice:
```
. /ra -function node
```

When running the blockchain, first initialize the node's trust value, network distance, QoS, QoSec, etc. by generating new_data.csv via R. Initial sharding is done via shard.go.

This is just a simple implementation of the basic functions of the paper, and will be updated to improve all the content.