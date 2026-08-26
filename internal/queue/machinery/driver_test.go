package machinery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	machineryConfig "github.com/RichardKnop/machinery/v2/config"

	"github.com/tubruk/kiyomi/internal/queue"
)

func TestMachineryDriverRegistration(t *testing.T) {
	cfg := &machineryConfig.Config{
		Broker:        "redis://localhost:6379",
		DefaultQueue:  "test_queue",
		ResultBackend: "redis://localhost:6379",
	}

	driver, err := NewDriver(cfg, "test-tag")
	require.NoError(t, err)
	assert.NotNil(t, driver)

	handler := queue.JobHandlerFunc(func(ctx context.Context, job *queue.Job) error {
		return nil
	})

	err = driver.Register("test-task", handler)
	require.NoError(t, err)

	// Verify the task is registered in the machinery server
	assert.True(t, driver.server.IsTaskRegistered("test-task"))
}
