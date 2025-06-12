package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

type Range struct {
	Min int
	Max int
}

type Query struct {
	Commonness    Range
	Offensiveness Range
	Sentiment     Range
	RandomCount   int
	RandomSeed    string
	WordLength    Range
	StartFrom     string
	Limit         int
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	switch req.RequestContext.HTTP.Method {
	case http.MethodGet:
		return getHandler(req)
	default:
		return notAllowed()
	}

}

func buildQuery(query Query) (string, []any) {

	queryParams := []any{}

	queryText := `
		SELECT text, commonness, offensiveness, sentiment
		FROM words
		WHERE text > $1 `

	queryParams = append(queryParams, query.StartFrom)

	if query.RandomCount > 0 {
		queryText = `SELECT text, commonness, offensiveness, sentiment FROM ( ` + queryText
	}

	queryText += ` AND commonness >= $2 AND commonness <= $3 `
	queryParams = append(queryParams, query.Commonness.Min, query.Commonness.Max)

	queryText += ` AND offensiveness >= $4 AND offensiveness <= $5 `
	queryParams = append(queryParams, query.Offensiveness.Min, query.Offensiveness.Max)

	queryText += ` AND sentiment >= $6 AND sentiment <= $7 `
	queryParams = append(queryParams, query.Sentiment.Min, query.Sentiment.Max)

	if query.WordLength.Min > 0 {
		queryText += ` AND LENGTH(text) >= $8 `
		queryParams = append(queryParams, query.WordLength.Min)
	}

	if query.WordLength.Max > 0 {
		paramCount := len(queryParams) + 1
		queryText += ` AND LENGTH(text) <= $` + fmt.Sprint(paramCount) + ` `
		queryParams = append(queryParams, query.WordLength.Max)
	}

	if query.RandomCount > 0 {
		paramCount := len(queryParams) + 1
		queryText += ` ORDER BY fnv64(CONCAT($` + fmt.Sprint(paramCount) + `, text)) LIMIT $` + fmt.Sprint(paramCount+1) + ` ) AS subquery `
		queryParams = append(queryParams, query.RandomSeed)
		queryParams = append(queryParams, query.RandomCount)
	}

	paramCount := len(queryParams) + 1
	queryText += `ORDER BY text ASC LIMIT $` + fmt.Sprint(paramCount) + `;`
	queryParams = append(queryParams, query.Limit+1)

	log.Printf("Generated query:\n%s", queryText)

	return queryText, queryParams
}

func getWordsPage(conn *pgx.Conn, query Query) ([]Word, bool, error) {

	queryText, queryParams := buildQuery(query)

	rows, err := conn.Query(context.Background(), queryText, queryParams...)
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
	if len(words) > query.Limit {
		hasMore = true
		words = words[:len(words)-1]
	}

	return words, hasMore, nil
}

func getRangeFromParameters(params map[string]string, name string, min int, max int) Range {

	minValue := min
	if value, exists := params["min"+name]; exists {
		minValue, _ = strconv.Atoi(value)
	}

	maxValue := max
	if value, exists := params["max"+name]; exists {
		maxValue, _ = strconv.Atoi(value)
	}

	if minValue < min || minValue > max {
		minValue = min
	}

	if maxValue < min || maxValue > max {
		maxValue = max
	}

	return Range{Min: minValue, Max: maxValue}
}

func getIntFromParameters(params map[string]string, name string, defaultValue int) int {

	value := defaultValue
	if valueText, exists := params[name]; exists {
		value, _ = strconv.Atoi(valueText)
	}

	return value
}

func getStringFromParameters(params map[string]string, name string, defaultValue string) string {

	value := defaultValue
	if valueText, exists := params[name]; exists {
		value = valueText
	}

	return value
}

func getQueryFromParameters(params map[string]string) Query {

	query := Query{
		Commonness:    getRangeFromParameters(params, "Commonness", 0, 5),
		Offensiveness: getRangeFromParameters(params, "Offensiveness", 0, 5),
		Sentiment:     getRangeFromParameters(params, "Sentiment", -5, 5),
		WordLength:    getRangeFromParameters(params, "Length", 0, 255),
		RandomCount:   getIntFromParameters(params, "randomCount", 0),
		RandomSeed:    getStringFromParameters(params, "randomSeed", fmt.Sprint(time.Now().Unix())),
		StartFrom:     getStringFromParameters(params, "startFrom", ""),
		Limit:         getIntFromParameters(params, "limit", 100),
	}

	return query
}

func getHandler(req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	connStr := os.Getenv("DB_CONNECTION_STRING")

	conn, connErr := pgx.Connect(context.Background(), connStr)
	if connErr != nil {
		panic(fmt.Sprintf("Failed to connect: %v", connErr))
	}

	defer conn.Close(context.Background())

	query := getQueryFromParameters(req.QueryStringParameters)

	words, hasMore, pageErr := getWordsPage(conn, query)

	if pageErr != nil {
		panic(fmt.Sprintf("Failed to get words: %v", pageErr))
	}

	response := struct {
		Query   Query  `json:"query"`
		Words   []Word `json:"words"`
		HasMore bool   `json:"hasMore"`
	}{
		Query:   query,
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
