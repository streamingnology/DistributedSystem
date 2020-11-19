# DistributedSystem
Lamport Algorithm: source code is in the lamport folder.
compile server: go build server.go utils.go
compile client: go build client.go message.go messagequeue.go utils.go
run client:
client.exe -c 5 -i 0
client.exe -c 5 -i 1
client.exe -c 5 -i 2
client.exe -c 5 -i 3
client.exe -c 5 -i 4
run server:
server.exe

Raymond Algorithm: source code is in RaymondAlgorithm folder.
compile server: go build server.go utils.go
compile client: go build client.go message.go utils.go
run client:
client.exe -i 0
client.exe -i 1
client.exe -i 2
client.exe -i 3
client.exe -i 4
run server:
server.exe
