package main

import (
"fmt"
)

const (
    Request int = iota + 1 // request mutual lock
    Reply                  // reply mutual lock
    Release                // release mutual lock
)

type LamportMessage struct {
    MsgType    int
    TS         int64
    Sender     int
    Receiver   int
    SendOut    bool
    Replied    bool
}

func NewRequest(ts int64, sender int, receiver int) *LamportMessage {
    return &LamportMessage{
        MsgType:    Request,
        TS:         ts,
        Sender:     sender,
        Receiver:   receiver,
    }
}

func NewReply(ts int64, sender int, receiver int) *LamportMessage {
    return &LamportMessage{
        TS:         ts,
        MsgType:    Reply,
        Sender:     sender,
        Receiver:   receiver,
    }
}

func NewRelease(ts int64, sender int, receiver int) *LamportMessage {
    return &LamportMessage{
        MsgType:    Release,
        TS:         ts,
        Sender:     sender,
        Receiver:   receiver,
    }
}

func (m *LamportMessage) String() string {
    var name string
    var replyStatus string
    switch m.MsgType {
    case Request:
        name = "Request"
        if m.Replied {
            replyStatus = "Replied"
        } else {
            replyStatus = "Not Replied"
        }
    case Release:
        name = "Release"
    case Reply:
        name = "Reply"
    }
    return fmt.Sprintf("[%s %d %d %d %s]", name,  m.TS, m.Sender, m.Receiver, replyStatus)
}
