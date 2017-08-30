package huffleraft

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

type command struct {
	Op    string `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Addr  string `json:"addr,omitempty"`
}

type RaftStore struct {
	httpAddr   string
	raftDir    string
	raftAddr   string
	httpPort   int
	kvs        *BadgerKV
	mu         sync.Mutex
	RaftServer *raft.Raft
	router     *mux.Router
	logger     *log.Logger
}

func NewRaftKV(raftDir, raftAddr string, kvs *BadgerKV, enableSingle bool) (*RaftStore, error) {
	httpAddr, httpPort, err := getHttpAddr(raftAddr)
	if err != nil {
		return nil, err
	}

	rs := &RaftStore{
		httpAddr: httpAddr,
		httpPort: httpPort,
		raftDir:  raftDir,
		raftAddr: raftAddr,
		kvs:      kvs,
		router:   mux.NewRouter(),
		logger:   log.New(os.Stderr, "[raftStore] ", log.LstdFlags),
	}
	if err := os.MkdirAll(raftDir, 0700); err != nil {
		return nil, err
	}

	// Handle the requests so they can be redirected to the leader
	rs.router.HandleFunc("/raftkv_join", rs.handleJoin)
	rs.router.HandleFunc("/raftkv_set", rs.redirectedSet)
	rs.router.HandleFunc("/raftkv_del", rs.redirectedDel)

	config := raft.DefaultConfig()
	transport, err := setupRaftCommunication(rs.raftAddr)
	if err != nil {
		return nil, err
	}

	// Create peer storage and check for existing storage
	peerStore := raft.NewJSONPeers(rs.raftDir, transport)
	peers, err := peerStore.Peers()
	if err != nil {
		return nil, err
	}

	// EnableSingleNode allows for a single node mode of operation, meaning
	// a node can elect itself as a leader. This is set if explicitly enabled
	// and there is only 1 node in the cluster already.
	if enableSingle && len(peers) <= 1 {
		rs.logger.Println("enabling single-node mode")
		config.EnableSingleNode = true
		config.DisableBootstrapAfterElect = false
	}

	// Create the snapshot store. This allows the Raft to truncate the log to
	// mitigate the issue of having an unbounded replicated log.
	snapshots, err := raft.NewFileSnapshotStore(rs.raftDir, retainSnapshotCount, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("file snapshot store: %s", err)
	}

	// Create the log store and stable store. This is the store used to keep
	// the raft logs.
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(rs.raftDir, "raft.db"))
	if err != nil {
		return nil, fmt.Errorf("new bolt store: %s", err)
	}

	// Instantiate the Raft systems. The second parameter is a finite state machien
	// which stores the actual kv pairs and is operated upon through Apply().
	rft, err := raft.NewRaft(config, (*fsm)(rs), logStore, logStore, snapshots, peerStore, transport)
	if err != nil {
		return nil, fmt.Errorf("new raft: %s", err)
	}

	rs.RaftServer = rft
	return rs, nil
}

func getHttpAddr(raftAddr string) (string, int, error) {
	addrParts := strings.Split(raftAddr, ":")
	httpHost := addrParts[0]
	port, err := strconv.Atoi(addrParts[1])
	if err != nil {
		return "", 0, err
	}
	httpPort := port + 1

	return fmt.Sprintf("%s:%d", httpHost, httpPort), httpPort, nil
}

func setupRaftCommunication(raftAddr string) (*raft.NetworkTransport, error) {
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	return transport, nil
}

// Leader returns the server's leader
func (rs *RaftStore) Leader() string {
	return rs.RaftServer.Leader()
}

// ListenAndServe starts the httpServer
func (rs *RaftStore) ListenAndServe() error {
	fmt.Println("=======WE IN LISTEN AND SERVER")
	httpServer := &http.Server{
		Addr:    ":80",
		Handler: rs.router,
	}
	fmt.Println(httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil {
		fmt.Println("=======WE IN ERROR")
		fmt.Println(err.Error())
		return err
	}

	return nil
}

// Set sets the value for the given key.
func (rs *RaftStore) Get(key string) ([]byte, error) {
	return rs.kvs.Get([]byte(key))
}

// Set sets the value for the given key.
func (rs *RaftStore) Set(key, value string) error {
	fmt.Println("=======WE IN SET")
	m := map[string]string{"key": key, "value": value}
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(m); err != nil {
		return err
	}

	_, err := rs.redirectToLeader("raftkv_set", b)
	fmt.Println("=======WE IN SET check the error")
	fmt.Println(err.Error())
	return err
}

// Detete deletes given key.
func (rs *RaftStore) Delete(key string) error {
	m := map[string]string{"key": key}
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(m); err != nil {
		return err
	}

	_, err := rs.redirectToLeader("raftkv_delete", b)
	return err
}

// Detete deletes given key.
func (rs *RaftStore) Join(addr string) error {
	m := map[string]string{"addr": addr}
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(m); err != nil {
		return err
	}

	_, err := rs.redirectToLeader("raftkv_join", b)
	return err
}

//Redirects the request to the leader
func (rs *RaftStore) redirectToLeader(op string, b bytes.Buffer) ([]byte, error) {
	fmt.Println("WE INSIDE REDIRECT TO LEADER =======")
	reply, err := processRequest(rs.RaftServer.Leader(), op, "application/json", &b)
	if err != nil {
		fmt.Println("WE INSIDE REDIRECT TO LEADER ERROR =======")
		fmt.Println(err.Error())
		return nil, err
	}
	fmt.Println("WE INSIDE REDIRECT TO LEADER REPLY =======")
	fmt.Println(string(reply))
	return reply, nil
}

func processRequest(leader, op, contentType string, b *bytes.Buffer) ([]byte, error) {
	fmt.Println("WE INSIDE PROCESS REQUEST=======")
	httpAddr, _, err := getHttpAddr(leader)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	target := fmt.Sprintf("http://%s/%s", httpAddr, op)
	resp, err := http.Post(target, contentType, b)
	if err != nil {
		fmt.Println("WE INSIDE PROCESS REQUEST RESP ERROR =======")
		fmt.Println(err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	reply, _ := ioutil.ReadAll(resp.Body)
	statusCode := resp.StatusCode
	if statusCode != http.StatusOK {
		fmt.Println("WE INSIDE PROCESS REQUEST RESP ERROR  STATUS CODE=======")
		fmt.Println(string(reply))
		return nil, errors.New(string(reply))
	}

	fmt.Println("WE INSIDE PROCESS REQUEST END =======")
	fmt.Println(string(reply))
	return reply, nil
}

// Applies the set command to the raft server. The actual set is
// applied to the key-value store in the finite state machine (fsm.go).
func (rs *RaftStore) redirectedSet(w http.ResponseWriter, req *http.Request) {
	m := map[string]string{}
	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c := &command{
		Op:    "set",
		Key:   m["key"],
		Value: m["value"],
	}
	b, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f := rs.RaftServer.Apply(b, raftTimeout)
	if f.Error() != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rs.logger.Printf("key: %s, value: %s set", m["key"], m["value"])
}

// Applies the delete command to the raft server. The actual delete is
// applied to the key-value store in the finite state machine (fsm.go).
func (rs *RaftStore) redirectedDel(w http.ResponseWriter, req *http.Request) {
	m := map[string]string{}
	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Rebuilding the command from the http request to be applied to
	// the raft log.
	c := &command{
		Op:  "delete",
		Key: m["key"],
	}
	b, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f := rs.RaftServer.Apply(b, raftTimeout)
	if f.Error() != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rs.logger.Printf("key: %s deleted", m["key"])
}

// Adds a peer to the raft server
func (rs *RaftStore) handleJoin(w http.ResponseWriter, r *http.Request) {
	m := map[string]string{}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	remoteAddr := m["addr"]
	f := rs.RaftServer.AddPeer(remoteAddr)
	if f.Error() != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rs.logger.Printf("node at %s joined successfully", remoteAddr)
}
