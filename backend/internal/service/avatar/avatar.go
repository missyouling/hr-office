// Package avatar 提供用户默认头像的纯函数生成逻辑。
// 所有函数均为纯函数：同一输入必然产生同一输出，便于单元测试与跨实例稳定。
package avatar

import (
	"fmt"
	"hash/fnv"
	"html"
	"strings"
	"unicode/utf8"
)

// 固定调色板：柔和背景色 + 白色前景，按种子哈希稳定选取。
// 颜色数量固定，新增颜色会改变既有用户的默认头像配色，需谨慎。
var palette = []struct{ bg, fg string }{
	{"#5B8DEF", "#FFFFFF"},
	{"#F2994A", "#FFFFFF"},
	{"#27AE60", "#FFFFFF"},
	{"#9B51E0", "#FFFFFF"},
	{"#EB5757", "#FFFFFF"},
	{"#2D9CDB", "#FFFFFF"},
	{"#F2C94C", "#FFFFFF"},
	{"#6FCF97", "#FFFFFF"},
}

// GenerateSeed 基于用户 ID 与用户名生成稳定种子（FNV-1a 哈希）。
// 同一用户（ID+用户名不变）在任何实例、任何时间生成的种子一致。
func GenerateSeed(userID uint, username string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%s", userID, username)))
	return fmt.Sprintf("%x", h.Sum64())
}

// Initial 提取显示名的首字母（UTF-8 安全）。
// 优先取第一个字符；空串或纯空白返回 "?" 兜底。
func Initial(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	r, _ := utf8.DecodeRuneInString(name)
	return string(r)
}

// DefaultSVG 生成 256x256 默认头像 SVG（纯函数）。
// 同一 seed + initial 必然输出完全相同的 SVG 字符串。
// initial 会做 HTML/XML 转义，防止特殊字符破坏 SVG 结构。
func DefaultSVG(seed, initial string) string {
	idx := int(fnv64(seed) % uint64(len(palette)))
	color := palette[idx]
	escaped := html.EscapeString(initial)
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256" viewBox="0 0 256 256">
  <rect width="256" height="256" fill="%s"/>
  <text x="128" y="128" font-family="Arial, Helvetica, sans-serif" font-size="112" fill="%s" text-anchor="middle" dominant-baseline="central">%s</text>
</svg>`, color.bg, color.fg, escaped)
}

// fnv64 计算字符串的 FNV-1a 64 位哈希
func fnv64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
