package main

import (
    "container/heap"
    "flag"
    "fmt"
    "math/rand"
    "net"
    "net/http"
    "net/rpc"
    "sync"
    "time"
)

var clientMutex sync.Mutex
var clientTimeStamp int64 = rand.Int63n(50)
var MsgQ = &MessageQ{}
var serverRPCAddr  string
var clientRPCAddrs map[int]string
var clientNames    map[int]string
var requestReply   map[int]bool

type ClientRPCService struct {

}

func (*ClientRPCService) SendLamportMessage(req LamportMessage, res *string) error {
    clientMutex.Lock()
    defer clientMutex.Unlock()
    fmt.Printf("[%s] Received message: %s \n",time.Now(), req.String())
    if req.TS > clientTimeStamp {
        clientTimeStamp = req.TS
    }
    clientTimeStamp++

    if req.MsgType == Request {
        heap.Push(MsgQ, *NewRequest(req.TS, req.Sender, req.Receiver))
    } else if req.MsgType == Reply {
        requestReply[req.Sender] = true
    } else if req.MsgType == Release {
        removeMessageInQ(MsgQ, req.Sender)
    } else {
        fmt.Println("Fatal Error occurred")
    }
    return nil
}

func logSendMessage(req LamportMessage) {
    fmt.Printf("[%s] Send message: %s \n",time.Now(), req.String())
}

func isAllClientReplied(h *MessageQ, sender int) bool {
    allReplied := true
    for i, m := range requestReply{
        if  (sender != i) && (m != true) {
            allReplied = false
        }
    }
    return allReplied
}

func resetClientReplied()  {
    for i,_ := range requestReply{
        requestReply[i] = false
    }
}
func removeMessageInQ(h *MessageQ, sender int) {
    index := 0
    for i, m := range *h{
        if m.Sender == sender {
            index = i
        }
    }
    old := *h
    *h = append(old[:index], old[index+1:]...)
}

func isSelfRequestInQ(id int) bool {
    for _, m := range *MsgQ{
        if m.Sender == id {
            return true
        }
    }
    return false
}

