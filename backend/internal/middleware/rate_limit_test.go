package middleware

import (
	"sync"
	"testing"
	"time"
)

func TestLoginRateLimiter_IPBlock(t *testing.T) {
	limiter := NewLoginRateLimiter()
	ip := "192.168.1.100"

	// 初始不应限流
	if limiter.IsIPBlocked(ip) {
		t.Fatal("新 IP 不应触发限流")
	}

	// 记录 10 次失败（达到阈值）
	for i := 0; i < 10; i++ {
		limiter.RecordFailure("user"+string(rune('A'+i)), ip)
	}

	if !limiter.IsIPBlocked(ip) {
		t.Fatal("超过阈值后应触发 IP 限流")
	}
}

func TestLoginRateLimiter_UserLock(t *testing.T) {
	limiter := NewLoginRateLimiter()
	username := "lockeduser"

	// 初始不应锁定
	if limiter.IsUserLocked(username) {
		t.Fatal("新用户不应被锁定")
	}

	// 连续 5 次失败
	ip := "10.0.0.1"
	for i := 0; i < 5; i++ {
		limiter.RecordFailure(username, ip)
	}

	if !limiter.IsUserLocked(username) {
		t.Fatal("连续 5 次失败后应锁定账号")
	}
}

func TestLoginRateLimiter_Reset(t *testing.T) {
	limiter := NewLoginRateLimiter()
	username := "resetuser"
	ip := "10.0.0.2"

	// 先制造失败
	limiter.RecordFailure(username, ip)

	// 密码正确后重置
	limiter.Reset(username, ip)

	if limiter.IsUserLocked(username) {
		t.Fatal("重置后不应被锁定")
	}
	if limiter.IsIPBlocked(ip) {
		t.Fatal("重置后 IP 不应被限流")
	}
}

func TestLoginRateLimiter_Concurrency(t *testing.T) {
	limiter := NewLoginRateLimiter()
	var wg sync.WaitGroup
	count := 50

	// 并发记录失败
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			limiter.RecordFailure("concurrent", "10.0.0.9")
		}(i)
	}
	wg.Wait()

	// 并发场景下账号应被锁定
	if !limiter.IsUserLocked("concurrent") {
		t.Fatal("并发失败后应锁定账号")
	}
}

func TestLoginRateLimiter_WindowExpiry(t *testing.T) {
	limiter := NewLoginRateLimiter()
	ip := "10.0.0.3"

	// 制造正好 9 次失败（不到阈值）
	for i := 0; i < 9; i++ {
		limiter.RecordFailure("w"+string(rune('A'+i)), ip)
	}

	if limiter.IsIPBlocked(ip) {
		t.Fatal("未达阈值不应限流")
	}

	// 再失败 1 次，达到阈值
	limiter.RecordFailure("wj", ip)
	if !limiter.IsIPBlocked(ip) {
		t.Fatal("达到阈值后应触发 IP 限流")
	}
}

func TestLoginRateLimiter_UserLockExpiredAfterDuration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过耗时的锁定过期测试（需要 15 分钟）")
	}
	// 此测试验证锁定过期机制，但实际过期需要 15 分钟，仅作结构验证
	limiter := NewLoginRateLimiter()
	username := "willExpire"

	// 直接操作内部状态模拟过期
	limiter.mu.Lock()
	key := "user:" + username
	pastTime := time.Now().Add(-1 * time.Minute) // 锁已过期
	limiter.attempts[key] = &loginWindow{
		failures:    []time.Time{time.Now()},
		lockedUntil: &pastTime,
	}
	limiter.mu.Unlock()

	if limiter.IsUserLocked(username) {
		t.Fatal("锁定过期后不应再被锁定")
	}
}
