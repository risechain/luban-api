//go:generate go run -modfile=../tools/go.mod github.com/ethereum/go-ethereum/cmd/abigen --abi build/TaiyiEscrow.abi --pkg escrow --type Escrow --out Escrow.go

package escrow
