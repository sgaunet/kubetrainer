package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sgaunet/kubetrainer/internal/database"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

// TestWaitDBHandlesCtx tests the WaitForDB function with a context
// that is cancelled before the database is ready
// This test must be the first one to run
func TestWaitDBHandlesCtx(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	err := database.WaitForDB(ctx, pgdsn.String())
	assert.NotNil(t, err)
}

func TestWaitDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err := database.WaitForDB(ctx, pgdsn.String())
	assert.Nil(t, err)
}

// TestInitDB tests the InitDB function
func TestInitDB(t *testing.T) {
	err := database.WaitForDB(context.Background(), pgdsn.String())
	assert.Nil(t, err)
	pg, err := database.NewPostgres(pgdsn.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	assert.NotNil(t, pg.DB)
	assert.Nil(t, err)

	err = pg.InitDB()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
}

func TestGetDB(t *testing.T) {
	err := database.WaitForDB(context.Background(), pgdsn.String())
	assert.Nil(t, err)
	pg, err := database.NewPostgres(pgdsn.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	db := pg.GetDB()
	assert.NotNil(t, db)
}
