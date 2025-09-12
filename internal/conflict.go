package internal

import (
	"fmt"
	"time"
)

// ConflictType 冲突类型.
type ConflictType string

const (
	ConflictTypeNewMirror     ConflictType = "new_mirror"     // 新增镜像源
	ConflictTypeDeletedMirror ConflictType = "deleted_mirror" // 删除镜像源
	ConflictTypeModifiedMirror ConflictType = "modified_mirror" // 修改镜像源
	ConflictTypeCurrentChange ConflictType = "current_change" // 当前激活源变更
)

// ConflictItem 冲突项.
type ConflictItem struct {
	Type        ConflictType    `json:"type"`         // 冲突类型
	Name        string          `json:"name"`         // 镜像源名称
	LocalMirror *MirrorConfig   `json:"local_mirror"` // 本地配置
	RemoteMirror *MirrorConfig  `json:"remote_mirror"` // 远程配置
	Description string          `json:"description"`  // 冲突描述
}

// ConflictResolution 冲突解决方案.
type ConflictResolution struct {
	Conflicts []ConflictItem `json:"conflicts"`     // 冲突列表
	Strategy  string         `json:"strategy"`      // 解决策略
	Timestamp time.Time      `json:"timestamp"`     // 检测时间
}

// ConflictResolver 冲突解决器.
type ConflictResolver struct {
	localConfig  *SystemConfig
	remoteData   *SyncData
}

// NewConflictResolver 创建冲突解决器.
func NewConflictResolver(localConfig *SystemConfig, remoteData *SyncData) *ConflictResolver {
	return &ConflictResolver{
		localConfig: localConfig,
		remoteData:  remoteData,
	}
}

// DetectConflicts 检测配置冲突.
func (cr *ConflictResolver) DetectConflicts() *ConflictResolution {
	var conflicts []ConflictItem
	
	// 创建本地镜像源映射
	localMirrors := make(map[string]*MirrorConfig)
	for i := range cr.localConfig.Mirrors {
		mirror := &cr.localConfig.Mirrors[i]
		localMirrors[mirror.Name] = mirror
	}
	
	// 创建远程镜像源映射
	remoteMirrors := make(map[string]*MirrorConfig)
	for i := range cr.remoteData.Mirrors {
		mirror := &cr.remoteData.Mirrors[i]
		remoteMirrors[mirror.Name] = mirror
	}
	
	// 检查远程新增或修改的镜像源
	for name, remoteMirror := range remoteMirrors {
		if localMirror, exists := localMirrors[name]; exists {
			// 镜像源存在，检查是否有修改
			if cr.isMirrorModified(localMirror, remoteMirror) {
				conflicts = append(conflicts, ConflictItem{
					Type:         ConflictTypeModifiedMirror,
					Name:         name,
					LocalMirror:  localMirror,
					RemoteMirror: remoteMirror,
					Description:  fmt.Sprintf("镜像源 '%s' 在本地和云端都有修改", name),
				})
			}
		} else {
			// 远程新增的镜像源
			conflicts = append(conflicts, ConflictItem{
				Type:         ConflictTypeNewMirror,
				Name:         name,
				LocalMirror:  nil,
				RemoteMirror: remoteMirror,
				Description:  fmt.Sprintf("云端新增了镜像源 '%s'", name),
			})
		}
	}
	
	// 检查本地删除的镜像源
	for name, localMirror := range localMirrors {
		if _, exists := remoteMirrors[name]; !exists {
			conflicts = append(conflicts, ConflictItem{
				Type:         ConflictTypeDeletedMirror,
				Name:         name,
				LocalMirror:  localMirror,
				RemoteMirror: nil,
				Description:  fmt.Sprintf("本地删除了镜像源 '%s'，但云端仍存在", name),
			})
		}
	}
	
	// 检查当前激活源的冲突
	if cr.localConfig.CurrentCodex != cr.remoteData.CurrentCodex {
		conflicts = append(conflicts, ConflictItem{
			Type:        ConflictTypeCurrentChange,
			Name:        "current_codex",
			Description: fmt.Sprintf("当前Codex镜像源冲突: 本地='%s', 云端='%s'", cr.localConfig.CurrentCodex, cr.remoteData.CurrentCodex),
		})
	}
	
	if cr.localConfig.CurrentClaude != cr.remoteData.CurrentClaude {
		conflicts = append(conflicts, ConflictItem{
			Type:        ConflictTypeCurrentChange,
			Name:        "current_claude",
			Description: fmt.Sprintf("当前Claude镜像源冲突: 本地='%s', 云端='%s'", cr.localConfig.CurrentClaude, cr.remoteData.CurrentClaude),
		})
	}
	
	return &ConflictResolution{
		Conflicts: conflicts,
		Strategy:  "manual", // 默认需要手动解决
		Timestamp: time.Now(),
	}
}

