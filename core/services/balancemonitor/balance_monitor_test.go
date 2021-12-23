package balancemonitor_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	ethmocks "github.com/smartcontractkit/chainlink/core/services/eth/mocks"

	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/smartcontractkit/chainlink/core/assets"
	"github.com/smartcontractkit/chainlink/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/core/internal/testutils/pgtest"
	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/chainlink/core/services/balancemonitor"
)

var nilBigInt *big.Int

func newEthClientMock(t mock.TestingT) *ethmocks.Client {
	mockEth := new(ethmocks.Client)
	mockEth.Test(t)
	mockEth.On("ChainID").Maybe().Return(big.NewInt(0))
	return mockEth
}

func TestBalanceMonitor_Start(t *testing.T) {
	cfg := cltest.NewTestGeneralConfig(t)

	t.Run("updates balance from nil for multiple keys", func(t *testing.T) {
		db := pgtest.NewSqlxDB(t)
		ethKeyStore := cltest.NewKeyStore(t, db, cfg).Eth()

		ethClient := newEthClientMock(t)
		defer ethClient.AssertExpectations(t)

		_, k0Addr := cltest.MustInsertRandomKey(t, ethKeyStore, 0)
		_, k1Addr := cltest.MustInsertRandomKey(t, ethKeyStore, 0)

		bm := balancemonitor.NewBalanceMonitor(ethClient, ethKeyStore, logger.TestLogger(t))
		defer bm.Close()

		k0bal := big.NewInt(42)
		k1bal := big.NewInt(43)
		assert.Nil(t, bm.GetEthBalance(k0Addr))
		assert.Nil(t, bm.GetEthBalance(k1Addr))

		ethClient.On("BalanceAt", mock.Anything, k0Addr, nilBigInt).Once().Return(k0bal, nil)
		ethClient.On("BalanceAt", mock.Anything, k1Addr, nilBigInt).Once().Return(k1bal, nil)

		assert.NoError(t, bm.Start())

		gomega.NewWithT(t).Eventually(func() *big.Int {
			return bm.GetEthBalance(k0Addr).ToInt()
		}).Should(gomega.Equal(k0bal))
		gomega.NewWithT(t).Eventually(func() *big.Int {
			return bm.GetEthBalance(k1Addr).ToInt()
		}).Should(gomega.Equal(k1bal))
	})

	t.Run("handles nil head", func(t *testing.T) {
		db := pgtest.NewSqlxDB(t)
		ethKeyStore := cltest.NewKeyStore(t, db, cfg).Eth()

		ethClient := newEthClientMock(t)
		defer ethClient.AssertExpectations(t)

		_, k0Addr := cltest.MustInsertRandomKey(t, ethKeyStore, 0)

		bm := balancemonitor.NewBalanceMonitor(ethClient, ethKeyStore, logger.TestLogger(t))
		defer bm.Close()
		k0bal := big.NewInt(42)

		ethClient.On("BalanceAt", mock.Anything, k0Addr, nilBigInt).Once().Return(k0bal, nil)

		assert.NoError(t, bm.Start())

		gomega.NewWithT(t).Eventually(func() *big.Int {
			return bm.GetEthBalance(k0Addr).ToInt()
		}).Should(gomega.Equal(k0bal))
	})

	t.Run("recovers on error", func(t *testing.T) {
		db := pgtest.NewSqlxDB(t)
		ethKeyStore := cltest.NewKeyStore(t, db, cfg).Eth()

		ethClient := newEthClientMock(t)
		defer ethClient.AssertExpectations(t)

		_, k0Addr := cltest.MustInsertRandomKey(t, ethKeyStore, 0)

		bm := balancemonitor.NewBalanceMonitor(ethClient, ethKeyStore, logger.TestLogger(t))
		defer bm.Close()

		ethClient.On("BalanceAt", mock.Anything, k0Addr, nilBigInt).
			Once().
			Return(nil, errors.New("a little easter egg for the 4chan link marines error"))

		assert.NoError(t, bm.Start())

		gomega.NewWithT(t).Consistently(func() *big.Int {
			return bm.GetEthBalance(k0Addr).ToInt()
		}).Should(gomega.BeNil())
	})
}

