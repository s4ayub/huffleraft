<p align="center">
  <a href="https://github.com/s4ayub/huffleraft/" target="_blank">
    <img width="100"src="https://user-images.githubusercontent.com/16456972/30003911-80f84b88-9093-11e7-8cb3-17ffc594ddce.png">
  </a>
</p>
<h1 align="center">huffleraft</h1>
<p align="center">
  Distributed fault-tolerant key-value store driven by 
  <a href="https://github.com/hashicorp/raft" target="_blank">hashicorp/raft</a> and 
  <a href="https://github.com/dgraph-io/badger" target="_blank">dgraph-io/badger</a>
</p>
<p align="center">
  <strong>Please read the 
    <a href="https://github.com/s4ayub/huffleraft/blob/master/example/example1.go" target="_blank">example</a> 
    to see the system in action, especially for CRUD operations on a 5 node cluster!
  </strong>
  <br/>
  <strong>  
      For full documentation and more thorough descriptions: <a href="https://godoc.org/github.com/s4ayub/huffleraft" target="_blank">godoc</a>
  </strong>

</p>
<p align="center">
  <a href="https://travis-ci.org/s4ayub/huffleraft" target="_blank"><img src="https://travis-ci.org/s4ayub/huffleraft.svg?branch=master" alt="Build Status" /> </a>
</p>
  
import with: `go get github.com/s4ayub/huffleraft`



### Features and Purpose:

The purpose of this package is to explore the raft consensus algorithm, specifically <a href="https://github.com/hashicorp/raft" target="_blank">hashicorp's implementation</a>. This package is **effective for experimenting with raft** because the API is such that the user doesn't even have to build any HTTP requests, rather, the package does that in the background through Get, Set, Delete, and Join. This makes it very easy and quick to play around with raft in conjunction with other code.
 - Perform CRUD operations on draph-io's Badger storage in a distributed manner
 - Fault tolerant as per the <a href="https://raft.github.io/" target="_blank">raft consensus algorithm</a>
 - Commands can be performed on any node in a cluster and they'll be **redirected to the leader of the cluster**

---

### Example logs:
Once a couple raftservers are running, and a cluster is formed, the logs may look something like this after a Set command has been performed on them:

```
2017/09/04 04:02:07 [DEBUG] raft: Node 127.0.0.1:8740 updated peer set (2): [127.0.0.1:8730 127.0.0.1:8750 127.0.0.1:8740]
2017/09/04 04:02:07 [DEBUG] raft-net: 127.0.0.1:8730 accepted connection from: 127.0.0.1:62560
2017/09/04 04:02:07 [DEBUG] raft-net: 127.0.0.1:8740 accepted connection from: 127.0.0.1:62561
[raftStore | 127.0.0.1:8750]2017/09/04 04:02:11 key: yes, value: no, set
```

--- 
### Motivation and References:

I wanted to dive deeper into distributed systems by building one of my own. The following repository was helpful in the development of huffleraft:
  - <a href="https://github.com/otoolep/hraftd" target="_blank">hraftd</a>
    - This is a simple reference use of hashicorp/raft also made to study the implementation of the consensus algorithm. However, it required the user to interact with the system as a client and use curl to submit requests to the server. My package builds on top of this by being embeddable inside code, implementing command redirection to the leader node and by using dragph-io's badger as a faster storage compared to a native Go map. The people at hashicorp were very quick to answer any questions I had as well.

---

### Design Decisions:
- Leader re-direction:
  - The `RaftStore` struct encases an http listener to handle requests and a `RaftServer` to handle the consensus. There are two important addresses associated with each raft store. The httpAddr listens for http requests, and the raftAddr which the raft server uses for its transport layer between raft nodes. As per the consensus algorithm, commands such as setting, deleting and joining must be sent to the leader node only. The leader's raftAddr can easily be accessed in a cluster by `raftServer.Leader()` (raftServer being an instance of Raft from hashicorp/raft). However, the httpAddr associated  cannot be easily retrieved, hence, a link must be made between the raftAddr and the httpAddr, since the httpAddr is the one that the user would be sending their commands to. I decided that the httpAddr would always be 1 port away from the raftAddr. The httpAddr can always be retrieved now by adding 1 to raftAddr.

- Using <a href="https://github.com/dgraph-io/badger" target="_blank">dgraph-io/badger</a>:
  - Bagder was proposed as a performant Go alternative to RocksDB which made it an attractive storage back-end for the key-value store. 

---

### Caveats and Suggested Improvements:
- Time delays must be put after certain commands to ensure that the command has been performed, and propagated to all nodes before further code can run
- Cannot use side by side ports (8866 and 8867 for example) for two raft servers because the port following that which a raft server is initiated on is used for the http server. This is done to establish a link between the raftAddr and the httpAddr as discussed above. This system can likely be improved upon.
- More robust error handling

---
