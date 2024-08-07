// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
