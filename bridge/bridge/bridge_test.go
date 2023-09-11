package bridge

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	wg sync.WaitGroup
)

// Receiver를 지정하지 않은 Deposit event를 Sender와 동일한 Receiver로 재 설정하여 처리하는가?
func TestEmptyReceiver(t *testing.T) {
	wg.Add(1)

	bridge := newTestBridge(t)
	go func() {
		err := bridge.Start()
		require.NoError(t, err)
		wg.Done()
	}()
	wg.Wait()
}
