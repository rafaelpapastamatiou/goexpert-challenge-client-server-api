package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Quotation struct {
	Bid string `json:"bid"`
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	c := http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		panic(err)
	}

	res, err := c.Do(req)
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		println("External request timeout")
		return
	}
	if err != nil {
		panic(err)
	}

	if res.StatusCode != http.StatusOK {
		fmt.Printf("Request error. Status: %v\n", res.Status)
		return
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var quotation Quotation
	err = json.Unmarshal(body, &quotation)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("cotacao.txt")
	if err != nil {
		panic(err)
	}

	resultString := fmt.Sprintf("Dólar: %v", quotation.Bid)

	_, err = file.WriteString(resultString)
	if err != nil {
		panic(err)
	}

	println(resultString)
}
