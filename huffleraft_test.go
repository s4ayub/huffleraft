package huffleraft_test

import (
	"github.com/s4ayub/huffleraft"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// There should be no errors upon starting the http server
func TestSetGetDelete(t *testing.T) {
	kvs, _ := huffleraft.NewBadgerKV("./badgerdb1")
	rkv, _ := huffleraft.NewRaftKV("./tonks", "127.0.0.1:8790", kvs, true)
	rkv.Start()
	time.Sleep(5000 * time.Millisecond)

	setErr := rkv.Set("pierce", "hawthorne")
	assert.Nil(t, setErr)
	time.Sleep(5000 * time.Millisecond)

	val, getErr := rkv.Get("pierce")
	assert.Nil(t, getErr)
	assert.Equal(t, "hawthorne", string(val), "Get should retrieve this value")

	delErr := rkv.Delete("pierce")
	time.Sleep(5000 * time.Millisecond)
	deletedVal, _ := rkv.Get("pierce")

	assert.Nil(t, setErr)
	assert.Equal(t, "hawthorne", string(val), "Get should retrieve value")
	assert.Nil(t, delErr)
	assert.Equal(t, "", string(deletedVal), "Get should return an empty string if key doesn't exist")
}

// The following will test joining a cluster, propagating logs to followers and
// redirecting a command to the leader
func Test3NodeCluster(t *testing.T) {
	kvs, _ := huffleraft.NewBadgerKV("./bdb")
	rkv, _ := huffleraft.NewRaftKV("./rkv", "127.0.0.1:8750", kvs, true)
	rkv.Start()
	time.Sleep(5000 * time.Millisecond)
	rkv.Set("pierce", "hawthorne")

	kvs2, _ := huffleraft.NewBadgerKV("./bdb2")
	rkv2, _ := huffleraft.NewRaftKV("./rkv2", "127.0.0.1:8730", kvs2, false)
	rkv2.Start()
	time.Sleep(5000 * time.Millisecond)

	kvs3, _ := huffleraft.NewBadgerKV("./bdb3")
	rkv3, _ := huffleraft.NewRaftKV("./rkv3", "127.0.0.1:8740", kvs3, false)
	rkv3.Start()
	time.Sleep(5000 * time.Millisecond)

	firstJoinErr := rkv.Join("127.0.0.1:8740")
	secondJoinErr := rkv.Join("127.0.0.1:8730")
	time.Sleep(5000 * time.Millisecond)

	// The key,value pair set in the first node should be propagated to the nodes that joined
	rkv3Get, _ := rkv3.Get("pierce")

	// Doesn't matter who the leader actually is, this command will be redirected to it
	// and the value can be retrieved from any node
	redirectedSetErr := rkv2.Set("yes", "no")
	time.Sleep(5000 * time.Millisecond)
	rkvGet, _ := rkv.Get("yes")

	assert.Nil(t, firstJoinErr)
	assert.Nil(t, secondJoinErr)
	assert.Equal(t, "hawthorne", string(rkv3Get), "Get should return the value for the key")
	assert.Nil(t, redirectedSetErr)
	assert.Equal(t, "no", string(rkvGet), "Get should return the value for the key")
}