func client(clientID int)  {
    clientTimeStamp = 0//int64(rand.Intn(50))
    fmt.Printf("%d init random timestamp is %d\n", clientID, clientTimeStamp)
    var connectedToServer = false
    var connectedToOtherClient = make(map[int]bool)

    var RPCToServer *rpc.Client = nil
    var RPCToOtherClient = make(map[int]*rpc.Client)

    for true {
        if !connectedToServer {
            RPCToServer_, err := rpc.DialHTTP("tcp", serverRPCAddr)
            if err != nil {
                fmt.Println("Dial to Server failed:", err)
                time.Sleep(3 * time.Second)
            } else {
                connectedToServer = true
                RPCToServer = RPCToServer_
            }
        }

        var connectedToAllClients bool = true
        for i, a := range clientRPCAddrs{
            if i != clientID {
                if !connectedToOtherClient[i] {
                    RPCToCient, err := rpc.DialHTTP("tcp", a)
                    if err != nil {
                        connectedToAllClients = false
                        fmt.Println("Dial to Client failed", err)
                    } else {
                        RPCToOtherClient[i] = RPCToCient
                        connectedToOtherClient[i] = true
                    }
                }
            } else {
                RPCToOtherClient[i] = nil
                connectedToOtherClient[i] = true
            }
        }

        if connectedToServer && connectedToAllClients{
            break
        } else {
            time.Sleep(3 * time.Second)
        }
    }

    selfRequestSendOut := false
    t1 := time.Now().Add(time.Second*(-50))
    for true {
        isRequestSharedResource := isSelfRequestInQ(clientID)
        t2 := time.Now()
        if !isRequestSharedResource && (t2.Sub(t1) > 15) {
            clientMutex.Lock()
            clientTimeStamp++
            currentTS := clientTimeStamp
            heap.Push(MsgQ, *NewRequest(currentTS, clientID, clientID))
            clientMutex.Unlock()
            selfRequestSendOut = false
            t1 = t2
        } else {

        }

        var msg *LamportMessage = nil
        clientMutex.Lock()
        sizeOfMsgQ := MsgQ.Len()
        if sizeOfMsgQ > 0 {
            minIndex := 0
            var minValue int64 = -1
            for i,v := range *MsgQ{
                if minValue == -1 {
                    minValue = v.TS
                    minIndex = i
                }
                if v.TS <= minValue {
                    minValue = v.TS
                    minIndex = i
                }
            }
            msg = &LamportMessage{
                MsgType:  (*MsgQ)[minIndex].MsgType,
                TS:       (*MsgQ)[minIndex].TS,
                Sender:   (*MsgQ)[minIndex].Sender,
                Receiver: (*MsgQ)[minIndex].Receiver,
                SendOut:  (*MsgQ)[minIndex].SendOut,
                Replied:  (*MsgQ)[minIndex].Replied,
            }
        }
        clientMutex.Unlock()
        if msg == nil {
            time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
            continue
        }
        if msg.Sender == clientID {
            if !selfRequestSendOut {
                for i, v := range RPCToOtherClient{
                    if i != clientID {
                        msg.Receiver = i
                        logSendMessage(*msg)
                        v.Call("ClientRPCService.SendLamportMessage", *msg, nil)
                    }
                }
                selfRequestSendOut = true
            } else {
                clientMutex.Lock()
                allReplied := isAllClientReplied(MsgQ, clientID)
                clientMutex.Unlock()
                if allReplied {
                    RPCToServer.Call("SharedResourceService.RequestSharedResource", clientNames[clientID], nil)
                    time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
                    RPCToServer.Call("SharedResourceService.ReleaseSharedResource", clientNames[clientID], nil)
                    clientMutex.Lock()
                    resetClientReplied()
                    clientTimeStamp++
                    currentTS := clientTimeStamp
                    removeMessageInQ(MsgQ, clientID)
                    clientMutex.Unlock()
                    for i, v := range RPCToOtherClient{
                        msgReply := &LamportMessage{
                            MsgType:  Release,
                            TS:       currentTS,
                            Sender:   clientID,
                            Receiver: i,
                        }
                        if i != clientID {
                            logSendMessage(*msgReply)
                            v.Call("ClientRPCService.SendLamportMessage", *msgReply, nil)
                        }
                    }
                }
            }
        } else {
            if !msg.Replied {
                clientMutex.Lock()
                clientTimeStamp++
                currentTS := clientTimeStamp
                for i,v := range *MsgQ{
                    if v.Sender == msg.Sender {
                        (*MsgQ)[i].Replied = true
                        break
                    }
                }
                clientMutex.Unlock()
                msgReply := &LamportMessage{
                    MsgType:  Reply,
                    TS:       currentTS,
                    Sender:   clientID,
                    Receiver: msg.Sender,
                }
                logSendMessage(*msgReply)
                RPCToOtherClient[msg.Sender].Call("ClientRPCService.SendLamportMessage", *msgReply, nil)
            }
        }
        time.Sleep(100 * time.Millisecond)
    }
}

func main() {
    heap.Init(MsgQ)
    totalClientNum := flag.Int("c", 3, "total client number")
    clientIndex := flag.Int("i", 2, "client index")
    rpcListenPortStart := flag.Int("p", 8090, "RPC service port start from")
    serverAddr := flag.String("s", "127.0.0.1:8080", "shared resource server address")
    flag.Parse()
    ip, err := getClientIp()
    if err != nil {
        fmt.Println("can not obtain ip")
        fmt.Println(err)
        return
    }
    serverRPCAddr  = *serverAddr
    requestReply   = make(map[int]bool)
    clientRPCAddrs = make(map[int]string)
    clientNames    = make(map[int]string)
    resetClientReplied()
    for i := 0; i < *totalClientNum; i++ {
        name := fmt.Sprintf("Client-%d", i)
        addr := fmt.Sprintf("%s:%d", ip, (*rpcListenPortStart)+i)
        clientRPCAddrs[i] = addr
        clientNames[i] = name
    }

    rpc.Register(new(ClientRPCService))
    rpc.HandleHTTP()
    l, e := net.Listen("tcp", clientRPCAddrs[*clientIndex])
    if e != nil {
        fmt.Println("listen error:", e)
        return
    }
    go client(*clientIndex)
    http.Serve(l, nil)
}
