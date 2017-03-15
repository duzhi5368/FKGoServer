package framework

import (
PROTO "FKGoServer/FKGRpc_Chat/Proto"
"testing"
"time"

"golang.org/x/net/context"
"google.golang.org/grpc"
)

const (
	address = "192.168.99.100:50008"
)

var (
	conn *grpc.ClientConn
	err  error
)

func TestChat(t *testing.T) {
	// Set up a connection to the server.
	conn, err = grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}

	c := PROTO.NewChatServiceClient(conn)

	// Contact the server and print out its response.
	_, err = c.Reg(context.Background(), &PROTO.Chat_Id{Id: 1})
	if err != nil {
		t.Logf("could not query: %v", err)
	}

	const COUNT = 10
	go send(&PROTO.Chat_Message{Id: 1, Body: []byte("Hello")}, COUNT, t)
	go recv(&PROTO.Chat_Id{1}, COUNT, t)
	go recv(&PROTO.Chat_Id{1}, COUNT, t)
	time.Sleep(3 * time.Second)
}

func send(m *PROTO.Chat_Message, count int, t *testing.T) {
	c := PROTO.NewChatServiceClient(conn)
	for {
		if count == 0 {
			return
		}
		_, err := c.Subscribe(context.Background(), m)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("send:", m)
		count--
	}
}

func recv(chat_id *PROTO.Chat_Id, count int, t *testing.T) {
	c := PROTO.NewChatServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := c.Subscribe(ctx, chat_id)
	if err != nil {
		t.Fatal(err)
	}

	for {
		if count == 0 {
			return
		}
		message, err := stream.Recv()
		if err != nil {
			t.Log(err)
			return
		}
		println("recv:", count)
		t.Log("recv:", message)
		count--
		cancel() // recv should continue until error
	}
}
