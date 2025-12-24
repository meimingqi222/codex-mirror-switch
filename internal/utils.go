package internal

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
