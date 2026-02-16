package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/brian-nunez/bbaas-api/sdk/go/bbaas"
	"github.com/playwright-community/playwright-go"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("BBAAS_API_KEY")
	if apiKey == "" {
		panic("set BBAAS_API_KEY before running this command")
	}

	client, err := bbaas.NewClient(
		"http://localhost:8080",
		bbaas.WithAPIToken(apiKey),
	)
	if err != nil {
		panic(err)
	}

	spawned, err := client.SpawnBrowser(ctx, bbaas.SpawnBrowserRequest{})
	if err != nil {
		panic(err)
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Error starting playwright %v", err)
	}

	browser, err := pw.Chromium.ConnectOverCDP(spawned.Browser.CDPURL)
	if err != nil {
		log.Fatalf("Error starting browser %v", err)
	}

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}

	err = page.SetViewportSize(1280, 664)
	if err != nil {
		log.Fatalf("could not set viewport size: %v", err)
	}

	if _, err = page.Goto("https://www.b8z.me/"); err != nil {
		log.Fatalf("could not goto: %v", err)
	}

	entries, err := page.Locator("header,footer,main").All()
	if err != nil {
		log.Fatalf("could not get entries: %v", err)
	}

	fmt.Printf("Found %d entries\n", len(entries))

	for i, entry := range entries {
		title, err := entry.Locator("a").AllTextContents()
		if err != nil {
			log.Fatalf("could not get text content: %v", err)
			continue
		}

		fmt.Printf("%d: %s\n", i+1, title)
	}

	err = client.CloseBrowser(context.Background(), spawned.Browser.ID)
	if err != nil {
		log.Fatalf("could not close browser: %v", err)
	}

	fmt.Println("Browser closed successfully")
}
