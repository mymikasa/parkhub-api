package gateway

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockSmsGateway_Send_Success(t *testing.T) {
	gw := NewMockSmsGateway()
	ctx := context.Background()

	err := gw.Send(ctx, "13800138000", "123456", "login")
	require.NoError(t, err)

	require.Len(t, gw.Calls, 1)
	assert.Equal(t, "13800138000", gw.Calls[0].Phone)
	assert.Equal(t, "123456", gw.Calls[0].Code)
	assert.Equal(t, "login", gw.Calls[0].Purpose)
}

func TestMockSmsGateway_Send_Failure(t *testing.T) {
	gw := NewMockSmsGateway()
	gw.Err = errors.New("provider unavailable")
	ctx := context.Background()

	err := gw.Send(ctx, "13800138000", "123456", "login")
	assert.EqualError(t, err, "provider unavailable")
	assert.Len(t, gw.Calls, 1)
}

func TestMockSmsGateway_Send_MultipleCalls(t *testing.T) {
	gw := NewMockSmsGateway()
	ctx := context.Background()

	require.NoError(t, gw.Send(ctx, "111", "111111", "login"))
	require.NoError(t, gw.Send(ctx, "222", "222222", "register"))

	require.Len(t, gw.Calls, 2)
	assert.Equal(t, "111", gw.Calls[0].Phone)
	assert.Equal(t, "222", gw.Calls[1].Phone)
}
