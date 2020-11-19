package main

import (
	"errors"
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
var recordAccess map[string]int

type SharedResourceService struct { }

func (*SharedResourceService) RequestSharedResource(req string, res *string) error {
	serverMutex.Lock()
	defer serverMutex.Unlock()
	fmt.Println(req, " is requesting shared resource")

	recordAccess[req] = recordAccess[req]+1
	for i:=0; i < 5; i++ {
		fmt.Printf("      %d : %d\n", i, recordAccess[fmt.Sprintf("%d",i)])
	}

	if usingSharedResource {
		s := sharedResourceUser + "is using shared resource"
		fmt.Println(sharedResourceUser, " is using shared resource")
		fmt.Println("Fatal Error occurred when request resource, Please check your Lamport Algorithm!!!!")
		return errors.New(s)
	}
	sharedResourceUser = req
	usingSharedResource = true
	fmt.Println(req, " is using shared resource")
	return nil
}

func (*SharedResourceService) ReleaseSharedResource(req string, res *string) error {
	serverMutex.Lock()
	defer serverMutex.Unlock()
	fmt.Println(req, " is releasing shared resource")
	if !usingSharedResource {
		fmt.Println("Fatal Error occurred when releasing resource, Please check your Lamport Algorithm!!!!")
		return errors.New("no user is using this shared resource")
	}
	sharedResourceUser = ""
	usingSharedResource = false
	fmt.Println(req, " released shared resource")
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

	recordAccess = make(map[string]int)
	for i := 0; i < 5; i++ {
		recordAccess[fmt.Sprintf("%d",i)] = 0
	}
	rpc.Register(new(SharedResourceService))
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":" + (*p))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}

