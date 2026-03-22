package main

import (
	"context"
	"fmt"

	"github.com/hghukasyan/flowcore"
)

func main() {
	wf := flowcore.New()

	wf.Step("validate", func(ctx *flowcore.Context) error {
		fmt.Println("validate input")
		ctx.Set("order_id", "42")
		return nil
	})

	wf.Step("persist", func(ctx *flowcore.Context) error {
		id := ctx.Get("order_id")
		fmt.Println("save order", id)
		return nil
	}, flowcore.DependsOn("validate"))

	wf.Step("notify", func(ctx *flowcore.Context) error {
		fmt.Println("send notification")
		return nil
	}, flowcore.DependsOn("persist"))

	if err := wf.Run(context.Background()); err != nil {
		panic(err)
	}
	fmt.Println("done")
}
