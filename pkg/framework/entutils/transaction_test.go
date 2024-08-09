package entutils_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	db1 "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db"
	db2 "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent2/db"
)

// db1Adapter and db2Adapter implement the generic SomeDB interface as DB adapters
// and implement the entutils.TxCreator and entutils.TxUser interfaces to allow for transaction handling

type SomeDB[T any] interface {
	Get(ctx context.Context, id string) (*T, error)
	Save(ctx context.Context, value *T) (*T, error)
}

type SomeDBTx[T any] interface {
	SomeDB[T]
	entutils.TxCreator
	entutils.TxUser[SomeDB[T]]
}

type db1Adapter struct {
	db *db1.Client
}

func (d *db1Adapter) Get(ctx context.Context, id string) (*db1.Example1, error) {
	return d.db.Example1.Get(ctx, id)
}

func (d *db1Adapter) Save(ctx context.Context, value *db1.Example1) (*db1.Example1, error) {
	return d.db.Example1.Create().
		SetID(value.ID).
		SetExampleValue1(value.ExampleValue1).
		Save(ctx)
}

// we have to implement the TxCreator and TxUser interfaces
func (d *db1Adapter) Tx(ctx context.Context) (context.Context, *entutils.TxDriver, error) {
	txCtx, rawConfig, eDriver, err := d.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (d *db1Adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) SomeDB[db1.Example1] {
	txClient := db1.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	res := &db1Adapter{db: txClient.Client()}
	return res
}

var _ SomeDBTx[db1.Example1] = &db1Adapter{}

type db2Adapter struct {
	db *db2.Client
}

func (d *db2Adapter) Get(ctx context.Context, id string) (*db2.Example2, error) {
	return d.db.Example2.Get(ctx, id)
}

func (d *db2Adapter) Save(ctx context.Context, value *db2.Example2) (*db2.Example2, error) {
	return d.db.Example2.Create().
		SetID(value.ID).
		SetExampleValue2(value.ExampleValue2).
		Save(ctx)
}

// we have to implement the TxCreator and TxUser interfaces
func (d *db2Adapter) Tx(ctx context.Context) (context.Context, *entutils.TxDriver, error) {
	txCtx, rawConfig, eDriver, err := d.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (d *db2Adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) SomeDB[db2.Example2] {
	txClient := db2.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return &db2Adapter{db: txClient.Client()}
}

var _ SomeDBTx[db2.Example2] = &db2Adapter{}

func TestTransaction(t *testing.T) {
	tc := []struct {
		name string
		run  func(t *testing.T, db1Adapter SomeDBTx[db1.Example1], db2Adapter SomeDBTx[db2.Example2])
	}{
		{
			name: "Should roll back everything manually",
			run: func(t *testing.T, db1Adapter SomeDBTx[db1.Example1], db2Adapter SomeDBTx[db2.Example2]) {
				ctx := context.Background()
				var ent1Id string
				var ent2Id string
				_, err := entutils.StartAndRunTx(ctx, db1Adapter, func(ctx context.Context, tx *entutils.TxDriver) (*interface{}, error) {
					// create entities
					ec1, err := db1Adapter.WithTx(ctx, tx).Save(ctx, &db1.Example1{
						ID:            "1",
						ExampleValue1: "value1",
					})
					if err != nil {
						return nil, err
					}
					ec2, err := db2Adapter.WithTx(ctx, tx).Save(ctx, &db2.Example2{
						ID:            "2",
						ExampleValue2: "value2",
					})
					if err != nil {
						return nil, err
					}

					// save it's id for later
					ent1Id = ec1.ID
					ent2Id = ec2.ID

					// check it exists in transaction
					ent1, err := db1Adapter.WithTx(ctx, tx).Get(ctx, ent1Id)
					assert.NoError(t, err)
					assert.NotNil(t, ent1)
					ent2, err := db2Adapter.WithTx(ctx, tx).Get(ctx, ent2Id)
					assert.NoError(t, err)
					assert.NotNil(t, ent2)

					// roll back
					err = tx.Rollback()
					assert.NoError(t, err)
					return nil, nil
				})
				if err != nil {
					t.Fatalf("failed to run transaction %s", err)
				}

				// check that it wasn't persisted
				ent1, err := db1Adapter.Get(ctx, ent1Id)
				assert.Error(t, err)
				assert.Nil(t, ent1)
				ent2, err := db2Adapter.Get(ctx, ent2Id)
				assert.Error(t, err)
				assert.Nil(t, ent2)
			},
		},
		{
			name: "Should commit everything by default",
			run: func(t *testing.T, db1Adapter SomeDBTx[db1.Example1], db2Adapter SomeDBTx[db2.Example2]) {
				ctx := context.Background()
				var ent1Id string
				var ent2Id string
				_, err := entutils.StartAndRunTx(ctx, db1Adapter, func(ctx context.Context, tx *entutils.TxDriver) (*interface{}, error) {
					// create entities
					ec1, err := db1Adapter.WithTx(ctx, tx).Save(ctx, &db1.Example1{
						ID:            "1",
						ExampleValue1: "value1",
					})
					if err != nil {
						return nil, err
					}
					ec2, err := db2Adapter.WithTx(ctx, tx).Save(ctx, &db2.Example2{
						ID:            "2",
						ExampleValue2: "value2",
					})
					if err != nil {
						return nil, err
					}

					// save it's id for later
					ent1Id = ec1.ID
					ent2Id = ec2.ID

					// check it exists in transaction
					ent1, err := db1Adapter.WithTx(ctx, tx).Get(ctx, ent1Id)
					assert.NoError(t, err)
					assert.NotNil(t, ent1)

					ent2, err := db2Adapter.WithTx(ctx, tx).Get(ctx, ent2Id)
					assert.NoError(t, err)
					assert.NotNil(t, ent2)

					return nil, nil
				})
				if err != nil {
					t.Fatalf("failed to run transaction %s", err)
				}

				// check that it was persisted
				ent1, err := db1Adapter.Get(ctx, ent1Id)
				assert.NoError(t, err)
				ent2, err := db2Adapter.Get(ctx, ent2Id)
				assert.NoError(t, err)

				assert.NotNil(t, ent1)
				assert.Equal(t, ent1Id, ent1.ID)

				assert.NotNil(t, ent2)
				assert.Equal(t, ent2Id, ent2.ID)
			},
		},
		{
			name: "Should roll back everything if context is canceled",
			run: func(t *testing.T, db1Adapter SomeDBTx[db1.Example1], db2Adapter SomeDBTx[db2.Example2]) {
				ctx, cancel := context.WithCancel(context.Background())
				var ent1Id string
				var ent2Id string

				wg := sync.WaitGroup{}
				ch := make(chan bool)

				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := entutils.StartAndRunTx(ctx, db1Adapter, func(ctx context.Context, tx *entutils.TxDriver) (*interface{}, error) {
						// create entities
						ec1, err := db1Adapter.WithTx(ctx, tx).Save(ctx, &db1.Example1{
							ID:            "1",
							ExampleValue1: "value1",
						})
						if err != nil {
							return nil, err
						}
						ec2, err := db2Adapter.WithTx(ctx, tx).Save(ctx, &db2.Example2{
							ID:            "2",
							ExampleValue2: "value2",
						})
						if err != nil {
							return nil, err
						}

						// save it's id for later
						ent1Id = ec1.ID
						ent2Id = ec2.ID

						// check it exists in transaction
						ent1, err := db1Adapter.WithTx(ctx, tx).Get(ctx, ent1Id)
						assert.NoError(t, err)
						assert.NotNil(t, ent1)

						ent2, err := db2Adapter.WithTx(ctx, tx).Get(ctx, ent2Id)
						assert.NoError(t, err)
						assert.NotNil(t, ent2)

						// we write to the channel to signify that we have written
						ch <- true

						// we wait to simulate some other code in the transaction
						time.Sleep(100 * time.Millisecond)

						return nil, nil
					})
					assert.ErrorContains(t, err, "transaction has already been committed or rolled back")
				}()

				// we cancel the context after the writes have finished
				wg.Add(1)
				go func() {
					defer wg.Done()
					// we wait for the channel signifying that the other routine has written
					<-ch
					cancel()
				}()

				wg.Wait()

				// check that it was rolled back
				ent1, err := db1Adapter.Get(context.TODO(), ent1Id)
				assert.Error(t, err)
				assert.Nil(t, ent1)
				ent2, err := db2Adapter.Get(context.TODO(), ent2Id)
				assert.Error(t, err)
				assert.Nil(t, ent2)
			},
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// create isolated pg db for tests
			driver := testutils.InitPostgresDB(t)
			defer driver.EntDriver.Close()

			// build db clients
			db1Client := db1.NewClient(db1.Driver(driver.EntDriver))
			db2Client := db2.NewClient(db2.Driver(driver.EntDriver))

			if err := db1Client.Schema.Create(context.Background()); err != nil {
				t.Fatalf("failed to migrate database %s", err)
			}
			if err := db2Client.Schema.Create(context.Background()); err != nil {
				t.Fatalf("failed to migrate database %s", err)
			}

			db1 := &db1Adapter{db: db1Client}
			db2 := &db2Adapter{db: db2Client}

			tt.run(t, db1, db2)
		})
	}
}
