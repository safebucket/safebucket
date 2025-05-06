package tests

import (
	"context"
	"github.com/stretchr/testify/mock"
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
	providerName,
	code,
	nonce string,
) (string, string, error) {
	args := m.Called(ctx, providerName, code, nonce)
	return args.String(0), args.String(1), args.Error(2)
}
