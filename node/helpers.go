package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rollkit/rollkit/pkg/p2p/key"
)

// Source is an enum representing different sources of height
type Source int

const (
	// Header is the source of height from the header service
	Header Source = iota
	// Block is the source of height from the block service
	Block
	// Store is the source of height from the block manager store
	Store
)

// MockTester is a mock testing.T
type MockTester struct {
}

// Fail is used to fail the test
func (m MockTester) Fail() {}

// FailNow is used to fail the test immediately
func (m MockTester) FailNow() {}

// Logf is used to log a message to the test logger
func (m MockTester) Logf(format string, args ...any) {}

// Errorf is used to log an error to the test logger
func (m MockTester) Errorf(format string, args ...any) {}

func waitForFirstBlock(node Node, source Source) error {
	return waitForAtLeastNBlocks(node, 1, source)
}

func waitForFirstBlockToBeDAIncludedHeight(node Node) error {
	return waitForAtLeastNDAIncludedHeight(node, 1)
}

func getNodeHeight(node Node, source Source) (uint64, error) {
	switch source {
	case Header:
		return getNodeHeightFromHeader(node)
	case Block:
		return getNodeHeightFromBlock(node)
	case Store:
		return getNodeHeightFromStore(node)
	default:
		return 0, errors.New("invalid source")
	}
}

func getNodeHeightFromHeader(node Node) (uint64, error) {
	if fn, ok := node.(*FullNode); ok {
		return fn.hSyncService.Store().Height(), nil
	}
	if ln, ok := node.(*LightNode); ok {
		return ln.hSyncService.Store().Height(), nil
	}
	return 0, errors.New("not a full or light node")
}

func getNodeHeightFromBlock(node Node) (uint64, error) {
	if fn, ok := node.(*FullNode); ok {
		return fn.dSyncService.Store().Height(), nil
	}
	return 0, errors.New("not a full node")
}

func getNodeHeightFromStore(node Node) (uint64, error) {
	if fn, ok := node.(*FullNode); ok {
		height, err := fn.blockManager.GetStoreHeight(context.Background())
		return height, err
	}
	return 0, errors.New("not a full node")
}

//nolint:unused
func safeClose(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// waitForAtLeastNBlocks waits for the node to have at least n blocks
func waitForAtLeastNBlocks(node Node, n uint64, source Source) error {
	return Retry(300, 100*time.Millisecond, func() error {
		nHeight, err := getNodeHeight(node, source)
		if err != nil {
			return err
		}
		if nHeight >= n {
			return nil
		}
		return fmt.Errorf("expected height > %v, got %v", n, nHeight)
	})
}

// waitForAtLeastNDAIncludedHeight waits for the DA included height to be at least n
func waitForAtLeastNDAIncludedHeight(node Node, n uint64) error {
	return Retry(300, 100*time.Millisecond, func() error {
		nHeight := node.(*FullNode).blockManager.GetDAIncludedHeight()
		if nHeight == 0 {
			return fmt.Errorf("waiting for DA inclusion")
		}
		if nHeight >= n {
			return nil
		}
		return fmt.Errorf("expected height > %v, got %v", n, nHeight)
	})
}

// Retry attempts to execute the provided function up to the specified number of tries,
// with a delay between attempts. It returns nil if the function succeeds, or the last
// error encountered if all attempts fail.
//
// Parameters:
//   - tries: The maximum number of attempts to make
//   - durationBetweenAttempts: The duration to wait between attempts
//   - fn: The function to retry, which returns an error on failure
//
// Returns:
//   - error: nil if the function succeeds, or the last error encountered
func Retry(tries int, durationBetweenAttempts time.Duration, fn func() error) (err error) {
	for i := 1; i <= tries-1; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(durationBetweenAttempts)
	}
	return fn()
}

// InitFiles initializes the files for the node.
// It creates a configuration directory and generates a node key.
// It returns the generated node key and an error if any occurs during the process.
func InitFiles(dir string) (*key.NodeKey, error) {
	// Create config directory
	configDir := filepath.Join(dir, "config")
	err := os.MkdirAll(configDir, 0700) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// create the nodekey file
	nodeKey, err := key.LoadOrGenNodeKey(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create node key: %w", err)
	}

	return nodeKey, nil
}
