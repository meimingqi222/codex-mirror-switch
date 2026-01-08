package internal

import (
	"fmt"
	"strings"
	"sync"
)

// MaskAPIKey 脱敏显示 API 密钥，只显示前4位和后4位.
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// ParallelTask 并行执行多个任务.
type ParallelTask struct {
	wg   sync.WaitGroup
	errs []error
	mu   sync.Mutex
}

// NewParallelTask 创建新的并行任务管理器.
func NewParallelTask() *ParallelTask {
	return &ParallelTask{
		errs: make([]error, 0),
	}
}

// Add 添加一个任务.
func (pt *ParallelTask) Add(fn func() error) {
	pt.wg.Add(1)
	go func() {
		defer pt.wg.Done()
		if err := fn(); err != nil {
			pt.mu.Lock()
			pt.errs = append(pt.errs, err)
			pt.mu.Unlock()
		}
	}()
}

// Wait 等待所有任务完成.
func (pt *ParallelTask) Wait() []error {
	pt.wg.Wait()
	return pt.errs
}

// CombinedError 组合多个错误.
func CombinedError(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	msgs := make([]string, len(errs))
	for i, err := range errs {
		msgs[i] = fmt.Sprintf("%d. %v", i+1, err)
	}
	return fmt.Errorf("多个错误发生:\n  %s", strings.Join(msgs, "\n  "))
}