// isMirrorModified 检查镜像源是否被修改.
func (cr *ConflictResolver) isMirrorModified(local, remote *MirrorConfig) bool {
	// 比较关键字段（忽略API密钥，因为远程的是加密的）
	return local.BaseURL != remote.BaseURL ||
		   local.ToolType != remote.ToolType ||
		   local.ModelName != remote.ModelName
}

// ResolveConflicts 解决冲突.
func (cr *ConflictResolver) ResolveConflicts(resolution *ConflictResolution, strategy string) (*SystemConfig, error) {
	// 创建解决后的配置副本
	resolvedConfig := &SystemConfig{
		CurrentMirror: cr.localConfig.CurrentMirror,
		CurrentCodex:  cr.localConfig.CurrentCodex,
		CurrentClaude: cr.localConfig.CurrentClaude,
		Mirrors:       make([]MirrorConfig, len(cr.localConfig.Mirrors)),
		Sync:          cr.localConfig.Sync,
	}
	copy(resolvedConfig.Mirrors, cr.localConfig.Mirrors)
	
	switch strategy {
	case "local":
		return cr.resolveWithLocalPriority(resolvedConfig, resolution)
	case "remote":
		return cr.resolveWithRemotePriority(resolvedConfig, resolution)
	case "merge":
		return cr.resolveWithMerge(resolvedConfig, resolution)
	default:
		return nil, fmt.Errorf("不支持的冲突解决策略: %s", strategy)
	}
}

// resolveWithLocalPriority 以本地配置为准解决冲突.
func (cr *ConflictResolver) resolveWithLocalPriority(config *SystemConfig, resolution *ConflictResolution) (*SystemConfig, error) {
	// 本地优先：保持本地配置不变，只添加远程新增的镜像源
	for _, conflict := range resolution.Conflicts {
		if conflict.Type == ConflictTypeNewMirror && conflict.RemoteMirror != nil {
			// 添加远程新增的镜像源，但清空API密钥（需要用户重新配置）
			newMirror := *conflict.RemoteMirror
			newMirror.APIKey = "" // 清空加密的API密钥
			config.Mirrors = append(config.Mirrors, newMirror)
		}
	}
	return config, nil
}

// resolveWithRemotePriority 以远程配置为准解决冲突.
func (cr *ConflictResolver) resolveWithRemotePriority(config *SystemConfig, resolution *ConflictResolution) (*SystemConfig, error) {
	// 远程优先：使用远程配置，但保留本地的API密钥
	localKeys := make(map[string]string)
	for _, mirror := range cr.localConfig.Mirrors {
		if mirror.APIKey != "" {
			localKeys[mirror.Name] = mirror.APIKey
		}
	}
	
	// 使用远程镜像源列表
	config.Mirrors = make([]MirrorConfig, len(cr.remoteData.Mirrors))
	copy(config.Mirrors, cr.remoteData.Mirrors)
	
	// 恢复本地API密钥
	for i := range config.Mirrors {
		mirror := &config.Mirrors[i]
		if localKey, exists := localKeys[mirror.Name]; exists {
			mirror.APIKey = localKey // 使用本地未加密的API密钥
		} else {
			mirror.APIKey = "" // 清空远程加密的API密钥，需要用户重新配置
		}
		
		// 设置环境变量key
		switch mirror.ToolType {
		case ToolTypeCodex:
			mirror.EnvKey = CodexSwitchAPIKeyEnv
		case ToolTypeClaude:
			mirror.EnvKey = "ANTHROPIC_AUTH_TOKEN"
		}
	}
	
	// 使用远程的当前激活源
	config.CurrentCodex = cr.remoteData.CurrentCodex
	config.CurrentClaude = cr.remoteData.CurrentClaude
	
	return config, nil
}

