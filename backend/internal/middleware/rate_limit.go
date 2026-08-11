package middleware

import (
	"sync"
	"time"
)

// LoginAttempt 记录一次登录尝试的信息
type LoginAttempt struct {
	Time    time.Time // 尝试时间
	Success bool      // 是否成功
	IP      string    // 客户端 IP
}

// LoginRateLimiter 进程内内存滑动窗口登录限流器
// 规则：
//   - IP 限流：同一 IP 每分钟最多 10 次失败尝试 → 返回 429
//   - 账号锁定：同一 username 连续 5 次失败 → 锁定 15 分钟 → 返回 423
//   - 密码正确后重置该 username 和 IP 的计数
type LoginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginWindow // key = "ip:" + ip 或 "user:" + username
}

// loginWindow 记录某个维度（IP 或 username）的失败窗口
type loginWindow struct {
	failures    []time.Time // 最近失败时间戳
	lockedUntil *time.Time  // 锁定截止时间（仅 username 维度生效）
}

const (
	ipMaxFailures    = 10               // 同一 IP 每分钟最多失败次数
	ipWindowDuration = 1 * time.Minute  // IP 滑动窗口时长
	userMaxFailures  = 5                // 同一用户名连续失败次数上限
	userLockDuration = 15 * time.Minute // 账号锁定时长
)

// NewLoginRateLimiter 创建新的登录限流器
func NewLoginRateLimiter() *LoginRateLimiter {
	return &LoginRateLimiter{
		attempts: make(map[string]*loginWindow),
	}
}

// IsIPBlocked 检查指定 IP 是否触发限流
func (l *LoginRateLimiter) IsIPBlocked(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := "ip:" + ip
	return l.isBlocked(key, ipMaxFailures, ipWindowDuration)
}

// IsUserLocked 检查指定用户名是否被锁定
func (l *LoginRateLimiter) IsUserLocked(username string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := "user:" + username
	win, exists := l.attempts[key]
	if !exists {
		return false
	}

	// 检查是否在锁定期内
	if win.lockedUntil != nil && time.Now().Before(*win.lockedUntil) {
		return true
	}

	// 锁定期已过，清除锁定
	if win.lockedUntil != nil {
		win.lockedUntil = nil
		delete(l.attempts, key)
	}

	return false
}

// RecordFailure 记录一次失败尝试
func (l *LoginRateLimiter) RecordFailure(username, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// 记录 IP 维度失败
	l.recordDimFailure("ip:"+ip, now)

	// 记录用户名维度失败
	userKey := "user:" + username
	l.recordDimFailure(userKey, now)

	// 检查用户名是否达到锁定阈值
	win := l.attempts[userKey]
	if win != nil && len(win.failures) >= userMaxFailures {
		locked := now.Add(userLockDuration)
		win.lockedUntil = &locked
	}
}

// Reset 密码正确后清空指定 username 和 IP 的计数
func (l *LoginRateLimiter) Reset(username, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, "ip:"+ip)
	delete(l.attempts, "user:"+username)
}

// isBlocked 检查指定 key 是否超出失败阈值（调用方需持有 mu 锁）
func (l *LoginRateLimiter) isBlocked(key string, maxFailures int, windowDuration time.Duration) bool {
	win, exists := l.attempts[key]
	if !exists {
		return false
	}

	cutoff := time.Now().Add(-windowDuration)
	l.pruneFailures(win, cutoff)

	return len(win.failures) >= maxFailures
}

// recordDimFailure 记录一次维度失败（调用方需持有 mu 锁）
func (l *LoginRateLimiter) recordDimFailure(key string, now time.Time) {
	win, exists := l.attempts[key]
	if !exists {
		l.attempts[key] = &loginWindow{
			failures: []time.Time{now},
		}
		return
	}

	cutoff := now.Add(-ipWindowDuration)
	l.pruneFailures(win, cutoff)
	win.failures = append(win.failures, now)
}

// pruneFailures 移除窗口外的旧失败记录
func (l *LoginRateLimiter) pruneFailures(win *loginWindow, cutoff time.Time) {
	if win == nil || len(win.failures) == 0 {
		return
	}

	n := 0
	for _, t := range win.failures {
		if t.After(cutoff) {
			win.failures[n] = t
			n++
		}
	}
	win.failures = win.failures[:n]
}
