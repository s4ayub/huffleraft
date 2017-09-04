package huffleraft

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// Start starts the http service using the listener within a RaftStore.
// The HTTP server is used to redirect commands like Set, Delete and Join
// to the leader RaftStore in a cluster. The HTTP address is always
// one away from the Raft address which the raft node uses for communication
// with other raft nodes.
func (rs *RaftStore) Start() error {
	server := http.Server{
		Handler: rs,
	}

	ln, err := net.Listen("tcp", rs.httpAddr)
	if err != nil {
		return err
	}
	rs.ln = ln

	http.Handle(fmt.Sprintf("/%s", rs.raftDir), rs)

	go func() {
		err := server.Serve(rs.ln)
		if err != nil {
			log.Fatalf("HTTP serve: %s", err)
		}
	}()

	return nil
}

// Close closes the HTTP server of the RaftStore.
func (rs *RaftStore) Close() {
	rs.ln.Close()
	return
}

// ServeHTTP allows the RaftStore to serve HTTP requests.
func (rs *RaftStore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "/key") {
		rs.handleKeyRequest(w, r)
	} else if strings.Contains(r.URL.Path, "/join") {
		rs.handleJoin(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleJoin actually applies the join upon receiving the http request.
func (rs *RaftStore) handleJoin(w http.ResponseWriter, r *http.Request) {
	m := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&m)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	remoteAddr := m["addr"]

	f := rs.RaftServer.AddPeer(remoteAddr)
	if f.Error() != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rs.logger.Printf("node at %s joined successfully", remoteAddr)
}

// handleKeyRequest actually applies the set or delete commands upon receiving the http request.
func (rs *RaftStore) handleKeyRequest(w http.ResponseWriter, r *http.Request) {
	getKey := func() string {
		parts := strings.Split(r.URL.Path, "/")
		return parts[len(parts)-1]
	}

	switch r.Method {
	case "POST":
		// Read the value from the POST body.
		m := map[string]string{}
		err := json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		c := &command{
			Op:    "set",
			Key:   m["key"],
			Value: m["val"],
		}

		b, err := json.Marshal(c)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		f := rs.RaftServer.Apply(b, raftTimeout)
		if f.Error() != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rs.logger.Printf("key: %s, value: %s, set", m["key"], m["val"])

	case "DELETE":
		k := getKey()
		if k == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		c := &command{
			Op:  "delete",
			Key: k,
		}

		b, err := json.Marshal(c)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		f := rs.RaftServer.Apply(b, raftTimeout)
		if f.Error() != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rs.logger.Printf("key: %s, deleted", k)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	return
}
