package main

import (
	"context"
	"fmt"

	"github.com/brian-nunez/bbaas-api/sdk/go/bbaas"
)

func main() {

	ctx := context.Background()

	client, err := bbaas.NewClient("http://localhost:8080")
	if err != nil {
		panic(err)
	}

	registered, err := client.RegisterApplication(ctx, bbaas.RegisterApplicationRequest{
		Name:              "automation-app",
		Description:       "Runs browser automation",
		GitHubProfileLink: "https://github.com/brian-nunez",
	})
	if err != nil {
		panic(err)
	}

	client.SetAPIToken(registered.APIToken)

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
