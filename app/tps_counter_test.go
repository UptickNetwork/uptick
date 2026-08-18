package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
)

func TestTPSCounterIncrements(t *testing.T) {
	tpc := newTPSCounter(log.NewNopLogger())
	tpc.incrementSuccess()
	tpc.incrementSuccess()
	tpc.incrementFailure()
	require.Equal(t, uint64(2), tpc.nSuccessful)
	require.Equal(t, uint64(1), tpc.NFailed)
}

func TestTPSCounterRecordValue(t *testing.T) {
	tpc := newTPSCounter(log.NewNopLogger())
	ctx := context.Background()

	n, err := tpc.recordValue(ctx, 10, 0, statusSuccess)
	require.NoError(t, err)
	require.Equal(t, int64(10), n)

	n, err = tpc.recordValue(ctx, 3, 10, statusFailure)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
}

func TestTPSCounterReports(t *testing.T) {
	buf := new(bytes.Buffer)
	wlog := &writerLogger{w: buf}
	tpc := newTPSCounter(wlog)
	tpc.reportPeriod = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = tpc.start(ctx) }()

	for i := 0; i < 10; i++ {
		tpc.incrementSuccess()
	}

	require.Eventually(t, func() bool {
		wlog.mu.Lock()
		defer wlog.mu.Unlock()
		return strings.Contains(buf.String(), "Transactions per second")
	}, time.Second, 5*time.Millisecond)

	cancel()
	<-tpc.doneCh

	wantReg := regexp.MustCompile(`Transactions per second tps \d+\.\d+`)
	require.Regexp(t, wantReg, buf.String())
	require.Greater(t, wlog.nTotalTPS, 0.0)
}

type writerLogger struct {
	nTotalTPS float64
	mu        sync.Mutex
	w         io.Writer
	log.Logger
}

var _ log.Logger = (*writerLogger)(nil)

func (wl *writerLogger) Info(msg string, keyVals ...interface{}) {
	wl.mu.Lock()
	defer wl.mu.Unlock()

	if len(keyVals) < 2 {
		return
	}
	tps, ok := keyVals[1].(float64)
	if !ok {
		return
	}
	wl.nTotalTPS += tps
	fmt.Fprintf(wl.w, "%s %s %.2f\n", msg, keyVals[0], tps)
}