func TestBalanceMonitor_OnNewLongestChain_UpdatesBalance(t *testing.T) {
	cfg := cltest.NewTestGeneralConfig(t)

	t.Run("updates balance for multiple keys", func(t *testing.T) {
		db := pgtest.NewSqlxDB(t)
		ethKeyStore := cltest.NewKeyStore(t, db, cfg).Eth()

		ethClient := newEthClientMock(t)
		defer ethClient.AssertExpectations(t)

		_, k0Addr := cltest.MustInsertRandomKey(t, ethKeyStore, 0)
		_, k1Addr := cltest.MustInsertRandomKey(t, ethKeyStore, 0)

		bm := balancemonitor.NewBalanceMonitor(ethClient, ethKeyStore, logger.TestLogger(t))
		k0bal := big.NewInt(42)
		// Deliberately larger than a 64 bit unsigned integer to test overflow
		k1bal := big.NewInt(0)
		k1bal.SetString("19223372036854776000", 10)

		head := cltest.Head(0)

		ethClient.On("BalanceAt", mock.Anything, k0Addr, nilBigInt).Once().Return(k0bal, nil)
		ethClient.On("BalanceAt", mock.Anything, k1Addr, nilBigInt).Once().Return(k1bal, nil)

		require.NoError(t, bm.Start())
		defer bm.Close()

		ethClient.AssertExpectations(t)

		ethClient.On("BalanceAt", mock.Anything, k0Addr, nilBigInt).Once().Return(k0bal, nil)
		ethClient.On("BalanceAt", mock.Anything, k1Addr, nilBigInt).Once().Return(k1bal, nil)

		// Do the thing
		bm.OnNewLongestChain(context.TODO(), head)

		gomega.NewWithT(t).Eventually(func() *big.Int {
			return bm.GetEthBalance(k0Addr).ToInt()
		}).Should(gomega.Equal(k0bal))
		gomega.NewWithT(t).Eventually(func() *big.Int {
			return bm.GetEthBalance(k1Addr).ToInt()
		}).Should(gomega.Equal(k1bal))

		// Do it again
		k0bal2 := big.NewInt(142)
		k1bal2 := big.NewInt(142)

		head = cltest.Head(1)

		ethClient.On("BalanceAt", mock.Anything, k0Addr, nilBigInt).Once().Return(k0bal2, nil)
		ethClient.On("BalanceAt", mock.Anything, k1Addr, nilBigInt).Once().Return(k1bal2, nil)

		bm.OnNewLongestChain(context.TODO(), head)

		gomega.NewWithT(t).Eventually(func() *big.Int {
			return bm.GetEthBalance(k0Addr).ToInt()
		}).Should(gomega.Equal(k0bal2))
		gomega.NewWithT(t).Eventually(func() *big.Int {
			return bm.GetEthBalance(k1Addr).ToInt()
		}).Should(gomega.Equal(k1bal2))

		ethClient.AssertExpectations(t)
	})
}

func TestBalanceMonitor_FewerRPCCallsWhenBehind(t *testing.T) {
	db := pgtest.NewSqlxDB(t)
	cfg := cltest.NewTestGeneralConfig(t)
	ethKeyStore := cltest.NewKeyStore(t, db, cfg).Eth()

	cltest.MustAddRandomKeyToKeystore(t, ethKeyStore)

	ethClient := newEthClientMock(t)

	bm := balancemonitor.NewBalanceMonitor(ethClient, ethKeyStore, logger.TestLogger(t))
	ethClient.On("BalanceAt", mock.Anything, mock.Anything, mock.Anything).
		Once().
		Return(big.NewInt(1), nil)
	require.NoError(t, bm.Start())

	head := cltest.Head(0)

	// Only expect this twice, even though 10 heads will come in
	mockUnblocker := make(chan time.Time)
	ethClient.On("BalanceAt", mock.Anything, mock.Anything, mock.Anything).
		WaitUntil(mockUnblocker).
		Once().
		Return(big.NewInt(42), nil)
	// This second call is Maybe because the SleeperTask may not have started
	// before we call `OnNewLongestChain` 10 times, in which case it's only
	// executed once
	var callCount atomic.Int32
	ethClient.On("BalanceAt", mock.Anything, mock.Anything, mock.Anything).
		Run(func(mock.Arguments) { callCount.Inc() }).
		Maybe().
		Return(big.NewInt(42), nil)

	// Do the thing multiple times
	for i := 0; i < 10; i++ {
		bm.OnNewLongestChain(context.TODO(), head)
	}

	// Unblock the first mock
	cltest.CallbackOrTimeout(t, "FewerRPCCallsWhenBehind unblock BalanceAt", func() {
		mockUnblocker <- time.Time{}
	})

	bm.Close()

	// Make sure the BalanceAt mock wasn't called more than once
	assert.LessOrEqual(t, callCount.Load(), int32(1))

	ethClient.AssertExpectations(t)
}

func Test_ApproximateFloat64(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      float64
		wantError bool
	}{
		{"zero", "0", 0, false},
		{"small", "1", 0.000000000000000001, false},
		{"rounding", "12345678901234567890", 12.345678901234567, false},
		{"large", "123456789012345678901234567890", 123456789012.34567, false},
		{"extreme", "1234567890123456789012345678901234567890123456789012345678901234567890", 1.2345678901234568e+51, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			eth := assets.NewEth(0)
			eth.SetString(test.input, 10)
			float, err := balancemonitor.ApproximateFloat64(eth)
			require.NoError(t, err)
			require.Equal(t, test.want, float)
		})
	}
}
