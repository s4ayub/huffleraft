package huffleraft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb"
	"log"
	"net"
	"net/http"
	"net/url"
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
	kvs        *BadgerKV
	mu         sync.Mutex
	RaftServer *raft.Raft
	ln         net.Listener
	logger     *log.Logger
}

func NewRaftKV(raftDir, raftAddr string, kvs *BadgerKV, enableSingle bool) (*RaftStore, error) {
	httpAddr, err := getHttpAddr(raftAddr)
	if err != nil {
		return nil, err
	}

	rs := &RaftStore{
		httpAddr: httpAddr,
		raftDir:  raftDir,
		raftAddr: raftAddr,
		kvs:      kvs,
		logger:   log.New(os.Stderr, fmt.Sprintf("[raftStore | %s]", raftAddr), log.LstdFlags),
	}
	if err := os.MkdirAll(raftDir, 0700); err != nil {
		return nil, err
	}

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

func getHttpAddr(raftAddr string) (string, error) {
	addrParts := strings.Split(raftAddr, ":")
	httpHost := addrParts[0]
	port, err := strconv.Atoi(addrParts[1])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", httpHost, port+1), nil
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

// Set sets the value for the given key.
func (rs *RaftStore) Get(key string) ([]byte, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.kvs.Get([]byte(key))
}

// Set sets the value for the given key.
func (rs *RaftStore) Set(key, value string) error {
	b, err := json.Marshal(map[string]string{"key": key, "val": value})
	if err != nil {
		return err
	}

	httpAddr, err := getHttpAddr(rs.RaftServer.Leader())
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/%s/key", httpAddr, rs.raftDir),
		"application-type/json",
		bytes.NewReader(b),
	)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}

// Detete deletes given key.
func (rs *RaftStore) Delete(key string) error {
	httpAddr, err := getHttpAddr(rs.RaftServer.Leader())
	if err != nil {
		return err
	}

	u, err := url.Parse(fmt.Sprintf("http://%s/%s/key/%s", httpAddr, rs.raftDir, key))
	if err != nil {
		return err
	}

	req := &http.Request{
		Method: "DELETE",
		URL:    u,
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Detete deletes given key.
func (rs *RaftStore) Join(addr string) error {
	b, err := json.Marshal(map[string]string{"addr": addr})
	if err != nil {
		return err
	}

	var postAddr string
	if rs.RaftServer.Leader() == "" {
		postAddr = rs.raftAddr
	} else {
		postAddr = rs.RaftServer.Leader()
	}

	httpAddr, err := getHttpAddr(postAddr)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/%s/join", httpAddr, rs.raftDir),
		"application-type/json",
		bytes.NewReader(b),
	)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}
