package raft

type AppendEntriesArgs struct {
    Term             int
    LeaderId         int
    PreLogIndex      int
    PreLogTerm       int
    Entries          []Entry
    LeaderCommit     int
}

type AppendEntriesReply struct {
    Term    int
    Success bool
}
/*
 两种日志：一是选举为leader后发空日志告知其他server,阻止其他server继续变为candidate
 二是 client发送了entries后的处理
*/
func (rf *Raft) appendEntries(heartbeat bool) {
    // lastLog := rf.log.lastLog()
    for peer, _ := range rf.peers {
        if peer == rf.me {
            rf.resetElectionTimer()
            continue
        }
        // rules for leader 3
        if heartbeat {
            args := AppendEntriesArgs {
                Term:   rf.currentTerm,
                LeaderId : rf.me,
            }
            go rf.leaderSendEntries(peer, &args)
        }
    }
}

func (rf *Raft) leaderSendEntries(serverId int, args *AppendEntriesArgs) {
    DPrintf("[%v]: %v send append entry rpc start!\n", rf.me, serverId)
    reply := AppendEntriesReply{}
    ok := rf.sendAppendEntries(serverId, args, &reply)
    DPrintf("[%v]: %v send append entry rpc end, then to process this AE reply!\n", rf.me, serverId)
    if !ok {
        return
    }
    rf.mu.Lock()
    defer rf.mu.Unlock()
    // 有更大的任期号,说明有任期更大的leader，则说明me竞选失败，自动变为follower
    if reply.Term > rf.currentTerm {
        rf.setNewTerm(reply.Term)
        return
    } else if reply.Term == rf.currentTerm {
        if reply.Success {
            DPrintf("[%v]: %v append entry success!\n", rf.me, serverId)
        } else {
            DPrintf("[%v]: %v append entry failed!\n", rf.me, serverId)
        }
    }
}

func (rf *Raft) sendAppendEntries(serverId int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
    ok := rf.peers[serverId].Call("Raft.AppendEntries", args, reply)
    return ok
}

// server收到AE RPC的处理逻辑 AppendEntriesResp 这个函数rpc流程不能识别，要用AppendEntries
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
    DPrintf("[%v]: recv append entry rpc\n", rf.me)
    rf.mu.Lock()
    defer rf.mu.Unlock()

    reply.Success = false
    reply.Term = rf.currentTerm
    if args.Term < rf.currentTerm {
        return
    } else if args.Term > rf.currentTerm { // 正常在request vote rpc
        rf.setNewTerm(args.Term)
        return
    }

    rf.resetElectionTimer()

    // candidate rules 3
    if rf.state == Candidate {
        rf.state = Follower
    }

    reply.Success = true
}




