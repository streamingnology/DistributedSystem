package main

import (
    "container/heap"
    "flag"
    "fmt"
    "net"
    "net/http"
    "net/rpc"
    "time"
)

type ClientRPCService struct {

}

func (*ClientRPCService) SendLamportMessage(req RaymondMessage) error {
    fmt.Printf("[%s] Received message: %s \n",time.Now(), req.String())

    return nil
}


func main() {
    totalClientNum := flag.Int("c", 3, "total client number")
    clientIndex := flag.Int("i", 2, "client index")
    rpcListenPortStart := flag.Int("p", 8090, "RPC service port start from")
    serverAddr := flag.String("s", "127.0.0.1:8080", "shared resource server address")
    flag.Parse()
    
    rpc.Register(new(ClientRPCService))
    rpc.HandleHTTP()
    l, e := net.Listen("tcp", clientRPCAddrs[*clientIndex])
    if e != nil {
        fmt.Println("listen error:", e)
        return
    }

    http.Serve(l, nil)
}
