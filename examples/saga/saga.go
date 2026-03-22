package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/hghukasyan/flowcore"
)

func main() {
	wf := flowcore.New()

	wf.Step("reserve_inventory", func(ctx *flowcore.Context) error {
		fmt.Println("reserve stock")
		ctx.Set("reserved", true)
		return nil
	}, flowcore.WithCompensation(func(ctx *flowcore.Context) error {
		fmt.Println("compensate: release inventory")
		return nil
	}))

	wf.Step("charge", func(ctx *flowcore.Context) error {
		fmt.Println("charge card")
		return errors.New("payment declined")
	},
		flowcore.DependsOn("reserve_inventory"),
		flowcore.WithCompensation(func(ctx *flowcore.Context) error {
			fmt.Println("compensate: refund")
			return nil
		}),
	)

	wf.Step("ship", func(ctx *flowcore.Context) error {
		fmt.Println("ship order")
		return nil
	}, flowcore.DependsOn("charge"))

	err := wf.Run(context.Background())
	fmt.Println("workflow error:", err)
}
