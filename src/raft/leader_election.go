package raft

import (
    "sync"
    "math/rand"
    "time"
)

func (rf *Raft) resetElectionTimer() {
    t := time.Now()
    electionTimeout := time.Duration(150 + rand.Intn(150)) * time.Millisecond
    rf.electionTime = t.Add(electionTimeout) // 更新leader election超时定时器的起始时间
}

func (rf *Raft) setNewTerm(term int) {
    if term > rf.currentTerm || rf.currentTerm == 0 {
        rf.state = Follower
        rf.currentTerm = term
        rf.votedFor = -1
        DPrintf("[%d]: set term %v\n", rf.me, rf.currentTerm)
    }
}

func (rf *Raft) leaderElection() {
    rf.currentTerm++
    rf.state = Candidate
    rf.votedFor = rf.me
    rf.resetElectionTimer() // 每次切换为candidate/收到AE/收到RequeVote时都有重置选举超时定时器

    // 准备发RequestVote RPC
    voteCnter := 1
    lastLog := rf.log.lastLog()
    DPrintf("[%v]: start leader election, term %d\n", rf.me, rf.currentTerm)
    args := RequestVoteArgs {
        Term :  rf.currentTerm,
        CandidateId : rf.me,
        LastLogIndex : lastLog.Index,
        LastLogTerm :  lastLog.Term,
    }

    var becomeLeader sync.Once
    for serverId, _ := range rf.peers {
        if serverId == rf.me {
            continue
        }
        go rf.candidateRequestVote(serverId, &args, &voteCnter, &becomeLeader)
    }
}


