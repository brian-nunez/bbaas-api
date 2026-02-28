package main

import (
	"context"
	"fmt"
	"log"

	"github.com/brian-nunez/bbaas-api/sdk/go/bbaas"
	"github.com/playwright-community/playwright-go"
)

func getCDPURLFromService(ctx context.Context) string {
	// apiKey := "bka_021dc41df7b15f4a36fe1376852c29d7288f37c70fcfef6b" // os.Getenv("BBAAS_API_KEY")
	// Prod
	apiKey := "bka_df1249d8247f54022c27bbcc0a00598726af5873d4eac712"
	if apiKey == "" {
		panic("set BBAAS_API_KEY before running this command")
	}

	fmt.Printf("Using API key: %s\n", apiKey)
	client, err := bbaas.NewClient(
		// "http://localhost:8080",
		"http://10.0.0.116:8080",
		bbaas.WithAPIToken(apiKey),
	)
	fmt.Printf("Created BBAAS client: %+v\n", client)
	if err != nil {
		fmt.Printf("Error creating BBAAS client: %v\n", err)
		panic(err)
	}

	spawned, err := client.SpawnBrowser(ctx, bbaas.SpawnBrowserRequest{})
	if err != nil {
		fmt.Printf("Error spawning browser: %v\n", err)
		panic(err)
	}

	fmt.Printf("Spawned browser with ID: %s\n", spawned.Browser.CDPHTTPURL)
	return spawned.Browser.CDPURL
}

func main() {
	// cdpURL := os.Getenv("BBAAS_CDP_URL")
	// if cdpURL == "" {
	cdpURL := getCDPURLFromService(context.Background())
	// }

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Error starting playwright %v", err)
	}

	browser, err := pw.Chromium.ConnectOverCDP(cdpURL)
	// browser, err := pw.Chromium.ConnectOverCDP("http://10.0.0.116:46865")
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

	if _, err = page.Goto("https://www.google.com"); err != nil {
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

	// err = client.CloseBrowser(context.Background(), spawned.Browser.ID)
	// if err != nil {
	// 	log.Fatalf("could not close browser: %v", err)
	// }
	//
	// fmt.Println("Browser closed successfully")
}
