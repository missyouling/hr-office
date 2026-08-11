package storage

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// ssrfWhitelist 从环境变量 STORAGE_SSRF_WHITELIST 加载（逗号分隔的 IP/CIDR）
var ssrfWhitelist []*net.IPNet

func init() {
	whitelistStr := os.Getenv("STORAGE_SSRF_WHITELIST")
	if whitelistStr == "" {
		return
	}
	for _, entry := range strings.Split(whitelistStr, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		_, cidr, err := net.ParseCIDR(entry)
		if err != nil {
			ip := net.ParseIP(entry)
			if ip != nil {
				mask := net.CIDRMask(32, 32)
				if ip.To4() == nil {
					mask = net.CIDRMask(128, 128)
				}
				cidr = &net.IPNet{IP: ip, Mask: mask}
			} else {
				continue
			}
		}
		ssrfWhitelist = append(ssrfWhitelist, cidr)
	}
}

// ValidateEndpoint 校验 URL endpoint 不指向内网/私有 IP（SSRF 防护）
// 白名单内的 IP 跳过校验
func ValidateEndpoint(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("empty hostname in endpoint: %s", rawURL)
	}
	ip := net.ParseIP(hostname)
	if ip == nil {
		// 不是 IP 地址（如域名），DNS 解析不在本层校验
		// 实际连接时由后续 HTTP 客户端做解析，此处仅校验直接 IP
		return nil
	}
	if isIPInWhitelist(ip) {
		return nil
	}
	if isRestrictedIP(ip) {
		return fmt.Errorf("SSRF blocked: endpoint %q resolves to restricted IP %s", rawURL, ip)
	}
	return nil
}

// isRestrictedIP 判断 IP 是否为受限地址（回环/私有/链路本地/组播）
func isRestrictedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// isIPInWhitelist 判断 IP 是否在白名单中
func isIPInWhitelist(ip net.IP) bool {
	for _, cidr := range ssrfWhitelist {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateStoragePath 校验存储路径不含路径遍历序列（"../" 或 "..\\"）
func ValidateStoragePath(filePath string) error {
	if strings.Contains(filePath, "..") {
		return errors.New("path traversal detected: path contains '..'")
	}
	return nil
}
