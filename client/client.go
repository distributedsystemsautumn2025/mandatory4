package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	// "context"
	// "time"
	pb "mandatory4/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Queue struct {
	Elements []*pb.Message
	Size     int
}
type node struct {
	pb.UnimplementedRicartAgrawalaServiceServer
	pb.RicartAgrawalaServiceClient
	id        string
	clock     int64
	state     string
	replies   [3]*pb.Message
	id2client map[string]pb.RicartAgrawalaServiceClient
	queue     Queue
}

var idToPort = map[string]string{
	"1": "localhost:9000",
	"2": "localhost:9001",
	"3": "localhost:9002",
}

func main() {
	//setup logFile
	logFile, err := os.OpenFile("../RicartAgrawala.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	//here, we let clients define their id at instantiation in terminal
	//when starting the program, run : "go run client.go --id='ID'"
	//port := flag.String("port", "", "listen on this port")

	id := flag.String("id", "", "node ID")
	flag.Parse()

	if *id == "" {
		log.Fatalf("Usage: go run client.go --id='id'")
	}

	nodeAddress, ok := idToPort[*id]
	if !ok {
		log.Fatalf("Did not find address for id=%s", *id)
	}
	node := &node{
		id:        *id,
		clock:     0,
		state:     "RELEASED",
		replies:   [3]*pb.Message{},
		queue:     Queue{Size: 3},
		id2client: map[string]pb.RicartAgrawalaServiceClient{},
	}
	// listen
	lis, err := net.Listen("tcp", nodeAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Node %s listening on %s", *id, nodeAddress)
	// serve
	grpcServer := grpc.NewServer()
	pb.RegisterRicartAgrawalaServiceServer(grpcServer, node)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	log.Printf("Node %s is serving on %s", *id, nodeAddress)

	// do node stuff i.e.: listen for CLI-args

	node.ConnectToPeers()
	go func() {
		for {
			_, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				log.Printf("Error reading CLI: %v", err)
			}
			node.Requester()
			fmt.Println("sending request...")

		}
	}()
	//graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// blocks code below, till signal is received
	<-signalChan
	log.Println("received termination signal, shutting down gracefully ...")
	fmt.Println("received termination signal, shutting down gracefully ...")

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

}

// here we attempt to connect to other nodes
func (n *node) ConnectToPeers() {
	n.id2client = make(map[string]pb.RicartAgrawalaServiceClient)
	for peerID, addr := range idToPort {
		//check that you don't attempt to connect to yourself:
		if peerID == n.id {
			continue
		}
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("failed to connect to peer: %s \n at port: %s", peerID, addr)
		}

		client := pb.NewRicartAgrawalaServiceClient(conn)
		n.id2client[peerID] = client
		log.Printf("Node %s connected to peer %s \n at %s", n.id, peerID, addr)
	}
}

// RequestReply is the server-side implementation of our grpc service
func (node *node) RequestAndReply(ctx context.Context, request *pb.Message) (*pb.Message, error) {
	log.Printf("Node %s received an access request from node %s at time: %d", node.id, request.Sender, request.LogicalTime)
	return node.RequestHandler(request), nil
}

// RequestHandler handles incoming requests by using the Ricart-Agrawala algorithm
func (node *node) RequestHandler(request *pb.Message) *pb.Message {
	if request.Sender == node.id {

	}
	if node.state == "HELD" || (node.state == "WANTED" && node.clock < request.LogicalTime) {
		node.queue.Enqueue(request)
	} else if node.state == "WANTED" && node.clock == request.LogicalTime {
		if node.id < request.Sender {
			node.queue.Enqueue(request)
		} else {
			return node.Reply()
		}
	}
	return node.Reply()
}

// Requester sends an access request out to all other nodes
func (node *node) Requester() {
	request := &pb.Message{
		Sender:         node.id,
		RequestOrReply: "REQUEST",
		LogicalTime:    node.clock,
	}
	// some clock stuff
	node.state = "WANTED"
	for clientID, client := range node.id2client {
		go func(clientID string, client pb.RicartAgrawalaServiceClient) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			reply, err := client.RequestAndReply(ctx, request)
			if err != nil {
				log.Printf("Failed to request reply: %v", err)
			}
			log.Printf("Node %s is received a reply from client %s at time: %d", node.id, clientID, reply.LogicalTime)
		}(clientID, client)
	}
	// check n-1 replies
	if node.AccessGranted() {
		node.CriticalSection()
	}
}

// AccessGranted checks if there are sufficient replies for this node to enter the Critical Section
func (node *node) AccessGranted() bool {
	// we realise this is not a scalable solution
	switch node.id {
	case "1":
		if node.replies[1] != nil && node.replies[2] != nil {
			return true
		}
	case "2":
		if node.replies[0] != nil && node.replies[2] == nil {
			return true
		}
	case "3":
		if node.replies[0] != nil && node.replies[1] == nil {
			return true
		}
	}
	return false
}
func (node *node) CriticalSection() {
	node.state = "HELD"
	log.Printf("%s is now in the Critical Section", node.id)
	node.LeaveCriticalSection()
}
func (node *node) LeaveCriticalSection() {
	node.state = "RELEASED"
	log.Printf("%s has left the Critical Section", node.id)
	for !node.queue.IsEmpty() {
		//request := node.queue.Dequeue()
		//node.Reply()
	}
}

func (node *node) Reply() *pb.Message {
	reply := pb.Message{
		Sender:         node.id,
		RequestOrReply: "REPLY",
		LogicalTime:    node.clock,
	}
	return &reply
}

// queue stuff
func (q *Queue) Enqueue(elem *pb.Message) {
	if q.GetLength() == q.Size {
		fmt.Println("Overflow")
		return
	}
	q.Elements = append(q.Elements, elem)
}

func (q *Queue) Dequeue() *pb.Message {
	if q.IsEmpty() {
		fmt.Println("Queue is empty")
		return nil
	}
	element := q.Elements[0]
	if q.GetLength() == 1 {
		q.Elements = nil
		return element
	}
	q.Elements = q.Elements[1:]
	return element // Slice off the element once it is dequeued.
}
func (q *Queue) GetLength() int {
	return len(q.Elements)
}

func (q *Queue) IsEmpty() bool {
	return len(q.Elements) == 0
}
