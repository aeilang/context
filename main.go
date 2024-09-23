package main

import (
	"fmt"

	"github.com/aeilang/test/mycontext"
)

func main() {
	ctx1 := mycontext.WithValue(mycontext.Background(), 1, 1)
	ctx2 := mycontext.WithValue(ctx1, 2, 2)
	ctx3 := mycontext.WithValue(ctx2, 3, 3)

	fmt.Println(ctx3.Value(2))
}
