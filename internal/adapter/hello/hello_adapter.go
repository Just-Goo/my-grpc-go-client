package hello

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/Just-Goo/my-grpc-go-client/internal/port"
	"github.com/Just-Goo/my-grpc-proto/protogen/go/hello"
	"google.golang.org/grpc"
)

type HelloAdapter struct {
	helloClient port.HelloClientPort
}

func NewHelloAdapter(conn *grpc.ClientConn) (*HelloAdapter, error) {
	client := hello.NewHelloServiceClient(conn)

	return &HelloAdapter{
		helloClient: client,
	}, nil
}

// unary server
func (h *HelloAdapter) SayHello(ctx context.Context, name string) (*hello.HelloResponse, error) {
	helloRequest := &hello.HelloRequest{
		Name: name,
	}

	greet, err := h.helloClient.SayHello(ctx, helloRequest)
	if err != nil {
		log.Fatalln("error saying hello: ", err)
	}

	return greet, nil
}

// server streaming
func (h *HelloAdapter) SayManyHellos(ctx context.Context, name string) {
	helloRequest := &hello.HelloRequest{
		Name: name,
	}

	greetStream, err := h.helloClient.SayManyHellos(ctx, helloRequest)
	if err != nil {
		log.Fatalln("error saying many hellos: ", err)
	}

	for {
		greet, err := greetStream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalln("error on saying many hellos: ", err)
		}

		log.Println(greet.Greet)
	}
}

// Client streaming
func (h *HelloAdapter) SayHelloToEveryone(ctx context.Context, names []string) {
	greetStream, err := h.helloClient.SayHelloToEveryone(ctx)
	if err != nil {
		log.Fatalln("error on say hello to everyone: ", err)
	}

	// loop over the names and send a request
	for _, name := range names {
		req := &hello.HelloRequest{
			Name: name,
		}

		greetStream.Send(req)
		time.Sleep(500 * time.Millisecond) // delay for half a second
	}

	// after sending all names, close stream and receive response
	response, err := greetStream.CloseAndRecv()
	if err != nil {
		log.Fatalln("error on saying many hellos: ", err)
	}

	log.Println(response.Greet)
}

// Bi-directional streaming
func (h *HelloAdapter) SayHelloContinuous(ctx context.Context, names []string) {
	greetStream, err := h.helloClient.SayHelloContinuous(ctx)
	if err != nil {
		log.Fatalln("error on saying hello continuous: ", err)
	}

	greetChan := make(chan struct{})

	// loop over all the names and sends a request to the stream
	go func() {
		for _, name := range names {
			req := hello.HelloRequest{
				Name: name,
			}

			greetStream.Send(&req)
		}

		greetStream.CloseSend() // close the stream
	}()

	// continuously receive response from the stream until it receives an 'io.EOF' error indicating that the stream has been closed
	go func() {
		for {
			greet, err := greetStream.Recv()

			if err == io.EOF {
				break
			}

			if err != nil {
				log.Fatalln("error on saying hello continuous: ", err)
			}

			log.Println(greet.Greet)
		}

		close(greetChan) // close the greet channel; this indicates that both goroutines have finished running
	}()

	<-greetChan // block the main thread
}
