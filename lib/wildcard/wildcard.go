package wildcard

// 实现了一个基于通配符的模式匹配，将通配符模式字符串转换为模式结构体，并提供了模式匹配的功能
// 通配符模式字符串的语法如下：
// * 匹配任意数量的任意字符
// ? 匹配任意单个字符
// [abc] 匹配 a 或 b 或 c
// [a-z] 匹配 a 到 z 的任意单个字符
// [^a] 匹配除了 a 以外的任意单个字符

const (
	normal     = iota
	all        // *
	any        // ?
	setSymbol  // []
	rangSymbol // [a-b]
	negSymbol  // [^a]
)

type item struct {
	character byte
	set       map[byte]bool
	typeCode  int
}

func (i *item) contains(c byte) bool {
	if i.typeCode == setSymbol {
		_, ok := i.set[c]
		return ok
	} else if i.typeCode == rangSymbol {
		if _, ok := i.set[c]; ok {
			return true
		}
		var (
			min uint8 = 255
			max uint8 = 0
		)
		for k := range i.set {
			if min > k {
				min = k
			}
			if max < k {
				max = k
			}
		}
		return c >= min && c <= max
	} else {
		_, ok := i.set[c]
		return !ok
	}
}

// Pattern represents a wildcard pattern
// 表示一个通配符模式
type Pattern struct {
	items []*item
}

// CompilePattern convert wildcard string to Pattern
// 将通配符模式字符串转换为模式结构体
func CompilePattern(src string) *Pattern {
	items := make([]*item, 0)
	escape := false
	inSet := false
	var set map[byte]bool
	for _, v := range src {
		c := byte(v)
		if escape {
			items = append(items, &item{typeCode: normal, character: c})
			escape = false
		} else if c == '*' {
			items = append(items, &item{typeCode: all})
		} else if c == '?' {
			items = append(items, &item{typeCode: any})
		} else if c == '\\' {
			escape = true
		} else if c == '[' {
			if !inSet {
				inSet = true
				set = make(map[byte]bool)
			} else {
				set[c] = true
			}
		} else if c == ']' {
			if inSet {
				inSet = false
				typeCode := setSymbol
				if _, ok := set['-']; ok {
					typeCode = rangSymbol
					delete(set, '-')
				}
				if _, ok := set['^']; ok {
					typeCode = negSymbol
					delete(set, '^')
				}
				items = append(items, &item{typeCode: typeCode, set: set})
			} else {
				items = append(items, &item{typeCode: normal, character: c})
			}
		} else {
			if inSet {
				set[c] = true
			} else {
				items = append(items, &item{typeCode: normal, character: c})
			}
		}
	}
	return &Pattern{
		items: items,
	}
}

// IsMatch returns whether the given string matches pattern
// 判断给定的字符串是否匹配模式
func (p *Pattern) IsMatch(s string) bool {
	if len(p.items) == 0 {
		return len(s) == 0
	}
	m := len(s)
	n := len(p.items)
	table := make([][]bool, m+1)
	for i := 0; i < m+1; i++ {
		table[i] = make([]bool, n+1)
	}
	table[0][0] = true
	for j := 1; j < n+1; j++ {
		table[0][j] = table[0][j-1] && p.items[j-1].typeCode == all
	}
	for i := 1; i < m+1; i++ {
		for j := 1; j < n+1; j++ {
			if p.items[j-1].typeCode == all {
				table[i][j] = table[i-1][j] || table[i][j-1]
			} else {
				table[i][j] = table[i-1][j-1] &&
					(p.items[j-1].typeCode == any ||
						(p.items[j-1].typeCode == normal && uint8(s[i-1]) == p.items[j-1].character) ||
						(p.items[j-1].typeCode >= setSymbol && p.items[j-1].contains(s[i-1])))
			}
		}
	}
	return table[m][n]
}
