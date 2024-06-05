package cdc

import (
	"context"
	"encoding/json"
	"fmt"

	"entgo.io/ent"
)

type Sink interface {
	NewTransaction() TransactionSink
}

type TransactionSink interface {
	Commit()
}

type MutatorFunc func(next ent.Mutator) ent.Mutator

type MutationSink interface {
	MutatorFunc() MutatorFunc
}

func NewMutationSink() MutationSink {
	return &mutationSink{}
}

var _ ent.Mutation

type mutationSink struct {
}

func (m *mutationSink) MutatorFunc() MutatorFunc {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mut ent.Mutation) (ent.Value, error) {
			m.mutationToJSON(mut)
			return next.Mutate(ctx, mut)
		})
	}
}

type cdcEntry struct {
	Op     string                 `json:"op"`
	IDs    []any                  `json:"ids"`
	Type   string                 `json:"type"` // e.g. Ledger
	Fields map[string]interface{} `json:"fields"`
}

type idGetter interface {
	CDCIDs(context.Context) ([]any, error)
}

type StringIDsGetter interface {
	IDs() ([]string, bool)
}

func (m *mutationSink) mutationToJSON(mut ent.Mutation) {
	res := cdcEntry{
		Op:     mut.Op().String(),
		Type:   mut.Type(),
		Fields: make(map[string]interface{}),
	}
	// report creation
	if idsGetter, ok := mut.(idGetter); ok {
		// TODO: what's the context here
		if ids, err := idsGetter.CDCIDs(context.Background()); err == nil {
			res.IDs = ids
		}

		// TODO: error handling
	}

	for _, edges := range mut.AddedEdges() {
		fmt.Println("AddedEdges: ", edges)
	}

	for _, field := range mut.Fields() {
		value, ok := mut.Field(field)
		if !ok {
			res.Fields[field] = nil
			continue
		}
		res.Fields[field] = value
	}

	for _, field := range mut.AddedFields() {
		value, ok := mut.AddedField(field)
		if !ok {
			res.Fields[field] = nil
			continue
		}
		res.Fields[field] = value
	}

	for _, field := range mut.ClearedFields() {
		res.Fields[field] = nil
	}

	enc, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}

	fmt.Println("CDC: ", string(enc))
}

/*
type EntCommiter[Tx any] interface {
	Commit(context.Context, *Tx) error
}

func CommitHook[Tx any](s TransactionSink) EntCommiter[Tx] {
	return EntCommiter[Tx](func(ctx context.Context, tx *Tx) error {
		err := s.Commit(ctx, tx)
		if err != nil {
			s.Commit()
		}
		return err
	})

}
*/
