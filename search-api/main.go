package main

import (
  "encoding/json"
  "fmt"
  "log"
  "net/http"
  "strconv"
  "time"

  "github.com/gin-gonic/gin"
  "github.com/olivere/elastic"
  "github.com/teris-io/shortid"
)

// Setup the index (collection of items) and type (single item) for elasticsearch
const (
  elasticIndexName = "documents"
  elasticTypeName  = "document"
)

// Define fields for a single Document item
type Document struct {
  ID        string    `json:"id"`
  Title     string    `json:"title"`
  CreatedAt time.Time `json:"created_at"`
  Content   string    `json:"content"`
}

// Define the request body for a single document
type DocumentRequest struct {
  Title   string `json:"title"`
  Content string `json:"content"`
}

// Define the response format for creating a document
type DocumentResponse struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Content   string    `json:"content"`
}

// Define the format of a response from Elasticsearch
type SearchResponse struct {
	Time      string             `json:"time"`
	Hits      string             `json:"hits"`
	Documents []DocumentResponse `json:"documents"`
}

var (
  elasticClient *elastic.Client
)

func main() {
  var err error

  // try to connect to elasticsearch every few seconds
  // this is because docker services aren't available immediately
  for {
    elasticClient, err = elastic.NewClient(
      elastic.SetURL("http://elasticsearch:9200"),
      elastic.SetSniff(false),
    )
    if err != nil {
      log.Println(err)
      time.Sleep(3 * time.Second)
    } else {
      break
    }
  }

  // run an http server on port 8080 using gin with two endpoints
  r := gin.Default()

  r.POST("/documents", createDocumentsEndpoint)
	r.GET("/search", searchEndpoint)

  if err = r.Run(":8080"); err != nil {
    log.Fatal(err)
  }

}

// Called when POSTing to /documents
func createDocumentsEndpoint(c *gin.Context) {
	// Parse request
	var docs []DocumentRequest
	if err := c.BindJSON(&docs); err != nil {
		errorResponse(c, http.StatusBadRequest, "Malformed request body")
		return
	}

	// Insert documents in bulk
	bulk := elasticClient.
		Bulk().
		Index(elasticIndexName).
		Type(elasticTypeName)

	for _, d := range docs {
		doc := Document{
			ID:        shortid.MustGenerate(),
			Title:     d.Title,
			CreatedAt: time.Now().UTC(),
			Content:   d.Content,
		}
		bulk.Add(elastic.NewBulkIndexRequest().Id(doc.ID).Doc(doc))
	}
	if _, err := bulk.Do(c.Request.Context()); err != nil {
		log.Println(err)
		errorResponse(c, http.StatusInternalServerError, "Failed to create documents")
		return
	}
	c.Status(http.StatusOK)
}

// Called when GETting /search
func searchEndpoint(c *gin.Context) {
	// Parse request
	query := c.Query("query")
	if query == "" {
		errorResponse(c, http.StatusBadRequest, "Query not specified")
		return
	}

  // set default values for these pagination params
	skip := 0
	take := 10
	if i, err := strconv.Atoi(c.Query("skip")); err == nil {
		skip = i
	}
	if i, err := strconv.Atoi(c.Query("take")); err == nil {
		take = i
	}
	// Perform search in elastic
	esQuery := elastic.NewMultiMatchQuery(query, "title", "content").
		Fuzziness("2").
		MinimumShouldMatch("2")
	result, err := elasticClient.Search().
		Index(elasticIndexName).
		Query(esQuery).
		From(skip).Size(take).
		Do(c.Request.Context())
	if err != nil {
		log.Println(err)
		errorResponse(c, http.StatusInternalServerError, "Something went wrong")
		return
	}
	res := SearchResponse{
		Time: fmt.Sprintf("%d", result.TookInMillis),
		Hits: fmt.Sprintf("%d", result.Hits.TotalHits),
	}
	// Transform search results before returning them
	docs := make([]DocumentResponse, 0)
	for _, hit := range result.Hits.Hits {
		var doc DocumentResponse
		json.Unmarshal(*hit.Source, &doc)
		docs = append(docs, doc)
	}
	res.Documents = docs
	c.JSON(http.StatusOK, res)
}

func errorResponse(c *gin.Context, code int, err string) {
	c.JSON(code, gin.H{
		"error": err,
	})

