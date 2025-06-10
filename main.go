package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"
)

type Word struct {
	Text          string `json:"text"`
	Commonness    int    `json:"commonness"`
	Offensiveness int    `json:"offensiveness"`
	Sentiment     int    `json:"sentiment"`
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	switch req.RequestContext.HTTP.Method {
	case http.MethodGet:
		return getHandler(req)
	default:
		return notAllowed()
	}

}

func getWordsPage(conn *pgx.Conn, fromWord string, limit int) ([]Word, bool, error) {
	rows, err := conn.Query(context.Background(), `
		SELECT text, commonness, offensiveness, sentiment 
		FROM words
		WHERE text > $1 
		ORDER BY text
		LIMIT $2`, fromWord, limit+1)
	if err != nil {
		panic(fmt.Sprintf("Query failed: %v", err))
	}

	defer rows.Close()

	var words []Word
	for rows.Next() {
		var word Word
		if err := rows.Scan(&word.Text, &word.Commonness, &word.Offensiveness, &word.Sentiment); err != nil {
			return nil, false, err
		}
		words = append(words, word)
	}

	var hasMore = false
	if len(words) > limit {
		hasMore = true
		words = words[:len(words)-1]
	}

	return words, hasMore, nil
}

func getHandler(req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	connStr := os.Getenv("DB_CONNECTION_STRING")

	conn, connErr := pgx.Connect(context.Background(), connStr)
	if connErr != nil {
		panic(fmt.Sprintf("Failed to connect: %v", connErr))
	}

	defer conn.Close(context.Background())

	words, hasMore, pageErr := getWordsPage(conn, req.PathParameters["from"], 100)
	if pageErr != nil {
		panic(fmt.Sprintf("Failed to get words: %v", pageErr))
	}

	response := struct {
		Words   []Word `json:"words"`
		HasMore bool   `json:"hasMore"`
	}{
		Words:   words,
		HasMore: hasMore,
	}

	responseBody, _ := json.Marshal(response)
	return ok(string(responseBody))
}

func ok(content string) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
		Body:       content,
	}, nil
}

func notAllowed() (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusMethodNotAllowed,
		Body:       `{"error": "Unsupported HTTP method"}`,
	}, nil
}

func main() {
	lambda.Start(handler)
}
