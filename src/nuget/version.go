package nuget

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var validVersion = regexp.MustCompile(
	`^[0-9]+(\.[0-9]+){0,3}(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`,
)

type parsedVersion struct {
	parts      [4]uint64
	prerelease string
}

type versionRange struct {
	raw             string
	minimum         *parsedVersion
	maximum         *parsedVersion
	includeMinimum  bool
	includeMaximum  bool
	allowPrerelease bool
}

func parseVersion(raw string) (parsedVersion, error) {
	if !validVersion.MatchString(raw) {
		return parsedVersion{}, fmt.Errorf("некорректная версия %q", raw)
	}
	withoutBuild, _, _ := strings.Cut(raw, "+")
	numeric, prerelease, _ := strings.Cut(withoutBuild, "-")
	var result parsedVersion
	result.prerelease = strings.ToLower(prerelease)
	for index, part := range strings.Split(numeric, ".") {
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return parsedVersion{}, fmt.Errorf("некорректная версия %q: %w", raw, err)
		}
		result.parts[index] = value
	}
	return result, nil
}

func parseRange(raw string) (versionRange, error) {
	raw = strings.TrimSpace(raw)
	result := versionRange{raw: raw}
	if raw == "" {
		return result, fmt.Errorf("пустой диапазон версий")
	}
	if raw[0] != '[' && raw[0] != '(' {
		minimum, err := parseVersion(raw)
		result.minimum = &minimum
		result.includeMinimum = true
		result.allowPrerelease = minimum.prerelease != ""
		return result, err
	}
	if len(raw) < 3 || (raw[len(raw)-1] != ']' && raw[len(raw)-1] != ')') {
		return result, fmt.Errorf("некорректный диапазон %q", raw)
	}
	result.includeMinimum = raw[0] == '['
	result.includeMaximum = raw[len(raw)-1] == ']'
	content := raw[1 : len(raw)-1]
	if !strings.Contains(content, ",") {
		if !result.includeMinimum || !result.includeMaximum {
			return result, fmt.Errorf("некорректный диапазон %q", raw)
		}
		version, err := parseVersion(content)
		result.minimum, result.maximum = &version, &version
		result.allowPrerelease = version.prerelease != ""
		return result, err
	}
	return parseRangeBounds(result, content)
}

func parseRangeBounds(result versionRange, content string) (versionRange, error) {
	parts := strings.Split(content, ",")
	if len(parts) != 2 {
		return result, fmt.Errorf("некорректный диапазон %q", result.raw)
	}
	parts[0], parts[1] = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if parts[0] == "" && parts[1] == "" {
		return result, fmt.Errorf("некорректный диапазон %q", result.raw)
	}
	minimum, err := optionalVersion(parts[0])
	if err != nil {
		return result, err
	}
	maximum, err := optionalVersion(parts[1])
	if err != nil {
		return result, err
	}
	result.minimum, result.maximum = minimum, maximum
	result.allowPrerelease = hasPrerelease(minimum) || hasPrerelease(maximum)
	return result, validateRangeBounds(result)
}

func optionalVersion(raw string) (*parsedVersion, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := parseVersion(raw)
	return &value, err
}

func hasPrerelease(version *parsedVersion) bool {
	return version != nil && version.prerelease != ""
}

func validateRangeBounds(result versionRange) error {
	if result.minimum == nil || result.maximum == nil {
		return nil
	}
	order := compareVersion(*result.minimum, *result.maximum)
	if order > 0 || (order == 0 && (!result.includeMinimum || !result.includeMaximum)) {
		return fmt.Errorf("пустой диапазон %q", result.raw)
	}
	return nil
}

func (r versionRange) lowest(versions []string) (string, error) {
	selected := ""
	var selectedVersion parsedVersion
	for _, raw := range versions {
		candidate, err := parseVersion(raw)
		if err != nil {
			return "", err
		}
		if !r.matches(candidate) {
			continue
		}
		if selected == "" || compareVersion(candidate, selectedVersion) < 0 {
			selected, selectedVersion = raw, candidate
		}
	}
	if selected == "" {
		return "", fmt.Errorf("нет опубликованной версии для диапазона %q", r.raw)
	}
	return selected, nil
}

func (r versionRange) matches(candidate parsedVersion) bool {
	if hasPrerelease(&candidate) && !r.allowPrerelease {
		return false
	}
	if !r.meetsMinimum(candidate) {
		return false
	}
	return r.meetsMaximum(candidate)
}

func (r versionRange) meetsMinimum(candidate parsedVersion) bool {
	if r.minimum == nil {
		return true
	}
	order := compareVersion(candidate, *r.minimum)
	return order > 0 || (order == 0 && r.includeMinimum)
}

func (r versionRange) meetsMaximum(candidate parsedVersion) bool {
	if r.maximum == nil {
		return true
	}
	order := compareVersion(candidate, *r.maximum)
	return order < 0 || (order == 0 && r.includeMaximum)
}

func compareVersion(left parsedVersion, right parsedVersion) int {
	for index := range left.parts {
		if left.parts[index] < right.parts[index] {
			return -1
		}
		if left.parts[index] > right.parts[index] {
			return 1
		}
	}
	if left.prerelease == "" || right.prerelease == "" {
		if left.prerelease == right.prerelease {
			return 0
		}
		if left.prerelease == "" {
			return 1
		}
		return -1
	}
	return comparePrerelease(left.prerelease, right.prerelease)
}

func comparePrerelease(left string, right string) int {
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	for index := 0; index < len(leftParts) && index < len(rightParts); index++ {
		order := compareIdentifier(leftParts[index], rightParts[index])
		if order != 0 {
			return order
		}
	}
	if len(leftParts) < len(rightParts) {
		return -1
	}
	if len(leftParts) > len(rightParts) {
		return 1
	}
	return 0
}

func compareIdentifier(left string, right string) int {
	leftNumber, rightNumber := digitsOnly(left), digitsOnly(right)
	if leftNumber && rightNumber {
		left, right = strings.TrimLeft(left, "0"), strings.TrimLeft(right, "0")
		if len(left) != len(right) {
			if len(left) < len(right) {
				return -1
			}
			return 1
		}
	} else if leftNumber != rightNumber {
		if leftNumber {
			return -1
		}
		return 1
	}
	return strings.Compare(left, right)
}

func digitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
