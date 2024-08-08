package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

func main() {
	account := flag.String("account", "", "Cloudflare account.")
	namespace := flag.String("namespace", "", "Cloudflare namespace.")
	pageSize := flag.Int("pageSize", 100, "Number of objects to list in each KV call.")
	prefix := flag.String("prefix", "", "The prefix which when provided will filter the listed namespaces.")
	fixStringEncodedJsonFlag := flag.Bool("fixStringEncodedJson", false, "Fix string encoded json. Only run this on namespaces containing JSON data incorrectly encoded as strings.")
	token := os.Getenv("CLOUDFLARE_API_TOKEN")
	flag.Parse()

	if *account == "" {
		log.Fatalf("missing account flag")
	}

	if *namespace == "" {
		log.Fatalf("missing namespace flag")
	}

	if token == "" {
		log.Fatalf("missing CLOUDFLARE_API_TOKEN env var")
	}

	client, err := cloudflare.NewWithAPIToken(token)
	if err != nil {
		log.Fatalf("problem constructing the client:%+v", err)
	}

	ctx := context.Background()
	accountContainer := cloudflare.AccountIdentifier(*account)
	var cursor string
	for {
		resp, err := client.ListWorkersKVKeys(ctx,
			accountContainer,
			cloudflare.ListWorkersKVsParams{
				NamespaceID: *namespace,
				Limit:       *pageSize,
				Cursor:      cursor,
				Prefix:      *prefix,
			},
		)

		if err != nil {
			log.Fatalf("problem listing kv namespace:%+v", err)
		}

		for _, key := range resp.Result {
			data, err := client.GetWorkersKV(ctx, accountContainer, cloudflare.GetWorkersKVParams{NamespaceID: *namespace, Key: key.Name})
			if err != nil {
				log.Fatalf("problem fetching kv value:%+v", err)
			}

			if *fixStringEncodedJsonFlag {
				result, err := client.ListWorkersKVKeys(ctx, accountContainer, cloudflare.ListWorkersKVsParams{Prefix: key.Name})
				if err != nil {
					log.Fatal(err)
				}

				if result.Count != 1 {
					log.Fatalf("unexpected key count:%d", result.Count)
				}

				keyMetadata := result.Result[0]

				data, err = fixStringEncodedJson(ctx, client, accountContainer, data, *namespace, keyMetadata)
				if err != nil {
					log.Fatal(err)
				}
			}

			js, err := json.Marshal(KVResult{
				Key:   key,
				Value: string(data),
			})
			if err != nil {
				log.Fatalf("problem marshaling json:%+v", err)
			}

			log.Println(string(js))
		}

		// KV pagination ends when the cursor is empty.
		if resp.Cursor == "" {
			return
		}

		// Set the cursor for the next page to list.
		cursor = resp.Cursor
	}
}

// KVResult is a single key value pair with metadata.
type KVResult struct {
	Key   cloudflare.StorageKey
	Value string
}

type kvClient interface {
	WriteWorkersKVEntries(ctx context.Context, rc *cloudflare.ResourceContainer, params cloudflare.WriteWorkersKVEntriesParams) (cloudflare.Response, error)
}

var jsRx = regexp.MustCompile(`^\s*["]?\s*{.*`)

func fixStringEncodedJson(ctx context.Context, client kvClient, accountContainer *cloudflare.ResourceContainer, data []byte, namespace string, metadata cloudflare.StorageKey) ([]byte, error) {
	var result json.RawMessage
	// Handle json data which cannot be decoded
	err := json.Unmarshal(data, &result)
	var jsError *json.SyntaxError
	if errors.As(err, &jsError) && jsRx.Match(data) {
		dStr := strings.ReplaceAll(string(data), `\"`, `"`)
		dStr = strings.ReplaceAll(dStr, `\\`, `\`)
		dStr = strings.TrimPrefix(dStr, `"`)
		dStr = strings.TrimSuffix(dStr, `"`)
		data = []byte(dStr)
		response, err := client.WriteWorkersKVEntries(ctx, accountContainer,
			cloudflare.WriteWorkersKVEntriesParams{
				NamespaceID: namespace,
				KVs: []*cloudflare.WorkersKVPair{
					{
						Key:        metadata.Name,
						Value:      dStr,
						Expiration: metadata.Expiration,
						Metadata:   metadata.Metadata,
					},
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error writing kv entry:%w", err)
		}
		if !response.Success {
			return nil, fmt.Errorf("uncessful kv write:%+v", response)
		}
	}
	return data, nil
}
