// Command openapi exports the typed HTTP contract without a database.
package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/transport"
	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	contract := transport.NewInfoHandler(nil).RegisterRoutes(router, router.Group(transport.BasePath))
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(contract); err != nil {
		log.Fatal(err)
	}
}
