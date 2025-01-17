// Copyright 2013-2018 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graft

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"
	"testing"
	"time"
)

var (
	clusterFormationTimeout = MaxElectionTimeout + 100*time.Millisecond
)

type dummyHandler struct {
}

func (*dummyHandler) AsyncError(err error)        {}
func (*dummyHandler) StateChange(from, to State)  {}
func (*dummyHandler) CurrentState() []byte        { return nil }
func (*dummyHandler) GrantVote(state []byte) bool { return true }

func stackFatalf(t *testing.T, f string, args ...interface{}) {
	lines := make([]string, 0, 32)
	msg := fmt.Sprintf(f, args...)
	lines = append(lines, msg)

	// Generate the Stack of callers: Skip us and verify* frames.
	for i := 1; true; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		msg := fmt.Sprintf("%d - %s:%d", i, file, line)
		lines = append(lines, msg)
	}
	t.Fatalf("%s", strings.Join(lines, "\n"))
}

func genNodeArgs(t *testing.T) (Handler, RPCDriver, string) {
	hand := &dummyHandler{}
	rpc := NewMockRpc()
	log, err := ioutil.TempFile("", "_grafty_log")
	if err != nil {
		t.Fatal("Could not create the log file")
	}
	defer log.Close()
	return hand, rpc, log.Name()
}

func createNodes(t *testing.T, name string, numNodes int) []*Node {
	ci := &ClusterInfo{name: name, size: numNodes}
	nodes := make([]*Node, numNodes)
	for i := 0; i < numNodes; i++ {
		hand, rpc, logPath := genNodeArgs(t)
		node, err := New(ci, hand, rpc, logPath)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		nodes[i] = node
	}
	return nodes
}

func countTypes(nodes []*Node) (leaders, followers, candidates int) {
	for _, n := range nodes {
		switch n.State() {
		case LEADER:
			leaders++
		case FOLLOWER:
			followers++
		case CANDIDATE:
			candidates++
		}
	}
	return
}

func findLeader(nodes []*Node) *Node {
	for _, n := range nodes {
		if n.State() == LEADER {
			return n
		}
	}
	return nil
}

func firstFollower(nodes []*Node) *Node {
	for _, n := range nodes {
		if n.State() == FOLLOWER {
			return n
		}
	}
	return nil
}

func expectedClusterState(t *testing.T, nodes []*Node, leaders, followers, candidates int) {
	var currentLeaders, currentFollowers, currentCandidates int
	end := time.Now().Add(clusterFormationTimeout)
	for time.Now().Before(end) {
		currentLeaders, currentFollowers, currentCandidates = countTypes(nodes)
		switch {
		case leaders != currentLeaders, followers != currentFollowers, candidates != currentCandidates:
			time.Sleep(5 * time.Millisecond)
			continue
		default:
			return
		}
	}
	stackFatalf(t, "Cluster doesn't match expect state: expected %d leaders, %d followers, %d candidates, "+
		"actual %d Leaders, %d followers, %d candidates\n",
		leaders, followers, candidates, currentLeaders, currentFollowers, currentCandidates)
}

func waitForLeader(node *Node, expectedLeader string) string {
	curLeader := ""
	timeout := time.Now().Add(5 * time.Second)
	for time.Now().Before(timeout) {
		curLeader = node.Leader()
		if curLeader == expectedLeader {
			return expectedLeader
		}
		time.Sleep(10 * time.Millisecond)
	}
	return curLeader
}

func waitForState(node *Node, expectedState State) State {
	curState := FOLLOWER
	timeout := time.Now().Add(5 * time.Second)
	for time.Now().Before(timeout) {
		curState = node.State()
		if curState == expectedState {
			return expectedState
		}
		time.Sleep(10 * time.Millisecond)
	}
	return curState
}
