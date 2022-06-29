package raft

import (
    "sync"
)

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
    Term int
    CandidateId int
    LastLogIndex int
    LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
    Term int
    VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
    rf.mu.Lock()
    defer rf.mu.Unlock()

    if args.Term < rf.currentTerm {
        reply.Term = rf.currentTerm
        reply.VoteGranted = false
        return
    } else if args.Term > rf.currentTerm {
        rf.setNewTerm(args.Term)
    }

    // request vote rpc receiver 2
    myLastLog := rf.log.lastLog()
    upToDate := args.LastLogTerm > myLastLog.Term ||
        (args.LastLogTerm == myLastLog.Term && args.LastLogIndex >= myLastLog.Index)
    if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && upToDate {
        reply.VoteGranted = true
        rf.votedFor = args.CandidateId
        rf.resetElectionTimer() // 投完票,follower重置选举超时timer
        DPrintf("[%v]: term %v vote %v", rf.me, rf.currentTerm, rf.votedFor)
    } else {
        reply.VoteGranted = false
    }
    reply.Term = rf.currentTerm
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) candidateRequestVote(serverId int, args *RequestVoteArgs, voteCnter *int, becomeLeader *sync.Once) {
    // 发送 request vote rpc
    DPrintf("[%d]: term %v send vote request to %d\n", rf.me, args.Term, serverId)
    reply := RequestVoteReply {}
    ok := rf.sendRequestVote(serverId, args, &reply)
    if !ok {
        DPrintf("[%d]: term %v send vote request to %d, failed!\n", rf.me, args.Term, serverId)
        return
    }

    // 接收rv rpc,并处理:有效则计数++,准备成为leader
    rf.mu.Lock()
    defer rf.mu.Unlock()
    if reply.Term > args.Term { // 投票人任期更大,竞选失败
        DPrintf("[%d]: %d 在新的term，更新term: %d，结束\n", rf.me, serverId, reply.Term)
        rf.setNewTerm(reply.Term)
        return
    } else if reply.Term < args.Term {
        DPrintf("[%d]: %d 的term %d 已经失效，结束\n", rf.me, serverId, reply.Term)
        return
    }
    // 投票人当前term和候选人term 相同, 则继续判断voteGranted
    if !reply.VoteGranted {
        DPrintf("[%d]: %d 没有投给me，结束\n", rf.me, serverId)
        return
    }

    // 投票结果检查ok,则将投票数++, 判断是否可以称为leader
    DPrintf("[%d]: from %d term一致，且投给%d\n", rf.me, serverId, rf.me)
    *voteCnter++
    if *voteCnter > len(rf.peers) / 2 &&
        rf.state == Candidate {
        DPrintf("[%d]: 获得多数选票，可以提前结束\n", rf.me)
        becomeLeader.Do(func() {
            DPrintf("[%d]: 当前term %d 结束\n", rf.me, rf.currentTerm)
            rf.state = Leader
            lastLogIndex := rf.log.lastLog().Index
            for i, _ := range rf.peers {
                rf.nextIndex[i] = lastLogIndex + 1
                rf.matchIndex[i] = 0
            }
            DPrintf("[%d]: leader - nextIndex %#v", rf.me, rf.nextIndex)
            rf.appendEntries(true)
        })
    }

}
