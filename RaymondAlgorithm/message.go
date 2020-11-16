package main

import "fmt"

const (
    Request int = iota + 1
    Reply
)

type RaymondMessage struct {
    MsgType     int
    Sender      int
}

func NewRequestMessage(sender int) RaymondMessage {
    return RaymondMessage{
        MsgType: Request,
        Sender: sender,
    }
}

func NewReplyMessage(sender int) RaymondMessage {
    return RaymondMessage{
        MsgType: Reply,
        Sender: sender,
    }
}

func (m RaymondMessage)String() string {
    msgType := ""
    switch m.MsgType {
    case Request:
        msgType = "Request"
    case Reply:
        msgType = "Reply"
    }
    return fmt.Sprintf("[%s from Sender %d]", msgType, m.Sender)
}