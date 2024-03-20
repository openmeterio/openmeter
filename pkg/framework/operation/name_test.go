package operation_test

// import (
// 	"context"
// 	"fmt"
//
// 	"github.com/openmeterio/openmeter/pkg/framework/operation"
// )
//
// func ExampleWithName() {
// 	op := operation.WithName("example", operation.OperationFunc[ExampleRequest, ExampleResponse](exampleOperation))
//
// 	resp, err := op.Do(context.Background(), ExampleRequest{Name: "World"})
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(resp.Greeting)
//
// 	type named interface {
// 		Name() string
// 	}
//
// 	fmt.Println(op.(named).Name())
//
// 	// Output:
// 	// Hello, World
// 	// example
// }
