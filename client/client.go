package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	// "context"
	// "time"
	pb "mandatory4/grpc"
)

type node struct {
	pb.UnimplementedRicartAgrawalaServiceServer
	peers map[string]chan *pb.AccessRequest
	clock int64
	state string
}

func main() {
	//setup logFile
	logFile, err := os.OpenFile("../RicartAgrawala.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// lis, err := net.Listen("tcp", "0.0.0.0:9000")
	// if err != nil {
	// 	log.Fatalf("failed to listen: %v", err)
	// }
	// log.Printf("The server is listening at %v", lis.Addr())

	//graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// blocks code below, till signal is received
	<-signalChan
	log.Println("received termination signal, shutting down gracefully ...")
	fmt.Println("received termination signal, shutting down gracefully ...")

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

}
func (n *node) multicast(msg *pb.AccessRequest) {
	for peer := range n.peers {
		n.peers[peer] <- msg
	}

}
