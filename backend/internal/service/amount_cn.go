package service

// AmountToCN 将浮点数金额转换为中文大写（最大支持万亿级）
// 例如：12345.67 → "壹万贰仟叁佰肆拾伍元陆角柒分"
func AmountToCN(amount float64) string {
	if amount < 0 {
		return "负" + AmountToCN(-amount)
	}
	if amount == 0 {
		return "零元整"
	}
	// 整数部分
	intPart := int64(amount)
	// 小数部分（保留两位）
	decPart := int64(amount*100+0.5) % 100

	s := ""
	if intPart > 0 {
		s += numToCN(intPart) + "元"
	} else {
		s += "零元"
	}

	if decPart == 0 {
		s += "整"
		return s
	}

	jiao := decPart / 10
	fen := decPart % 10
	if jiao > 0 {
		s += digitCN[jiao] + "角"
	}
	if fen > 0 {
		s += digitCN[fen] + "分"
	}
	return s
}

var digitCN = []string{"零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"}

// numToCN 整数字转中文（0~万亿）
func numToCN(n int64) string {
	if n == 0 {
		return "零"
	}

	units := []string{"", "拾", "佰", "仟"}
	sections := []string{"", "万", "亿", "万"}

	var result string
	sectionIdx := 0
	prevZero := false

	for n > 0 {
		section := int(n % 10000)
		n /= 10000

		if section == 0 {
			// 空段：如果之前有非零段，添加零
			if result != "" && !prevZero {
				result = "零" + result
				prevZero = true
			}
			sectionIdx++
			continue
		}

		prevZero = false
		sectionStr := ""
		zeroFlag := false
		for i := 0; i < 4 && section > 0; i++ {
			digit := section % 10
			section /= 10
			if digit == 0 {
				if i == 0 {
					// 个位为 0，不显示
				} else if !zeroFlag {
					sectionStr = "零" + sectionStr
					zeroFlag = true
				}
			} else {
				sectionStr = digitCN[digit] + units[i] + sectionStr
				zeroFlag = false
			}
		}
		if sectionIdx > 0 {
			sectionStr += sections[sectionIdx]
		}
		result = sectionStr + result
		sectionIdx++
	}

	return result
}
