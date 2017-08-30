package main

import (
	"fmt"
	"github.com/s4ayub/huffleraft"
	"time"
)

func main() {
	kvs, _ := huffleraft.NewBadgerKV("./badgerdb1")
	fmt.Println("listening")
	rkv, _ := huffleraft.NewRaftKV(
		"./raft1",
		"127.0.0.1:8790",
		kvs,
		true,
	)
	fmt.Println("listening")
	err := rkv.ListenAndServe()
	fmt.Println(err.Error())
	time.Sleep(2000 * time.Millisecond)
	fmt.Println("helloo")
	erra := rkv.Set("yes", "no")
	fmt.Println(erra)
	fmt.Println("helloooo")

	// time.Sleep(2000 * time.Millisecond)
	// rkv.Set("hi", "bye")
	// time.Sleep(2000 * time.Millisecond)
	// rkv.Set("crazy", "guy")
	// time.Sleep(2000 * time.Millisecond)
	// hello, _ := rkv.Get("yes")
	// time.Sleep(2000 * time.Millisecond)
	// nyeah, _ := rkv.Get("crazy")
	//
	// fmt.Println(string(hello))
	// fmt.Println(string(nyeah))
	//
	// rkv.Delete("crazy")
	// time.Sleep(2000 * time.Millisecond)
	// ayyooo, _ := rkv.Get("crazy")
	// fmt.Println(ayyooo)
	//
	// fmt.Println("=======")
	// fmt.Println("=======")
	// //
	// kvs2, _ := huffleraft.NewBadgerKV("./badgerdb2")
	// rkv2, _ := huffleraft.NewRaftKV(
	// 	"./raft2",
	// 	"127.0.0.1:8795",
	// 	kvs2,
	// 	false,
	// )
	//
	// time.Sleep(2000 * time.Millisecond)
	// joinErr := rkv.Join("127.0.0.1:8800")
	// fmt.Println(joinErr)
	// time.Sleep(2000 * time.Millisecond)
	// //
	//
	// fmt.Println(rkv2)
	// fmt.Println("=======")
	// fmt.Println("=======")
	// //
	// kvs3, _ := huffleraft.NewBadgerKV("./badgerdb3")
	// rkv3, _ := huffleraft.NewRaftKV(
	// 	"./raft3",
	// 	"127.0.0.1:8800",
	// 	kvs3,
	// 	false,
	// )
	//
	// time.Sleep(2000 * time.Millisecond)
	// fmt.Println(rkv3)
	// yahh := rkv.Join("127.0.0.1:8800")
	// fmt.Println(yahh)
	// time.Sleep(2000 * time.Millisecond)
	//
	// nah, _ := rkv3.Get("yes")
	// fmt.Println(string(nah))
	// go rkv3.ListenAndServe()
	// time.Sleep(2000 * time.Millisecond)
	//
	// es := rkv.Join("8800")
	// fmt.Println("====es===")
	// fmt.Println(es)
	// fmt.Println("=======")
	// time.Sleep(1300* time.Millisecond)
	//
	// fmt.Println("====rkv3 leader===")
	// fmt.Println(rkv3.Leader())
	// fmt.Println("=======")
	//
	// fmt.Println("===rkv2 leader====")
	// fmt.Println(rkv2.Leader())
	// fmt.Println("=======")
	//
	// erra := rkv.Set("yes", "no")
	// time.Sleep(500 * time.Millisecond)
	// fmt.Println("====erra===")
	// fmt.Println(erra)
	// hello, err := rkv.Get("yes")
	// fmt.Println("====err===")
	// fmt.Println(err)
	// fmt.Println("=======")
	// fmt.Println("====hello===")
	// fmt.Println(string(hello))
	// fmt.Println("=======")

	// func (rs *RaftStore) redirectedGet(w http.ResponseWriter, req *http.Request) {
	// 	m := map[string]string{}
	// 	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 		return
	// 	}
	//
	// 	rs.mu.Lock()
	// 	defer rs.mu.Unlock()
	//
	// 	val, err := rs.kvs.Get([]byte(m["key"]))
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	}
	// 	w.Write(val)
	// 	return
	// }
}
