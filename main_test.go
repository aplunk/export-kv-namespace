package main

import (
	"context"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

type testClient struct {
	test func(ctx context.Context, rc *cloudflare.ResourceContainer, params cloudflare.WriteWorkersKVEntryParams) (cloudflare.Response, error)
}

func (t testClient) WriteWorkersKVEntry(ctx context.Context, rc *cloudflare.ResourceContainer, params cloudflare.WriteWorkersKVEntryParams) (cloudflare.Response, error) {
	return t.test(ctx, rc, params)
}

func TestFixStringEncodedJson(t *testing.T) {
	tests := []struct {
		description string
		input       string
		response    string
		called      bool
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
			description: "fixes partially escaped json",
			input:       `{\"test\": \"me"}`,
			response:    `{"test": "me"}`,
			called:      true,
		},
		{
			description: "does not modify json encoded strings",
			input:       `afwfewfw\"dfawefwa`,
			response:    `afwfewfw\"dfawefwa`,
			called:      false,
		},
	}

	for _, test := range tests {
		t.Logf("running test:%s", test.description)
		called := false
		data, err := fixStringEncodedJson(context.Background(),
			testClient{
				func(ctx context.Context, rc *cloudflare.ResourceContainer, params cloudflare.WriteWorkersKVEntryParams) (cloudflare.Response, error) {
					called = true
					return cloudflare.Response{Success: true}, nil
				},
			},
			nil, []byte(test.input), "test_namespace", test.description,
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
