# BBAAS Go SDK

Lightweight API client for the BBAAS API. This SDK wraps HTTP endpoints only and does not include Playwright dependencies.

## Install

This SDK currently lives in the same repository/module:

```go
import "github.com/brian-nunez/bbaas-api/sdk/go/bbaas"
```

## Usage

```go
ctx := context.Background()

client, err := bbaas.NewClient("http://localhost:8080")
if err != nil {
    panic(err)
}

client.SetAPIToken("bka_...generated_from_dashboard...")

spawned, err := client.SpawnBrowser(ctx, bbaas.SpawnBrowserRequest{})
if err != nil {
    panic(err)
}

fmt.Println(spawned.Browser.CDPURL)
```

## Methods

- `SpawnBrowser`
- `ListBrowsers`
- `GetBrowser`
- `KeepAliveBrowser`
- `CloseBrowser`

## Auth

Use `SetAPIToken` or `WithAPIToken(...)` during client construction.
