package bank

import (
	"context"
	"io"
	"log"
	"time"

	dbank "github.com/Just-Goo/my-grpc-go-client/internal/application/domain/bank"
	"github.com/Just-Goo/my-grpc-go-client/internal/port"
	"github.com/Just-Goo/my-grpc-proto/protogen/go/bank"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BankAdapter struct {
	bankClient port.BankClientPort
}

func NewBankAdapter(conn *grpc.ClientConn) (*BankAdapter, error) {
	client := bank.NewBankServiceClient(conn)

	return &BankAdapter{
		bankClient: client,
	}, nil
}

func (b *BankAdapter) GetCurrentBalance(ctx context.Context, acct string) (*bank.CurrentBalanceResponse, error) {
	bankRequest := &bank.CurrentBalanceRequest{
		AccountNumber: acct,
	}

	bal, err := b.bankClient.GetCurrentBalance(ctx, bankRequest)
	if err != nil {
		st, _ := status.FromError(err)
		log.Fatalln("[FATAL] Error on GetCurrentBalance: ", st)
	}

	return bal, nil
}

func (b *BankAdapter) FetchExchangeRates(ctx context.Context, fromCur, toCur string) {
	bankRequest := &bank.ExchangeRateRequest{
		FromCurrency: fromCur,
		ToCurrency:   toCur,
	}

	exchangeRateStream, err := b.bankClient.FetchExchangeRates(ctx, bankRequest)
	if err != nil {
		st, _ := status.FromError(err)
		log.Fatalln("[FATAL] Error on FetchExchangeRates: ", st)
	}

	for {
		rate, err := exchangeRateStream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			st, _ := status.FromError(err)

			if st.Code() == codes.InvalidArgument {
				log.Fatalln("[FATAL] Error on FetchExchangeRates: ", st.Message())
			}
		}

		log.Printf("Rate at %v from %v to %v is %v\n", rate.Timestamp, rate.FromCurrency, rate.ToCurrency, rate.Rate)

	}
}

func (b *BankAdapter) SummarizeTransactions(ctx context.Context, acct string, tx []dbank.Transaction) {
	txStream, err := b.bankClient.SummarizeTransactions(ctx)
	if err != nil {
		log.Fatalln("[FATAL] Error on SummarizeTransactions: ", err)
	}

	for _, t := range tx {
		now := time.Now()
		ttype := bank.TransactionType_TRANSACTION_TYPE_UNSPECIFIED

		if t.TransactionType == dbank.TransactionTypeIn {
			ttype = bank.TransactionType_TRANSACTION_TYPE_IN
		} else if t.TransactionType == dbank.TransactionTypeOut {
			ttype = bank.TransactionType_TRANSACTION_TYPE_OUT
		}

		bankRequest := &bank.Transaction{
			AccountNumber: acct,
			Type:          ttype,
			Amount:        t.Amount,
			Notes:         t.Notes,
			Timestamp: &datetime.DateTime{
				Year:  int32(now.Year()),
				Month: int32(now.Month()),
				Day:   int32(now.Day()),
			},
		}

		txStream.Send(bankRequest) // handle error
	}
	summary, err := txStream.CloseAndRecv()
	if err != nil {
		st, _ := status.FromError(err) // handle error
		log.Fatalln("[FATAL] Error on SummarizeTransactions: ", st)
	}

	log.Println(summary)
}

func (b *BankAdapter) TransferMultiple(ctx context.Context, trf []dbank.TransferTransaction) {
	trfStream, err := b.bankClient.TransferMultiple(ctx)
	if err != nil {
		log.Fatalln("[FATAL] Error on TransferMultiple: ", err)
	}

	trfChan := make(chan struct{})

	// build and send each transfer request message to the server
	go func() {
		for _, tt := range trf {
			req := &bank.TransferRequest{
				FromAccountNumber: tt.FromAccountNumber,
				ToAccountNumber:   tt.ToAccountNumber,
				Currency:          tt.Currency,
				Amount:            tt.Amount,
			}

			trfStream.Send(req)
		}

		trfStream.CloseSend() // close the stream (this indicates that there are no more messages to be sent)
	}()

	// receive messages from the server
	go func() {
		for {
			res, err := trfStream.Recv()

			// continue receiving messages until server indicates that it has finished sending by returning an 'io.EOF' error
			if err == io.EOF {
				break
			}

			if err != nil {
				handleTransferErrorGrpc(err) // dedicated error handler function
				break
			}

			log.Printf("Transfer status %v on %v\n", res.Status, res.Timestamp)

		}

		close(trfChan) // close the greet channel; this indicates that both goroutines have finished running
	}()

	<-trfChan
}

func handleTransferErrorGrpc(err error) {
	st := status.Convert(err)

	log.Printf("Error %v on TransferMultiple : %v", st.Code(), st.Message())

	for _, detail := range st.Details() {
		switch t := detail.(type) {
		case *errdetails.PreconditionFailure:
			for _, violation := range t.GetViolations() {
				log.Println("[VIOLATION]", violation)
			}
		case *errdetails.ErrorInfo:
			log.Printf("Error on : %v, with reason %v\n", t.Domain, t.Reason)
			for k, v := range t.GetMetadata() {
				log.Printf("  %v : %v\n", k, v)
			}
		}
	}
}
