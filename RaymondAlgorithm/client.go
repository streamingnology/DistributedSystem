package main

import (
    "flag"
    "fmt"
    "math/rand"
    "net"
    "net/http"
    "net/rpc"
    "sync"
    "time"
)

var mutex             sync.Mutex

var requestQ []int
var requestStatusQ []bool // Is request send out or executed?

var serverRPCAddr     string
var clientRPCAddrs    map[int]string = make(map[int]string)
var rootOfThisClient  int = -1
var neigborClientIndexs []int

var RPCToServer *rpc.Client = nil
var RPCToNeigbors = make(map[int]*rpc.Client)

type ClientRPCService struct { }

func (*ClientRPCService) SendRaymondMessage(req RaymondMessage, rep *string) error {
    mutex.Lock()
    defer mutex.Unlock()
    fmt.Printf("[%s] Received message: %s \n",time.Now(), req.String())
    if req.MsgType == Request {
        requestQ =  append(requestQ, req.Sender)
        requestStatusQ = append(requestStatusQ, false)
    } else if req.MsgType == Reply {
        rootOfThisClient = -1
    } else {
        fmt.Println("Fatal Error Occurred, Invalid reply message type")
    }
    return nil
}

func initConnect()  {
    var connectedToServer = false
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

        var connectedToAllNeigbors bool = true
        for _, v := range neigborClientIndexs{
            if RPCToNeigbors[v] == nil {
                RPCToCient, err := rpc.DialHTTP("tcp", clientRPCAddrs[v])
                if err != nil {
                    connectedToAllNeigbors = false
                    fmt.Println("Dial to Client failed", err)
                } else {
                    RPCToNeigbors[v] = RPCToCient
                }
            }
        }

        if connectedToServer && connectedToAllNeigbors {
            break
        } else {
            time.Sleep(3 * time.Second)
        }
    }
}

func initTreeGraph(clientIndex int) {
    if clientIndex == 0 {
        rootOfThisClient = -1
        neigborClientIndexs = append(neigborClientIndexs, 1)
    } else if clientIndex == 1 {
        rootOfThisClient = 0
        neigborClientIndexs = append(neigborClientIndexs, 0)
        neigborClientIndexs = append(neigborClientIndexs, 2)
    } else if clientIndex == 2 {
        rootOfThisClient = 1
        neigborClientIndexs = append(neigborClientIndexs, 1)
        neigborClientIndexs = append(neigborClientIndexs, 3)
        neigborClientIndexs = append(neigborClientIndexs, 4)
    } else if clientIndex == 3 {
        rootOfThisClient = 2
        neigborClientIndexs = append(neigborClientIndexs, 2)
    } else if clientIndex == 4 {
        rootOfThisClient = 2
        neigborClientIndexs = append(neigborClientIndexs, 2)
    } else {
        fmt.Printf("Invalid client index %d\n", clientIndex)
    }
}

func client(clientIndex int)  {
    initTreeGraph(clientIndex)
    initConnect()

    t1 := time.Now().Add(time.Second * (-50))
    for true {
        t2 := time.Now()
        mutex.Lock()
        isSelfRequestInQ := false
        for _, v := range requestQ{
            if v == clientIndex {
                isSelfRequestInQ = true
                break
            }
        }
        if !isSelfRequestInQ && (t2.Sub(t1) > 15) {
            requestQ = append(requestQ, clientIndex)
            requestStatusQ = append(requestStatusQ, false)
        }
        var topRequest = -1
        var topRequestStatus = true
        if len(requestQ) > 0 {
            topRequest = requestQ[0]
            topRequestStatus = requestStatusQ[0]
        }
        mutex.Unlock()
        if topRequest < 0 {
            // NO REQUEST HERE
            time.Sleep(100 * time.Millisecond)
            continue
        }
        if rootOfThisClient < 0 { // THIS NODE is ROOT of TREE
            if topRequest == clientIndex {
                // TODO: USE SHARED RESOURCE
                RPCToServer.Call("SharedResourceService.RequestSharedResource", fmt.Sprintf("%d", clientIndex), nil)
                time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
                RPCToServer.Call("SharedResourceService.ReleaseSharedResource", fmt.Sprintf("%d", clientIndex), nil)
                mutex.Lock()
                requestQ = requestQ[1:]
                requestStatusQ = requestStatusQ[1:]
                mutex.Unlock()
            } else {
                // TODO: RELAY THIS LOCK TO REQUESTER
                msg := NewReplyMessage(clientIndex)
                RPCToNeigbors[topRequest].Call("ClientRPCService.SendRaymondMessage", msg, nil)
                mutex.Lock()
                rootOfThisClient = topRequest
                requestQ = requestQ[1:]
                requestStatusQ = requestStatusQ[1:]
                mutex.Unlock()
            }

        } else { // THIS NODE IS NOT ROOT OF TREE
            if topRequestStatus {
                // THIS REQUEST HAS ALREADY SEND OUT, WAITE
            } else {
                msg := NewRequestMessage(clientIndex)
                RPCToNeigbors[rootOfThisClient].Call("ClientRPCService.SendRaymondMessage", msg, nil)
                mutex.Lock()
                requestStatusQ[0] = true
                mutex.Unlock()
            }
        }
    }
}

func main() {
    totalClientNum := flag.Int("c", 5, "total client number")
    clientIndex := flag.Int("i", 1, "client index")
    rpcListenPortStart := flag.Int("p", 8090, "RPC service port start from")
    serverAddr := flag.String("s", "127.0.0.1:8080", "shared resource server address")
    flag.Parse()

    ip, err := getClientIp()
    if err != nil {
        fmt.Println("can not obtain ip")
        fmt.Println(err)
        return
    }
    for i := 0; i < *totalClientNum; i++ {
        addr := fmt.Sprintf("%s:%d", ip, (*rpcListenPortStart)+i)
        clientRPCAddrs[i] = addr
    }
    serverRPCAddr = *serverAddr

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
