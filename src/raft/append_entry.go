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
    lastLog := rf.log.lastLog()
    for peer, _ := range rf.peers {
        if peer == rf.me {
            rf.resetElectionTimer()
            continue
        }
        // rules for leader 3
        nextIndex := rf.nextIndex[peer]
        if lastLog.Index >= nextIndex || heartbeat {
            preLog := rf.log.at(nextIndex - 1) // rf.nextIndex[]中记录本term内所有follower已追加的日志信息
            args := AppendEntriesArgs {
                Term:       rf.currentTerm,
                LeaderId:   rf.me,
                PreLogIndex:preLog.Index,
                PreLogTerm: preLog.Term,
                Entries:    make([]Entry, lastLog.Index - nextIndex + 1),// AE日志长度为上次匹配的至leader最新的日志长度
                LeaderCommit: rf.commitIndex,
            }
            copy(args.Entries, rf.log.slice(nextIndex)) // 拷贝leader的日志条目到entres中
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
            match := args.PreLogIndex + len(args.Entries)
            next := match + 1
            rf.nextIndex[serverId] = max(rf.nextIndex[serverId], next)
            rf.matchIndex[serverId] = max(rf.matchIndex[serverId], match)
            DPrintf("[%v]: %v append entry success! next %v, match %v\n",
                rf.me, serverId, rf.nextIndex[serverId], rf.matchIndex[serverId])
        } else {
            if rf.nextIndex[serverId] > 1 {
                rf.nextIndex[serverId]--
            }
            DPrintf("[%v]: %v append entry failed!\n", rf.me, serverId)
        }
    }
}

func (rf *Raft) sendAppendEntries(serverId int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
    ok := rf.peers[serverId].Call("Raft.AppendEntries", args, reply)
    return ok
}

// server收到AE RPC的处理逻辑 AppendEntrie 这个函数rpc流程不能识别，主要原因是AppendEntries这个rpc接受函数不带返回值，而我写了bool返回类型
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

    for idx, entry := range args.Entries {
        // AE RPC rule 3:if an existing entry conflicts with a new one(same idx but diff term),del the existing entry and all that follow it
        if entry.Index <= rf.log.lastLog().Index && rf.log.at(entry.Index).Term != entry.Term {
            rf.log.truncate(entry.Index)
        }
        // AE RPC rule 4:append any new entries not already in the log
        if entry.Index > rf.log.lastLog().Index {
            rf.log.append(args.Entries[idx:]...)
            DPrintf("[%d]: append entry: follower append [%v]\n", rf.me, args.Entries[idx:])
            break
        }
    }

    // AE RPC rule 5:
    if args.LeaderCommit > rf.commitIndex {
        rf.commitIndex = min(args.LeaderCommit, rf.log.lastLog().Index)
    }
    reply.Success = true
}




