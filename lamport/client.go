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
var clientTimeStamp int64
var MsgQ = &MessageQ{}
var serverRPCAddr  string
var clientRPCAddrs map[int]string
var requestReply   map[int]bool

type ClientRPCService struct { }

func (*ClientRPCService) SendLamportMessage(req LamportMessage, res *string) error {
    clientMutex.Lock()
    defer clientMutex.Unlock()
    fmt.Printf("[%s] Received message: %s \n",time.Now().Format("15:04:05.000"), req.String())
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
    fmt.Printf("[%s] Send message: %s \n",time.Now().Format("15:04:05.000"), req.String())
}

func isAllClientReplied() bool {
    allReplied := true
    for _, m := range requestReply{
        if m != true {
            allReplied = false
            break
        }
    }
    return allReplied
}

func resetClientReplied(sender int)  {
    for i,_ := range requestReply{
        requestReply[i] = false
        if i == sender {
            requestReply[i] = true
        }
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
    rand.Seed(time.Now().UnixNano())
    clientTimeStamp = int64(rand.Intn(100)) //int64(rand.Intn(50))
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
        clientMutex.Lock()
        isRequestSharedResource := isSelfRequestInQ(clientID)
        clientMutex.Unlock()
        t2 := time.Now()
        if !isRequestSharedResource && (t2.Sub(t1) > 15) {
            clientMutex.Lock()
            workaroundSameTS := false
            clientTimeStamp++
            currentTS := clientTimeStamp
            for _,v := range *MsgQ{
                if v.TS == clientTimeStamp {
                    workaroundSameTS = true
                    break
                }
            }
            if !workaroundSameTS {
                heap.Push(MsgQ, *NewRequest(currentTS, clientID, clientID))
            } else {
                clientTimeStamp--
            }
            clientMutex.Unlock()
            if !workaroundSameTS {
                selfRequestSendOut = false
            } else {
                time.Sleep(1*time.Second)
            }

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
                if v.TS < minValue {
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
                allReplied := isAllClientReplied()
                clientMutex.Unlock()
                if allReplied {
                    fmt.Printf("[%s] %d enter critial section\n", time.Now().Format("15:04:05.000"), clientID)
                    if RPCToServer != nil {
                        e := RPCToServer.Call("SharedResourceService.RequestSharedResource", fmt.Sprintf("%d", clientID), nil)
                        if e != nil {
                            fmt.Println(e)
                        }
                        time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
                        e = RPCToServer.Call("SharedResourceService.ReleaseSharedResource", fmt.Sprintf("%d", clientID), nil)
                        if e != nil {
                            fmt.Println(e)
                        }
                    } else {
                        fmt.Println("Please Initialize RPC to Server")
                    }
                    fmt.Printf("[%s] %d leave critial section\n", time.Now().Format("15:04:05.000"), clientID)
                    clientMutex.Lock()
                    resetClientReplied(clientID)
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
    totalClientNum := flag.Int("c", 5, "total client number")
    clientIndex := flag.Int("i", 0, "client index")
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

    for i := 0; i < *totalClientNum; i++ {
        addr := fmt.Sprintf("%s:%d", ip, (*rpcListenPortStart)+i)
        clientRPCAddrs[i] = addr
        requestReply[i] = false
    }
    resetClientReplied(*clientIndex)

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
