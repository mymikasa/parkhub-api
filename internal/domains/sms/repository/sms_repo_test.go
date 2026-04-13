package repository

import (
	"context"
	"testing"
	"time"

	"github.com/parkhub/api/internal/domains/sms/domain"
	"github.com/parkhub/api/internal/domains/sms/errs"
	cachemocks "github.com/parkhub/api/internal/domains/sms/repository/cache/mocks"
	daomocks "github.com/parkhub/api/internal/domains/sms/repository/dao/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	go_mock "go.uber.org/mock/gomock"
)

func setupSmsRepo(t *testing.T) (SmsRepository, *daomocks.MockSmsRecordDAO, *cachemocks.MockSmsCache) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockDAO := daomocks.NewMockSmsRecordDAO(ctrl)
	mockCache := cachemocks.NewMockSmsCache(ctrl)
	repo := NewSmsRepository(mockDAO, mockCache)
	return repo, mockDAO, mockCache
}

func TestSmsRepository_SaveCode(t *testing.T) {
	repo, mockDAO, mockCache := setupSmsRepo(t)
	ctx := context.Background()

	code, err := domain.NewSmsCode("13800138000", domain.SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)

	mockCache.EXPECT().Store(go_mock.Any(), go_mock.Any()).Return(nil)
	mockDAO.EXPECT().Insert(go_mock.Any(), go_mock.Any()).Return(nil)

	err = repo.SaveCode(ctx, code)
	assert.NoError(t, err)
}

func TestSmsRepository_SaveSendFailure(t *testing.T) {
	repo, mockDAO, _ := setupSmsRepo(t)
	ctx := context.Background()

	mockDAO.EXPECT().Insert(go_mock.Any(), go_mock.Any()).DoAndReturn(
		func(_ context.Context, record interface{}) error {
			return nil
		},
	)

	err := repo.SaveSendFailure(ctx, "13800138000", domain.SmsPurposeLogin, "gateway timeout")
	assert.NoError(t, err)
}

func TestSmsRepository_GetCode(t *testing.T) {
	repo, _, mockCache := setupSmsRepo(t)
	ctx := context.Background()

	code, err := domain.NewSmsCode("13800138000", domain.SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)

	mockCache.EXPECT().Retrieve(go_mock.Any(), "13800138000", domain.SmsPurposeLogin).Return(code, nil)

	result, err := repo.GetCode(ctx, "13800138000", domain.SmsPurposeLogin)
	require.NoError(t, err)
	assert.Equal(t, code.Code, result.Code)
}

func TestSmsRepository_GetCode_NotFound(t *testing.T) {
	repo, _, mockCache := setupSmsRepo(t)
	ctx := context.Background()

	mockCache.EXPECT().Retrieve(go_mock.Any(), "13800138000", domain.SmsPurposeLogin).Return(nil, errs.ErrCodeNotFound)

	_, err := repo.GetCode(ctx, "13800138000", domain.SmsPurposeLogin)
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}

func TestSmsRepository_VerifyAndConsume_Success(t *testing.T) {
	repo, _, mockCache := setupSmsRepo(t)
	ctx := context.Background()

	mockCache.EXPECT().VerifyAndConsume(go_mock.Any(), "13800138000", domain.SmsPurposeLogin, "123456").Return(nil)

	err := repo.VerifyAndConsume(ctx, "13800138000", domain.SmsPurposeLogin, "123456")
	assert.NoError(t, err)
}

func TestSmsRepository_VerifyAndConsume_Mismatch(t *testing.T) {
	repo, _, mockCache := setupSmsRepo(t)
	ctx := context.Background()

	mockCache.EXPECT().VerifyAndConsume(go_mock.Any(), "13800138000", domain.SmsPurposeLogin, "000000").Return(errs.ErrCodeMismatch)

	err := repo.VerifyAndConsume(ctx, "13800138000", domain.SmsPurposeLogin, "000000")
	assert.ErrorIs(t, err, errs.ErrCodeMismatch)
}

func TestSmsRepository_VerifyAndConsume_NotFound(t *testing.T) {
	repo, _, mockCache := setupSmsRepo(t)
	ctx := context.Background()

	mockCache.EXPECT().VerifyAndConsume(go_mock.Any(), "13800138000", domain.SmsPurposeLogin, "123456").Return(errs.ErrCodeNotFound)

	err := repo.VerifyAndConsume(ctx, "13800138000", domain.SmsPurposeLogin, "123456")
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}

func TestSmsRepository_SetRateLimit(t *testing.T) {
	repo, _, mockCache := setupSmsRepo(t)
	ctx := context.Background()

	mockCache.EXPECT().SetRateLimit(go_mock.Any(), "13800138000", 60*time.Second).Return(nil)

	err := repo.SetRateLimit(ctx, "13800138000", 60*time.Second)
	assert.NoError(t, err)
}

func TestSmsRepository_CheckRateLimit(t *testing.T) {
	repo, _, mockCache := setupSmsRepo(t)
	ctx := context.Background()

	mockCache.EXPECT().CheckRateLimit(go_mock.Any(), "13800138000").Return(true, nil)

	limited, err := repo.CheckRateLimit(ctx, "13800138000")
	require.NoError(t, err)
	assert.True(t, limited)
}
