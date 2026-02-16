package main

import (
	"context"
	"fmt"
	"os"

	"github.com/brian-nunez/bbaas-api/sdk/go/bbaas"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("BBAAS_API_KEY")
	if apiKey == "" {
		panic("set BBAAS_API_KEY before running this command")
	}

	client, err := bbaas.NewClient("http://localhost:8080", bbaas.WithAPIToken(apiKey))
	if err != nil {
		panic(err)
	}

	spawned, err := client.SpawnBrowser(ctx, bbaas.SpawnBrowserRequest{})
	if err != nil {
		panic(err)
	}

	fmt.Println(spawned.Browser.CDPURL)

	err = client.CloseBrowser(context.Background(), spawned.Browser.ID)
	if err != nil {
		panic(err)
	}

	fmt.Println("Browser closed successfully")
}
