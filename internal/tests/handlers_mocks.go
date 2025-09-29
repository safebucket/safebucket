package tests

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockOpenIDBeginFunc struct {
	mock.Mock
}

func (m *MockOpenIDBeginFunc) OpenIDBegin(providerName, state, nonce string) (string, error) {
	args := m.Called(providerName, state, nonce)
	return args.String(0), args.Error(1)
}

type MockOpenIDCallbackFunc struct {
	mock.Mock
}

func (m *MockOpenIDCallbackFunc) OpenIDCallback(
	ctx context.Context,
	logger *zap.Logger,
	providerName,
	code,
	nonce string,
) (string, string, error) {
	args := m.Called(ctx, logger, providerName, code, nonce)
	return args.String(0), args.String(1), args.Error(2)
}

type MockCreateFunc[In any, Out any] struct {
	mock.Mock
}

func (m *MockCreateFunc[In, Out]) Create(ids uuid.UUIDs, input In) (Out, error) {
	args := m.Called(ids, input)
	return args.Get(0).(Out), args.Error(1)
}

type MockGetListFunc[Out any] struct {
	mock.Mock
}

func (m *MockGetListFunc[Out]) GetList() []Out {
	args := m.Called()
	return args.Get(0).([]Out)
}

type MockGetOneFunc[Out any] struct {
	mock.Mock
}

func (m *MockGetOneFunc[Out]) GetOne(ids uuid.UUIDs) (Out, error) {
	args := m.Called(ids)
	return args.Get(0).(Out), args.Error(1)
}

type MockUpdateFunc[In any, Out any] struct {
	mock.Mock
}

func (m *MockUpdateFunc[In, Out]) Update(ids uuid.UUIDs, input In) (Out, error) {
	args := m.Called(ids, input)
	return args.Get(0).(Out), args.Error(1)
}

type MockDeleteFunc struct {
	mock.Mock
}

func (m *MockDeleteFunc) Delete(ids uuid.UUIDs) error {
	args := m.Called(ids)
	return args.Error(0)
}
