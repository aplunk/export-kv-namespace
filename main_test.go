package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	test func(ctx context.Context, rc *cloudflare.ResourceContainer, params cloudflare.WriteWorkersKVEntriesParams) (cloudflare.Response, error)
}

func (t testClient) WriteWorkersKVEntries(ctx context.Context, rc *cloudflare.ResourceContainer, params cloudflare.WriteWorkersKVEntriesParams) (cloudflare.Response, error) {
	return t.test(ctx, rc, params)
}

func TestFixStringEncodedJson(t *testing.T) {
	tests := []struct {
		description string
		input       string
		response    string
		called      bool
		expiration  int
		metadata    any
	}{
		{
			description: "handles clean json correctly",
			input:       `{"test": "me"}`,
			response:    `{"test": "me"}`,
			called:      false,
		},
		{
			description: "fixes escaped json",
			input:       `{\"test\": \"me\"}`,
			response:    `{"test": "me"}`,
			called:      true,
		},
		{
			description: "fixes escaped json leading whitespace",
			input:       ` {\"test\": \"me\"}`,
			response:    ` {"test": "me"}`,
			called:      true,
		},
		{
			description: "fixes stringified json",
			input:       `"{\"test\": \"me"}"`,
			response:    `{"test": "me"}`,
			called:      true,
		},
		{
			description: "fixes json strings",
			input:       `"{\"test\": \"me"}"`,
			response:    `{"test": "me"}`,
			called:      true,
		},
		{
			description: "handles metadata and expiration",
			input:       `"{\"test\": \"me"}"`,
			response:    `{"test": "me"}`,
			expiration:  1,
			metadata:    "testt",
			called:      true,
		},
		{
			description: "handles nested escaped quotes",
			input:       `"{\"test\": \"\\""}"`,
			response:    `{"test": "\""}`,
			called:      true,
		},
		{
			description: "handles tripple slashes",
			input:       `{\"test\": \"\\\"\"}`,
			response:    `{"test": "\""}`,
			called:      true,
		},
		{
			description: "handles nested escaped slashes",
			input:       `"{\"test\": \" \\\ \"}"`,
			response:    `{"test": " \\ "}`,
			called:      true,
		},
		{
			description: "does not modify json encoded strings",
			input:       `afwfewfw\"dfawefwa`,
			response:    `afwfewfw\"dfawefwa`,
			called:      false,
		},

		{
			description: "handles quad slashes",
			input:       `{\"test\": \" \\\\ \"}`,
			response:    `{"test": " \\ "}`,
			called:      true,
		},
	}

	for _, test := range tests {
		t.Logf("running test:%s", test.description)
		called := false
		data, err := fixStringEncodedJson(context.Background(),
			testClient{
				func(ctx context.Context, rc *cloudflare.ResourceContainer, params cloudflare.WriteWorkersKVEntriesParams) (cloudflare.Response, error) {
					called = true
					if params.KVs[0].Expiration != test.expiration {
						t.Fatalf("unexpected expiration actual:%d expected:%d", params.KVs[0].Expiration, test.expiration)
					}

					if params.KVs[0].Metadata != test.metadata {
						t.Fatalf("unexpected metadata actual:%d expected:%d", params.KVs[0].Metadata, test.metadata)
					}
					return cloudflare.Response{Success: true}, nil
				},
			},
			nil, []byte(test.input), "test_namespace",
			cloudflare.StorageKey{Name: test.description, Expiration: test.expiration, Metadata: test.metadata},
		)
		if err != nil {
			t.Fatalf("unexpected test error:%+v", err)
		}
		if test.called != called {
			t.Fatalf("expected %s to be called", test.description)
		}
		if string(data) != test.response {
			t.Fatalf("unexpected test response expected:%s actual:%s", test.response, string(data))
		}
	}
}

func TestMoreCases(t *testing.T) {
	data, err := os.ReadFile("./testdata/cases.txt")
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	for i := 0; i < len(lines); i += 2 {
		got, ok := fixStringEncodedJsonOnly([]byte(lines[i+1]))
		require.Equal(t, lines[i], string(got))
		require.True(t, ok)
	}
}
