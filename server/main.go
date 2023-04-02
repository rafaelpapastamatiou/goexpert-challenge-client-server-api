package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Format of the quotation in the external api
type ExternalQuotationResponse struct {
	USDBRL Quotation
}

// Format of the response
type QuotationResponse struct {
	Bid string `json:"bid"`
}

// Internal quotation struct
type Quotation struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

// Gorm entity
type DbQuotation struct {
	ID int `gorm:"primaryKey"`
	Quotation
	gorm.Model
}

// Change Gorm quotation table name
func (DbQuotation) TableName() string {
	return "quotations"
}

func main() {
	ConnectToDb()

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", HandleQuotation)
	http.ListenAndServe(":8080", mux)
}

func HandleQuotation(w http.ResponseWriter, r *http.Request) {
	// CURRENT REQUEST CONTEXT
	ctx := r.Context()

	errChan := make(chan error)
	quotationChan := make(chan *Quotation)

	go SearchAndSaveQuotation(ctx, quotationChan, errChan)

	select {
	case <-ctx.Done():
		println("Request cancelled by the client")
		return

	case err := <-errChan:
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}

		panic(err)

	case quotation := <-quotationChan:
		// WRITE RESPONSE
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&QuotationResponse{Bid: quotation.Bid})
	}
}

func SearchAndSaveQuotation(ctx context.Context, quotationChan chan *Quotation, errChan chan error) {
	quotation, err := SearchQuotation(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			println("API context timeout")
		}
		errChan <- err
		return
	}

	err = SaveQuotation(ctx, quotation)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			println("SQL context timeout")
		}
		errChan <- err
		return
	}

	quotationChan <- quotation
}

// SEND THE REQUEST TO THE EXTERNAL API
func SearchQuotation(ctx context.Context) (*Quotation, error) {
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var quotationResponse ExternalQuotationResponse

	err = json.Unmarshal(body, &quotationResponse)
	if err != nil {
		return nil, err
	}

	return &quotationResponse.USDBRL, nil
}

// SAVE THE QUOTATION IN THE DATABASE
func SaveQuotation(ctx context.Context, quotation *Quotation) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	err := db.WithContext(ctx).Create(&DbQuotation{Quotation: *quotation}).Error
	if err != nil {
		return err
	}

	return nil
}

// GORM DB CONNECTION
var db *gorm.DB // CREATE A GLOBAL VAR TO HOLD GORM.DB

func ConnectToDb() {
	var err error

	db, err = gorm.Open(sqlite.Open("server.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&DbQuotation{})
}
