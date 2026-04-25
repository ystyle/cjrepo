package semverutil

import (
	"fmt"
	"sort"
	"strings"

	sv "github.com/Masterminds/semver/v3"
)

// Check 检查版本是否满足版本需求
// version: 待检查的版本号，如 "1.2.3"
// requirement: 版本需求，如 "[1.0.0, 3.0.0)"、"(, 2.0.0]"、"1.0.0"
//  支持格式：单一版本号、[lower,upper]区间、多区间组合（逗号分隔）
func Check(version, requirement string) (bool, error) {
	v, err := sv.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("parse version %q: %w", version, err)
	}

	ranges := parseRequirements(requirement)
	for _, r := range ranges {
		if r.match(v) {
			return true, nil
		}
	}
	return false, nil
}

type versionRange struct {
	lower     string
	lowerInc  bool
	upper     string
	upperInc  bool
	exact     string
}

func (r *versionRange) match(v *sv.Version) bool {
	if r.exact != "" {
		return v.String() == r.exact
	}

	if r.lower != "" {
		lv, err := sv.NewVersion(r.lower)
		if err != nil {
			return false
		}
		if r.lowerInc {
			if v.LessThan(lv) {
				return false
			}
		} else {
			if v.LessThan(lv) || v.Equal(lv) {
				return false
			}
		}
	}

	if r.upper != "" {
		uv, err := sv.NewVersion(r.upper)
		if err != nil {
			return false
		}
		if r.upperInc {
			if v.GreaterThan(uv) {
				return false
			}
		} else {
			if v.GreaterThan(uv) || v.Equal(uv) {
				return false
			}
		}
	}

	return true
}

// parseRequirements 解析版本需求字符串，支持多区间组合
// 如: "(, 1.0.0), [2.0.0, 3.0.0), 4.0.0"
func parseRequirements(req string) []versionRange {
	req = strings.TrimSpace(req)
	if req == "" {
		return nil
	}

	// 先拆成段（根据括号确定边界）
	var segments []string
	depth := 0
	start := 0
	for i, c := range req {
		switch c {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
			if depth == 0 {
				segments = append(segments, strings.TrimSpace(req[start:i+1]))
				start = i + 1
			}
		case ',':
			if depth == 0 {
				seg := strings.TrimSpace(req[start:i])
				if seg != "" {
					segments = append(segments, seg)
				}
				start = i + 1
			}
		}
	}
	if start < len(req) {
		seg := strings.TrimSpace(req[start:])
		if seg != "" {
			segments = append(segments, seg)
		}
	}

	var result []versionRange
	for _, seg := range segments {
		r := parseSingle(seg)
		if r != nil {
			result = append(result, *r)
		}
	}
	return result
}

func parseSingle(seg string) *versionRange {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return nil
	}

	// 检查是否为区间格式 [a,b] 或 (a,b)
	if (seg[0] == '[' || seg[0] == '(') && (seg[len(seg)-1] == ']' || seg[len(seg)-1] == ')') {
		lowerInc := seg[0] == '['
		upperInc := seg[len(seg)-1] == ']'

		inner := strings.TrimSpace(seg[1 : len(seg)-1])
		parts := strings.SplitN(inner, ",", 2)
		if len(parts) != 2 {
			return nil
		}

		lower := strings.TrimSpace(parts[0])
		upper := strings.TrimSpace(parts[1])

		return &versionRange{
			lower:    lower,
			lowerInc: lowerInc,
			upper:    upper,
			upperInc: upperInc,
		}
	}

	// 处理无符号时的纯版本号
	// 检查逗号隔开的多个版本号
	if strings.Contains(seg, ",") {
		var ors []versionRange
		for _, s := range strings.Split(seg, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				if r := parseSingle(s); r != nil {
					ors = append(ors, *r)
				}
			}
		}
		if len(ors) > 0 {
			// 把 ors 当作多个 exact 匹配处理
			// 实际用户一般不会写 "1.0.0, 2.0.0"（除非确实如此）
			// 这里简化为只取第一个精确匹配
			return &ors[0]
		}
		return nil
	}

	return &versionRange{exact: seg}
}

// BestMatch 从可用版本列表中选择满足 requirement 的最佳版本
// available: 可用版本列表（已排序，从旧到新）
// requirement: 版本需求
// 返回最佳版本号（最高匹配版本）
func BestMatch(available []string, requirement string) (string, error) {
	if len(available) == 0 {
		return "", fmt.Errorf("no available versions")
	}

	ranges := parseRequirements(requirement)
	if len(ranges) == 0 {
		return "", fmt.Errorf("invalid requirement: %s", requirement)
	}

	// 从高到低检查
	for i := len(available) - 1; i >= 0; i-- {
		v, err := sv.NewVersion(available[i])
		if err != nil {
			continue
		}
		for _, r := range ranges {
			if r.match(v) {
				return available[i], nil
			}
		}
	}

	return "", fmt.Errorf("no matching version for requirement: %s", requirement)
}

// Sort 版本号排序（从旧到新，原地修改）
func Sort(versions []string) {
	parsed := make([]*sv.Version, len(versions))
	for i, v := range versions {
		pv, err := sv.NewVersion(v)
		if err != nil {
			versions[i] = v
			continue
		}
		parsed[i] = pv
	}
	sort.Sort(sv.Collection(parsed))
	for i, pv := range parsed {
		versions[i] = pv.Original()
	}
}

// Compare 比较两个版本号
// return: -1 if a < b, 0 if a == b, 1 if a > b
func Compare(a, b string) (int, error) {
	va, err := sv.NewVersion(a)
	if err != nil {
		return 0, err
	}
	vb, err := sv.NewVersion(b)
	if err != nil {
		return 0, err
	}
	if va.LessThan(vb) {
		return -1, nil
	}
	if vb.LessThan(va) {
		return 1, nil
	}
	return 0, nil
}

// ValidateVersion 验证版本号格式
func ValidateVersion(version string) bool {
	_, err := sv.NewVersion(version)
	return err == nil
}


