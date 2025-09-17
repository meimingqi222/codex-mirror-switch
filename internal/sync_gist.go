package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GistProvider GitHub Gist 同步提供商.
type GistProvider struct {
	token  string
	gistID string
	client *http.Client
}

// NewGistProvider 创建新的 GitHub Gist 提供商.
func NewGistProvider(token, gistID string) (*GistProvider, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token 不能为空")
	}

	provider := &GistProvider{
		token:  token,
		gistID: gistID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// 如果没有提供 Gist ID，尝试自动发现
	if gistID == "" {
		if discoveredID, err := provider.discoverExistingGist(); err == nil && discoveredID != "" {
			provider.gistID = discoveredID
			fmt.Printf("🔍 自动发现现有配置 Gist: %s\n", discoveredID)
		}
	}

	return provider, nil
}

// Upload 上传数据到 GitHub Gist.
func (g *GistProvider) Upload(data []byte, filename string) error {
	// 将数据编码为 base64
	encodedData := base64.StdEncoding.EncodeToString(data)

	// 构建 Gist 数据
	gistData := map[string]interface{}{
		"description": "Codex Mirror Switch Configuration - codex-mirror-sync", // 添加特殊标识
		"public":      false,
		"files": map[string]interface{}{
			filename: map[string]string{
				"content": encodedData,
			},
		},
	}

	// 序列化请求数据
	requestData, err := json.Marshal(gistData)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	var url string
	var method string

	if g.gistID == "" {
		// 创建新的 Gist
		url = "https://api.github.com/gists"
		method = "POST"
	} else {
		// 更新现有的 Gist
		url = fmt.Sprintf("https://api.github.com/gists/%s", g.gistID)
		method = "PATCH"
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest(method, url, bytes.NewBuffer(requestData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// 发送请求
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitHub API 错误 (%d): %s", resp.StatusCode, string(respBody))
	}

	// 解析响应获取 Gist ID（如果是新创建的）
	if g.gistID == "" {
		var gistResp map[string]interface{}
		if err := json.Unmarshal(respBody, &gistResp); err == nil {
			if id, ok := gistResp["id"].(string); ok {
				g.gistID = id
			}
		}
	}

	return nil
}

// Download 从 GitHub Gist 下载数据.
func (g *GistProvider) Download(filename string) ([]byte, error) {
	if g.gistID == "" {
		return nil, fmt.Errorf("Gist ID 未设置")
	}

	respBody, err := g.fetchGistData()
	if err != nil {
		return nil, err
	}

	fileContent, err := g.extractFileContent(respBody, filename)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(fileContent)
}

func (g *GistProvider) fetchGistData() ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/gists/%s", g.gistID)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 错误 (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (g *GistProvider) extractFileContent(respBody []byte, filename string) (string, error) {
	var gistResp map[string]interface{}
	if err := json.Unmarshal(respBody, &gistResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	files, ok := gistResp["files"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("响应中没有找到文件信息")
	}

	if filename != "" {
		return g.getSpecificFile(files, filename)
	}
	return g.getFirstFile(files)
}

func (g *GistProvider) getSpecificFile(files map[string]interface{}, filename string) (string, error) {
	file, exists := files[filename]
	if !exists {
		return "", fmt.Errorf("未找到文件: %s", filename)
	}

	return extractContentFromFile(file)
}

func (g *GistProvider) getFirstFile(files map[string]interface{}) (string, error) {
	for _, file := range files {
		content, err := extractContentFromFile(file)
		if err == nil && content != "" {
			return content, nil
		}
	}
	return "", fmt.Errorf("未找到文件内容")
}

func extractContentFromFile(file interface{}) (string, error) {
	fileData, ok := file.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("文件数据格式错误")
	}

	content, ok := fileData["content"].(string)
	if !ok {
		return "", fmt.Errorf("未找到文件内容")
	}

	return content, nil
}

// List 列出 Gist 中的所有文件.
func (g *GistProvider) List() ([]string, error) {
	if g.gistID == "" {
		return []string{}, nil // 如果没有 Gist ID，返回空列表
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://api.github.com/gists/%s", g.gistID)

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// 发送请求
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 错误 (%d): %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var gistResp map[string]interface{}
	if err := json.Unmarshal(respBody, &gistResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 获取文件列表
	files, ok := gistResp["files"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("响应中没有找到文件信息")
	}

	var fileList []string
	for filename := range files {
		// 返回新的统一配置文件或旧的设备特定配置文件
		if filename == ConfigFileName ||
			(strings.HasPrefix(filename, "codex-mirror-config-") && strings.HasSuffix(filename, ".json")) {
			fileList = append(fileList, filename)
		}
	}

	return fileList, nil
}

// Delete 删除 Gist 中的文件.
func (g *GistProvider) Delete(filename string) error {
	if g.gistID == "" {
		return fmt.Errorf("Gist ID 未设置")
	}

	// 构建删除文件的数据（设置文件内容为 null）
	gistData := map[string]interface{}{
		"files": map[string]interface{}{
			filename: nil,
		},
	}

	// 序列化请求数据
	requestData, err := json.Marshal(gistData)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://api.github.com/gists/%s", g.gistID)

	// 创建 HTTP 请求
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(requestData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// 发送请求
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API 错误 (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetInfo 获取提供商信息.
func (g *GistProvider) GetInfo() ProviderInfo {
	return ProviderInfo{
		Name:        "GitHub Gist",
		Type:        "gist",
		Endpoint:    "https://api.github.com",
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		Description: "使用 GitHub Gist 存储配置文件",
	}
}

// GetGistID 获取 Gist ID.
func (g *GistProvider) GetGistID() string {
	return g.gistID
}

// SetGistID 设置 Gist ID.
func (g *GistProvider) SetGistID(gistID string) {
	g.gistID = gistID
}

// GistCandidate 表示一个候选的配置 Gist.
type GistCandidate struct {
	ID          string
	Description string
	UpdatedAt   string
	CreatedAt   string
}

// discoverExistingGist 自动发现现有的配置 Gist.
func (g *GistProvider) discoverExistingGist() (string, error) {
	// 构建请求 URL - 获取用户的所有 Gist
	url := "https://api.github.com/gists"

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// 发送请求
	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API 错误 (%d): %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var gists []map[string]interface{}
	if err := json.Unmarshal(respBody, &gists); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 收集所有匹配的 Gist 候选项
	candidates := g.extractCandidatesFromGists(gists)

	// 如果没有找到候选项
	if len(candidates) == 0 {
		return "", nil
	}

	// 如果只有一个候选项，直接返回
	if len(candidates) == 1 {
		return candidates[0].ID, nil
	}

	// 多个候选项：选择最新更新的
	fmt.Printf("🔍 发现 %d 个配置 Gist，选择最新的...\n", len(candidates))

	latestCandidate := candidates[0]
	latestTime, err := time.Parse(time.RFC3339, candidates[0].UpdatedAt)
	if err != nil {
		// 如果解析时间失败，回退到第一个
		return candidates[0].ID, nil
	}

	for i := 1; i < len(candidates); i++ {
		candidateTime, err := time.Parse(time.RFC3339, candidates[i].UpdatedAt)
		if err != nil {
			continue
		}

		if candidateTime.After(latestTime) {
			latestCandidate = candidates[i]
			latestTime = candidateTime
		}
	}

	fmt.Printf("   选中最新的 Gist (更新于: %s): %s\n",
		latestTime.Format("2006-01-02 15:04:05"), latestCandidate.ID)

	return latestCandidate.ID, nil
}

// validateGistContent 验证 Gist 是否包含有效的配置文件.
func (g *GistProvider) validateGistContent(gistID string) bool {
	// 构建请求 URL
	url := fmt.Sprintf("https://api.github.com/gists/%s", gistID)

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return false
	}

	// 设置请求头
	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// 发送请求
	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return false
	}

	// 解析响应
	var gistResp map[string]interface{}
	if err := json.Unmarshal(respBody, &gistResp); err != nil {
		return false
	}

	// 检查文件
	files, ok := gistResp["files"].(map[string]interface{})
	if !ok {
		return false
	}

	// 查找配置文件 - 检查新的统一命名格式
	if _, exists := files[ConfigFileName]; exists {
		return true
	}

	// 同时支持旧的设备特定命名格式（向后兼容）
	for filename := range files {
		if strings.HasPrefix(filename, "codex-mirror-config-") && strings.HasSuffix(filename, ".json") {
			return true
		}
	}

	return false
}

// extractCandidatesFromGists 从 Gist 列表中提取有效的候选项.
func (g *GistProvider) extractCandidatesFromGists(gists []map[string]interface{}) []GistCandidate {
	var candidates []GistCandidate
	for _, gist := range gists {
		candidate := g.extractCandidateFromGist(gist)
		if candidate != nil {
			candidates = append(candidates, *candidate)
		}
	}
	return candidates
}

// extractCandidateFromGist 从单个 Gist 中提取候选项信息.
func (g *GistProvider) extractCandidateFromGist(gist map[string]interface{}) *GistCandidate {
	description, ok := gist["description"].(string)
	if !ok {
		return nil
	}

	// 检查描述是否包含我们的标识
	if !strings.Contains(description, "codex-mirror-sync") {
		return nil
	}

	id, ok := gist["id"].(string)
	if !ok {
		return nil
	}

	// 进一步验证 Gist 内容
	if !g.validateGistContent(id) {
		return nil
	}

	candidate := GistCandidate{
		ID:          id,
		Description: description,
	}

	// 获取时间信息
	if updatedAt, ok := gist["updated_at"].(string); ok {
		candidate.UpdatedAt = updatedAt
	}
	if createdAt, ok := gist["created_at"].(string); ok {
		candidate.CreatedAt = createdAt
	}

	return &candidate
}