// resolveWithMerge 合并本地和远程配置.
func (cr *ConflictResolver) resolveWithMerge(config *SystemConfig, resolution *ConflictResolution) (*SystemConfig, error) {
	// 智能合并：保留本地API密钥，合并镜像源列表
	localKeys := make(map[string]string)
	for _, mirror := range cr.localConfig.Mirrors {
		if mirror.APIKey != "" {
			localKeys[mirror.Name] = mirror.APIKey
		}
	}
	
	// 创建合并后的镜像源映射
	mergedMirrors := make(map[string]MirrorConfig)
	
	// 先添加本地镜像源
	for _, mirror := range cr.localConfig.Mirrors {
		mergedMirrors[mirror.Name] = mirror
	}
	
	// 合并远程镜像源
	for _, remoteMirror := range cr.remoteData.Mirrors {
		if localMirror, exists := mergedMirrors[remoteMirror.Name]; exists {
			// 镜像源已存在，保留本地API密钥，使用远程的其他配置
			merged := remoteMirror
			merged.APIKey = localMirror.APIKey // 保留本地API密钥
			
			// 设置环境变量key
			switch merged.ToolType {
			case ToolTypeCodex:
				merged.EnvKey = CodexSwitchAPIKeyEnv
			case ToolTypeClaude:
				merged.EnvKey = "ANTHROPIC_AUTH_TOKEN"
			}
			
			mergedMirrors[remoteMirror.Name] = merged
		} else {
			// 新的镜像源，清空API密钥
			newMirror := remoteMirror
			newMirror.APIKey = ""
			
			// 设置环境变量key
			switch newMirror.ToolType {
			case ToolTypeCodex:
				newMirror.EnvKey = CodexSwitchAPIKeyEnv
			case ToolTypeClaude:
				newMirror.EnvKey = "ANTHROPIC_AUTH_TOKEN"
			}
			
			mergedMirrors[remoteMirror.Name] = newMirror
		}
	}
	
	// 转换为数组
	config.Mirrors = make([]MirrorConfig, 0, len(mergedMirrors))
	for _, mirror := range mergedMirrors {
		config.Mirrors = append(config.Mirrors, mirror)
	}
	
	// 智能选择当前激活源
	if cr.remoteData.CurrentCodex != "" {
		// 检查远程激活源是否存在于合并后的配置中
		if _, exists := mergedMirrors[cr.remoteData.CurrentCodex]; exists {
			config.CurrentCodex = cr.remoteData.CurrentCodex
		}
	}
	
	if cr.remoteData.CurrentClaude != "" {
		if _, exists := mergedMirrors[cr.remoteData.CurrentClaude]; exists {
			config.CurrentClaude = cr.remoteData.CurrentClaude
		}
	}
	
	return config, nil
}

// FormatConflicts 格式化冲突信息用于显示.
func (cr *ConflictResolver) FormatConflicts(resolution *ConflictResolution) string {
	if len(resolution.Conflicts) == 0 {
		return "没有检测到配置冲突"
	}
	
	output := fmt.Sprintf("检测到 %d 个配置冲突:\n", len(resolution.Conflicts))
	output += "==================================================\n"
	
	for i, conflict := range resolution.Conflicts {
		output += fmt.Sprintf("%d. %s\n", i+1, conflict.Description)
		
		switch conflict.Type {
		case ConflictTypeModifiedMirror:
			output += fmt.Sprintf("   本地: %s (%s)\n", conflict.LocalMirror.BaseURL, conflict.LocalMirror.ToolType)
			output += fmt.Sprintf("   云端: %s (%s)\n", conflict.RemoteMirror.BaseURL, conflict.RemoteMirror.ToolType)
		case ConflictTypeNewMirror:
			output += fmt.Sprintf("   云端配置: %s (%s)\n", conflict.RemoteMirror.BaseURL, conflict.RemoteMirror.ToolType)
		case ConflictTypeDeletedMirror:
			output += fmt.Sprintf("   本地配置: %s (%s)\n", conflict.LocalMirror.BaseURL, conflict.LocalMirror.ToolType)
		}
		output += "\n"
	}
	
	return output
}