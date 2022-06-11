package raft

import (
    "sync"
    "math/rand"
    "time"
)

func (rf *Raft) resetElectionTimer() {
    t := time.Now()
    electionTimeout := time.Duration(150 + rand.Intn(150)) * time.Millisecond
    rf.electionTime = t.Add(electionTimeout)
}

func (rf *Raft) leaderElection() {
    rf.currentTerm++
    rf.state = Candidate
    rf.voteFor = rf.me
    rf.resetElectionTimer() // 每次切换为candidate/收到AE/收到RequeVote时都有重置选举超时定时器

    // 准备发RequestVote RPC
     voteCnter := 1
    // LastLog := rf.log.lastLog()
    DPrintf("[%v]: start leader election, term %d\n", rf.me, rf.currentTerm)
    args := RequestVoteArgs {
        Term :  rf.currentTerm,
        CandidatedId : rf.me,
        // LastLogIndex : ?
        // LastLogTerm : ?
    }

    var becomeLeader sync.Once
    for serverId, _ := range rf.peers {
        if serverId == rf.me {
            continue
        }
        go rf.candidateRequestVote(serverId, &args, &voteCnter, &becomeLeader)
    }
}

func (rf *Raft) candidateRequestVote(serverId int, args *RequestVoteArgs, voteCnter *int, becomeLeader *sync.Once) {

}


