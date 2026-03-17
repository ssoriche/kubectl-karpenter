package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	fmt.Println("kubectl-karpenter", version)
	os.Exit(0)
}
