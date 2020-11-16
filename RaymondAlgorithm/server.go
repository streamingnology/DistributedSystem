package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
)

var serverMutex sync.Mutex
var usingSharedResource bool = false
var sharedResourceUser  string

type SharedResourceService struct {

}

func (*SharedResourceService) RequestSharedResource(req string, res *string) error {
	fmt.Println(req, " is requesting shared resource")
	serverMutex.Lock()
	if usingSharedResource {
		fmt.Println(sharedResourceUser, " is using shared resource")
		fmt.Println("Fatal Error occurred when request resource, Please check your Lamport Algorithm!!!!")
		return nil
	}
	sharedResourceUser = req
	usingSharedResource = true
	fmt.Println(req, " is using shared resource")
	serverMutex.Unlock()
	return nil
}

func (*SharedResourceService) ReleaseSharedResource(req string, res *string) error {
	fmt.Println(req, " is releasing shared resource")
	serverMutex.Lock()
	if !usingSharedResource {
		fmt.Println("Fatal Error occurred when releasing resource, Please check your Lamport Algorithm!!!!")
		return nil
	}
	sharedResourceUser = ""
	usingSharedResource = false
	fmt.Println(req, " released shared resource")
	serverMutex.Unlock()
	return nil
}

func main() {
	var p = flag.String("p", "8080", "shared resource server listen port")
	flag.Parse()
	ip, err := getClientIp()
	if err != nil {
		fmt.Println("can not obtain ip")
		fmt.Println(err)
		return
	}
	fmt.Printf("server address is %s:%s \n", ip, *p)

	rpc.Register(new(SharedResourceService))
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":" + (*p))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}

