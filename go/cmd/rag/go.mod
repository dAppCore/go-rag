module dappco.re/go/rag/cmd/rag

go 1.26.0

require (
	dappco.re/go v0.9.0
	dappco.re/go/cli v0.9.0
	dappco.re/go/rag v0.0.0
)

require (
	dappco.re/go/log v0.9.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728 // indirect
	github.com/mailru/easyjson v0.9.2 // indirect
	github.com/ollama/ollama v0.18.1 // indirect
	github.com/qdrant/go-client v1.17.1 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260316180232-0b37fe3546d5 // indirect
	google.golang.org/grpc v1.79.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace dappco.re/go/cli => ../../internal/cli

replace dappco.re/go/rag => ../..
