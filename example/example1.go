package main

import (
	"fmt"
	"github.com/s4ayub/huffleraft"
	"strconv"
	"time"
)

func main() {
	kvs, _ := huffleraft.NewBadgerKV("./badgerdb1")
	rkv, _ := huffleraft.NewRaftKV(
		"./tonks",
		"127.0.0.1:8790",
		kvs,
		true,
	)
	rkv.Start()
	time.Sleep(2000 * time.Millisecond)

	rkv.Set("huffle", "puff")
	rkv.Set("gryffin", "dor")
	rkv.Set("syther", "in")
	rkv.Set("raven", "claw")

	fmt.Println("=====Before Delete=====")
	a, _ := rkv.Get("huffle")
	fmt.Println(string(a))
	fmt.Println("=======================")

	rkv.Delete("huffle")
	time.Sleep(2000 * time.Millisecond)

	fmt.Println("=====After delete=====")
	b, _ := rkv.Get("huffle")
	fmt.Println(b) // Should be blank array byte
	fmt.Println("=======================")

	// Start the second raft
	kvs2, _ := huffleraft.NewBadgerKV("./badgerdb2")
	rkv2, _ := huffleraft.NewRaftKV(
		"./cedric",
		"127.0.0.1:8795",
		kvs2,
		false,
	)
	fmt.Println("listening")
	rkv2.Start()

	// The system should be in an indeterminate state until the 3rd node joins
	// because quorum of votes won't be reached with only 2 nodes in a cluster
	rkv.Join("127.0.0.1:8795")

	// Start the third raft
	kvs3, _ := huffleraft.NewBadgerKV("./badgerdb3")
	rkv3, _ := huffleraft.NewRaftKV(
		"./newt",
		"127.0.0.1:8798",
		kvs3,
		false,
	)
	rkv3.Start()

	// With 3 nodes, the consensus algorithm can elect a leader now
	rkv.Join("127.0.0.1:8798")
	time.Sleep(5000 * time.Millisecond)
	fmt.Println(string("=====EVERYBODY AT ONCE NOW====="))

	// Lets perform Gets at the other nodes
	c, _ := rkv3.Get("syther")
	fmt.Println(string(c))

	d, _ := rkv2.Get("raven")
	fmt.Println(string(d))

	// Lets try a lot of sets at one of the newly joined nodes
	for i := 0; i < 100; i++ {
		t := strconv.Itoa(i)
		rkv3.Set(t, "yes")
	}
	time.Sleep(5000 * time.Millisecond)
	// Lets ensure data looks the same across all nodes
	e, _ := rkv.Get("3")
	fmt.Println(e)

	f, _ := rkv2.Get("3")
	fmt.Println(f)

	g, _ := rkv3.Get("3")
	fmt.Println(g)

	rkv3.Delete("3")
	time.Sleep(5000 * time.Millisecond)
	// Lets ensure data looks the same across all nodes after a delete
	h, _ := rkv.Get("3")
	fmt.Println(h)

	i, _ := rkv2.Get("3")
	fmt.Println(i)

	j, _ := rkv3.Get("3")
	fmt.Println(j)

	// Ensure joins work on other nodes as well
	kvs4, _ := huffleraft.NewBadgerKV("./badgerdb4")
	rkv4, _ := huffleraft.NewRaftKV(
		"./sprout",
		"127.0.0.1:8750",
		kvs4,
		false,
	)
	rkv4.Start()
	rkv3.Join("127.0.0.1:8750")

	// Start a fifth server
	kvs5, _ := huffleraft.NewBadgerKV("./badgerdb5")
	rkv5, _ := huffleraft.NewRaftKV(
		"./ernie",
		"127.0.0.1:8755",
		kvs5,
		false,
	)
	fmt.Println("listening")
	rkv5.Start()

	rkv2.Join("127.0.0.1:8755") // 5 node cluster has now begun
	time.Sleep(5000 * time.Millisecond)

	z, _ := rkv5.Get("7")
	fmt.Println(string(z))
}
