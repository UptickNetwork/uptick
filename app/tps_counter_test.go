package app

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/log"
)

func TestTPSCounter(t *testing.T) {
	t.Skip("FIXME: non deterministic")

	buf := new(bytes.Buffer)
	wlog := &writerLogger{w: buf}
	tpc := newTPSCounter(wlog)
	tpc.reportPeriod = 5 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	go tpc.start(ctx)

	// Concurrently increment the counter.
	n := 50
	repeat := 5
	go func() {
		defer cancel()
		for i := 0; i < repeat; i++ {
			for j := 0; j < n; j++ {
				if j&1 == 0 {
					tpc.incrementSuccess()
				} else {
					tpc.incrementFailure()
				}
			}
			<-time.After(tpc.reportPeriod)
		}
	}()

	<-ctx.Done()
	<-tpc.doneCh

	// We expect that the TPS reported will be:
	// 100 / 5ms => 100 / 0.005s = 20,000 TPS
	lines := strings.Split(buf.String(), "\n")
	require.True(t, len(lines) > 1, "Expected at least 1 line")
	wantReg := regexp.MustCompile(`Transactions per second tps \d+\.\d+`)
	matches := wantReg.FindAllString(buf.String(), -1)
	require.Equal(t, 5, len(matches))
	wantTotalTPS := float64(len(matches)) * float64(n) / (float64(tpc.reportPeriod) / float64(time.Second))
	require.Equal(t, wantTotalTPS, wlog.nTotalTPS, "Mismatched total TPS")
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

	wl.nTotalTPS += keyVals[1].(float64)
	fmt.Fprintf(wl.w, msg+" "+fmt.Sprintf("%s %.2f\n", keyVals[0], keyVals[1]))
}

func TestTAddress(t *testing.T) {

	fmt.Printf("test address ..\n")

	//// cosmos -> evm
	//rawBytes, err := sdk.GetFromBech32("uptick1n3t0zuwq4u47ke48qm3pfhj96f4ujhs70f52sg", "uptick")
	//if err != nil {
	//	fmt.Printf("error %v\n", err)
	//}x
	//
	//fmt.Printf("normal : %v\n", rawBytes)
	//encodedString := hex.EncodeToString(rawBytes)
	//fmt.Println("Encoded Hex String1: ", encodedString)

	//rawBytes2, err2 := hex.DecodeString("0x9c56F171C0aF2beB66a706e214DE45D26Bc95e1e"[2:])
	//if err2 != nil {
	//	fmt.Printf("error %v\n", err2)
	//}
	//fmt.Println("Encoded Hex String2: ", rawBytes2)
	//
	//// evm -> cosmos
	//strAddress, err3 := sdk.Bech32ifyAddressBytes("uptick", rawBytes2)
	//if err3 != nil {
	//	fmt.Printf("error %v\n", err3)
	//}
	//fmt.Println("Encoded Hex String3: ", strAddress)
	//

	// var test [32]byte
	byteArray := []byte{209, 100, 252, 60, 37, 120, 9, 212, 159, 213, 157, 170, 117, 182, 27, 24, 41, 132, 9, 108, 182, 135, 211, 148, 92, 128, 168, 178, 203, 26, 73, 65}
	result := hex.EncodeToString(byteArray)
	fmt.Printf("Encoded Hex result: %v \n", result)

}
