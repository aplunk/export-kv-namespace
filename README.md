# Example of Exporting a KV Namespace

This repo contains an example written in go that exports a KV namespace. It makes no attempt to concurrently download keys.

## Requirements

- Set the env variable CLOUDFLARE_API_TOKEN to your api token created in the Cloudflare dashboard (use the workers edit template to create the token).
- Create a KV namespace in the Cloudflare dashboard.


## Usage

### Help Text

```shell
go run main.go --help
Usage of export-kv-namespace:
  -account string
        Cloudflare account.
  -namespace string
        Cloudflare namespace.
  -pageSize int
        Number of objects to list in each KV call. (default 100)
  -prefix string
        The prefix which when provided will filter the listed namespaces.
```

### Running

```shell
CLOUDFLARE_API_TOKEN=<MY_API_TOKEN> go run main.go -account 8853d02acf45198ad5f7b5ae22224e81 -namespace 217c5dd53e124bb7a5463495bc4ea484 -pageSize 10
2024/04/12 17:17:59 {"Key":{"name":"1","expiration":0,"metadata":null},"Value":"1"}
2024/04/12 17:18:01 {"Key":{"name":"2","expiration":0,"metadata":null},"Value":"2"}
```
