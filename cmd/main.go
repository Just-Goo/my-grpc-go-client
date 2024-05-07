package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/Just-Goo/my-grpc-go-client/internal/adapter/bank"
	"github.com/Just-Goo/my-grpc-go-client/internal/adapter/hello"
	dbank "github.com/Just-Goo/my-grpc-go-client/internal/application/domain/bank"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(logWriter{})

	// disable TLS for now
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// open connection to gRPC server
	conn, err := grpc.Dial("localhost:9090", opts...)
	if err != nil {
		log.Fatalln("cannot connect to gRPC server:", err)
	}

	defer conn.Close()

	bankAdapter, err := bank.NewBankAdapter(conn)
	if err != nil {
		log.Fatalln("cannot create bank adapter:", err)
	}

	// runGetCurrentBalance(bankAdapter, "7835697001")
	// runFetchExchangeRates(bankAdapter, "USD", "IDR")
	// runSummarizeTransactions(bankAdapter, "7835697002", 10)
	runTransferMultiple(bankAdapter, "7835697002", "7835697001", 100)

	// helloAdapter, err := hello.NewHelloAdapter(conn)
	// if err != nil {
	// 	log.Fatalln("cannot create hello adapter:", err)
	// }

	// runSayHello(helloAdapter, "Charles")
	// runSayManyHellos(helloAdapter, "Miana")
	// runSayHelloToEveryone(helloAdapter, []string{"Jake", "Logan", "Paul", "Tina", "Florence"})
	// runSayHelloContinuous(helloAdapter, []string{"Jake", "Logan", "Paul", "Tina", "Florence", "Dan", "Faith"})

}

func runGetCurrentBalance(adapter *bank.BankAdapter, acct string) {
	bal, err := adapter.GetCurrentBalance(context.Background(), acct)
	if err != nil {
		log.Fatalln("cannot call GetCurrentBalance:", err)
	}

	log.Println(bal)

}

func runFetchExchangeRates(adapter *bank.BankAdapter, fromCur, toCur string) {
	adapter.FetchExchangeRates(context.Background(), fromCur, toCur)
}

func runSummarizeTransactions(adapter *bank.BankAdapter, acct string, numDummyTransactions int) {
	var tx []dbank.Transaction

	for i := 1; i < numDummyTransactions; i++ {
		ttype := dbank.TransactionTypeIn

		if i%3 == 0 {
			ttype = dbank.TransactionTypeOut
		}

		t := dbank.Transaction{
			Amount:          float64(rand.Intn(500) + 10),
			TransactionType: ttype,
			Notes:           fmt.Sprintf("Dummy trnsaction %v", i),
		}

		tx = append(tx, t)
	}
	adapter.SummarizeTransactions(context.Background(), acct, tx)
}

func runTransferMultiple(adapter *bank.BankAdapter, fromAcct, toAcct string, numDummyTransactions int) {
	var trf []dbank.TransferTransaction

	for i := 1; i < numDummyTransactions; i++ {
		tr := dbank.TransferTransaction{
			FromAccountNumber: fromAcct,
			ToAccountNumber:   toAcct,
			Currency:          "USD",
			Amount:            float64(rand.Intn(200) + 5),
		}

		trf = append(trf, tr)
	}

	adapter.TransferMultiple(context.Background(), trf)
}

func runSayHello(adapter *hello.HelloAdapter, name string) {
	greetResponse, err := adapter.SayHello(context.Background(), name)
	if err != nil {
		log.Fatalln("cannot call sayHello:", err)
	}

	log.Println(greetResponse.Greet)
}

func runSayManyHellos(adapter *hello.HelloAdapter, name string) {
	adapter.SayManyHellos(context.Background(), name)
}

func runSayHelloToEveryone(adapter *hello.HelloAdapter, names []string) {
	adapter.SayHelloToEveryone(context.Background(), names)
}

func runSayHelloContinuous(adapter *hello.HelloAdapter, names []string) {
	adapter.SayHelloContinuous(context.Background(), names)
}
