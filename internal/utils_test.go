package internal

import (
	"testing"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "空字符串",
			apiKey:   "",
			expected: "",
		},
		{
			name:     "长度为1",
			apiKey:   "a",
			expected: "****",
		},
		{
			name:     "长度为8",
			apiKey:   "12345678",
			expected: "****",
		},
		{
			name:     "长度为9",
			apiKey:   "123456789",
			expected: "1234****6789",
		},
		{
			name:     "长度为16",
			apiKey:   "abcdefghijklmnop",
			expected: "abcd****mnop",
		},
		{
			name:     "标准API key长度 - 测试用假数据",
			apiKey:   "sk-test123456789012345678901234567890",
			expected: "sk-t****7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("MaskAPIKey(%q) = %q, expected %q", tt.apiKey, result, tt.expected)
			}
		})
	}
}
