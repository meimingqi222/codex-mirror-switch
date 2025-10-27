package internal

import (
	"fmt"
	"time"
)

// ConflictType 冲突类型.
type ConflictType string

const (
	ConflictTypeNewMirror      ConflictType = "new_mirror"      // 新增镜像源
	ConflictTypeDeletedMirror  ConflictType = "deleted_mirror"  // 删除镜像源
	ConflictTypeModifiedMirror ConflictType = "modified_mirror" // 修改镜像源
	ConflictTypeCurrentChange  ConflictType = "current_change"  // 当前激活源变更

	// Conflict resolution strategies.
	StrategyLocal  string = "local"  // 本地优先
	StrategyRemote string = "remote" // 远程优先
	StrategyMerge  string = "merge"  // 智能合并
	StrategyAbort  string = "abort"  // 取消操作

	// Configuration file names.
	ConfigFileName string = "codex-mirror-config.json"
)

// ConflictItem 冲突项.
type ConflictItem struct {
	Type         ConflictType  `json:"type"`          // 冲突类型
	Name         string        `json:"name"`          // 镜像源名称
	LocalMirror  *MirrorConfig `json:"local_mirror"`  // 本地配置
	RemoteMirror *MirrorConfig `json:"remote_mirror"` // 远程配置
	Description  string        `json:"description"`   // 冲突描述
}

// ConflictResolution 冲突解决方案.
type ConflictResolution struct {
	Conflicts []ConflictItem `json:"conflicts"` // 冲突列表
	Strategy  string         `json:"strategy"`  // 解决策略
	Timestamp time.Time      `json:"timestamp"` // 检测时间
}

// ConflictResolver 冲突解决器.
type ConflictResolver struct {
	localConfig *SystemConfig
	remoteData  *SyncData
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
	localMirrors := cr.createMirrorMap(cr.localConfig.Mirrors)
	remoteMirrors := cr.createMirrorMap(cr.remoteData.Mirrors)
	remoteDeletedMirrors := cr.createMirrorMap(cr.remoteData.DeletedMirrors)

	var conflicts []ConflictItem

	// 检查远程新增或修改的镜像源
	conflicts = append(conflicts, cr.checkRemoteChanges(localMirrors, remoteMirrors, remoteDeletedMirrors)...)

	// 检查本地删除的镜像源
	conflicts = append(conflicts, cr.checkLocalDeleted(localMirrors, remoteMirrors, remoteDeletedMirrors)...)

	// 检查云端删除但本地仍活跃的镜像源
	conflicts = append(conflicts, cr.checkRemoteDeletedActive(localMirrors, remoteDeletedMirrors)...)

	// 检查当前激活源的冲突
	conflicts = append(conflicts, cr.checkCurrentConflicts()...)

	return &ConflictResolution{
		Conflicts: conflicts,
		Strategy:  "manual", // 默认需要手动解决
		Timestamp: time.Now(),
	}
}

// createMirrorMap 创建镜像源映射.
func (cr *ConflictResolver) createMirrorMap(mirrors []MirrorConfig) map[string]*MirrorConfig {
	mirrorMap := make(map[string]*MirrorConfig)
	for i := range mirrors {
		mirror := &mirrors[i]
		mirrorMap[mirror.Name] = mirror
	}
	return mirrorMap
}

// checkRemoteChanges 检查远程新增或修改的镜像源.
func (cr *ConflictResolver) checkRemoteChanges(localMirrors, remoteMirrors, remoteDeletedMirrors map[string]*MirrorConfig) []ConflictItem {
	var conflicts []ConflictItem

	for name, remoteMirror := range remoteMirrors {
		if localMirror, exists := localMirrors[name]; exists {
			conflicts = append(conflicts, cr.checkLocalRemoteConflict(name, localMirror, remoteMirror)...)
		} else {
			conflicts = append(conflicts, cr.checkRemoteOnlyMirror(name, remoteMirror, remoteDeletedMirrors)...)
		}
	}

	return conflicts
}

// checkLocalRemoteConflict 检查本地和远程都存在的镜像源冲突.
func (cr *ConflictResolver) checkLocalRemoteConflict(name string, localMirror, remoteMirror *MirrorConfig) []ConflictItem {
	var conflicts []ConflictItem

	if localMirror.Deleted && !localMirror.DeletedAt.IsZero() {
		conflicts = append(conflicts, ConflictItem{
			Type:         ConflictTypeDeletedMirror,
			Name:         name,
			LocalMirror:  localMirror,
			RemoteMirror: remoteMirror,
			Description: fmt.Sprintf("本地删除了镜像源 '%s' (删除时间: %s)，但云端仍存在",
				name, localMirror.DeletedAt.Format("2006-01-02 15:04:05")),
		})
	} else if cr.isMirrorModified(localMirror, remoteMirror) {
		conflicts = append(conflicts, ConflictItem{
			Type:         ConflictTypeModifiedMirror,
			Name:         name,
			LocalMirror:  localMirror,
			RemoteMirror: remoteMirror,
			Description:  fmt.Sprintf("镜像源 '%s' 在本地和云端都有修改", name),
		})
	}

	return conflicts
}

// checkRemoteOnlyMirror 检查仅在远程存在的镜像源.
func (cr *ConflictResolver) checkRemoteOnlyMirror(name string, remoteMirror *MirrorConfig, remoteDeletedMirrors map[string]*MirrorConfig) []ConflictItem {
	var conflicts []ConflictItem

	if deletedMirror, wasDeleted := remoteDeletedMirrors[name]; wasDeleted {
		if cr.isRecentlyDeleted(deletedMirror) {
			conflicts = append(conflicts, ConflictItem{
				Type:         ConflictTypeNewMirror,
				Name:         name,
				LocalMirror:  nil,
				RemoteMirror: remoteMirror,
				Description: fmt.Sprintf("镜像源 '%s' 在云端被删除后重新添加 (删除时间: %s)",
					name, deletedMirror.DeletedAt.Format("2006-01-02 15:04:05")),
			})
		} else {
			conflicts = append(conflicts, ConflictItem{
				Type:         ConflictTypeNewMirror,
				Name:         name,
				LocalMirror:  nil,
				RemoteMirror: remoteMirror,
				Description:  fmt.Sprintf("云端新增了镜像源 '%s'", name),
			})
		}
	} else {
		conflicts = append(conflicts, ConflictItem{
			Type:         ConflictTypeNewMirror,
			Name:         name,
			LocalMirror:  nil,
			RemoteMirror: remoteMirror,
			Description:  fmt.Sprintf("云端新增了镜像源 '%s'", name),
		})
	}

	return conflicts
}

// checkLocalDeleted 检查本地删除的镜像源.
func (cr *ConflictResolver) checkLocalDeleted(localMirrors, remoteMirrors, remoteDeletedMirrors map[string]*MirrorConfig) []ConflictItem {
	var conflicts []ConflictItem

	for name, localMirror := range localMirrors {
		if _, exists := remoteMirrors[name]; !exists {
			if localMirror.Deleted && !localMirror.DeletedAt.IsZero() {
				conflicts = append(conflicts, ConflictItem{
					Type:         ConflictTypeDeletedMirror,
					Name:         name,
					LocalMirror:  localMirror,
					RemoteMirror: nil,
					Description: fmt.Sprintf("本地删除了镜像源 '%s' (删除时间: %s)，建议同步删除云端配置",
						name, localMirror.DeletedAt.Format("2006-01-02 15:04:05")),
				})
			} else if remoteDeleted, wasRemoteDeleted := remoteDeletedMirrors[name]; wasRemoteDeleted {
				conflicts = append(conflicts, ConflictItem{
					Type:         ConflictTypeDeletedMirror,
					Name:         name,
					LocalMirror:  localMirror,
					RemoteMirror: nil,
					Description: fmt.Sprintf("镜像源 '%s' 在云端被删除 (删除时间: %s)，本地配置将保持",
						name, remoteDeleted.DeletedAt.Format("2006-01-02 15:04:05")),
				})
			} else {
				conflicts = append(conflicts, ConflictItem{
					Type:         ConflictTypeDeletedMirror,
					Name:         name,
					LocalMirror:  localMirror,
					RemoteMirror: nil,
					Description:  fmt.Sprintf("本地删除了镜像源 '%s'，但云端仍存在", name),
				})
			}
		}
	}

	return conflicts
}

// checkRemoteDeletedActive 检查云端删除但本地仍活跃的镜像源.
func (cr *ConflictResolver) checkRemoteDeletedActive(localMirrors, remoteDeletedMirrors map[string]*MirrorConfig) []ConflictItem {
	var conflicts []ConflictItem

	for name, remoteDeleted := range remoteDeletedMirrors {
		if localMirror, exists := localMirrors[name]; exists && !localMirror.Deleted {
			if cr.isRecentlyDeleted(remoteDeleted) {
				conflicts = append(conflicts, ConflictItem{
					Type:         ConflictTypeDeletedMirror,
					Name:         name,
					LocalMirror:  localMirror,
					RemoteMirror: nil,
					Description: fmt.Sprintf("云端删除了镜像源 '%s' (删除时间: %s)，建议同步删除本地配置",
						name, remoteDeleted.DeletedAt.Format("2006-01-02 15:04:05")),
				})
			}
		}
	}

	return conflicts
}

// checkCurrentConflicts 检查当前激活源的冲突.
func (cr *ConflictResolver) checkCurrentConflicts() []ConflictItem {
	var conflicts []ConflictItem

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

	return conflicts
}

// isMirrorModified 检查镜像源是否被修改.
func (cr *ConflictResolver) isMirrorModified(local, remote *MirrorConfig) bool {
	// 比较关键字段（忽略API密钥，因为远程的是加密的）
	return local.BaseURL != remote.BaseURL ||
		local.ToolType != remote.ToolType ||
		local.ModelName != remote.ModelName
}

// isIntentionalDeletion 检查是否是明确的本地删除操作.
func (cr *ConflictResolver) isIntentionalDeletion(localMirror *MirrorConfig, remoteDeletedMirrors map[string]*MirrorConfig) bool {
	// 检查本地镜像源是否有删除标记
	if localMirror.Deleted && !localMirror.DeletedAt.IsZero() {
		return true
	}

	// 检查云端是否也有删除记录（可能之前已在云端删除）
	if remoteDeleted, exists := remoteDeletedMirrors[localMirror.Name]; exists {
		if remoteDeleted.Deleted && !remoteDeleted.DeletedAt.IsZero() {
			return true
		}
	}

	// 检查创建和删除时间间隔，排除可能是临时配置的情况
	if !localMirror.CreatedAt.IsZero() && !localMirror.LastModified.IsZero() {
		// 如果镜像源存在时间很短（比如1小时内），可能是误操作
		existenceDuration := localMirror.LastModified.Sub(localMirror.CreatedAt)
		if existenceDuration < time.Hour {
			return false
		}
	}

	return false
}

// isRecentlyDeleted 检查是否是最近删除的操作.
func (cr *ConflictResolver) isRecentlyDeleted(mirror *MirrorConfig) bool {
	if !mirror.Deleted || mirror.DeletedAt.IsZero() {
		return false
	}

	// 删除时间在7天内认为是最近删除
	threshold := time.Now().Add(-7 * 24 * time.Hour)
	return mirror.DeletedAt.After(threshold)
}

// findLocalDeletedMirror 查找本地已删除的镜像源记录.
func (cr *ConflictResolver) findLocalDeletedMirror(name string) *MirrorConfig {
	for i := range cr.localConfig.Mirrors {
		mirror := &cr.localConfig.Mirrors[i]
		if mirror.Name == name && mirror.Deleted && !mirror.DeletedAt.IsZero() {
			return mirror
		}
	}
	return nil
}

// selectDefaultMirror 选择默认镜像源（当当前激活源被删除时）.
func (cr *ConflictResolver) selectDefaultMirror(availableMirrors map[string]MirrorConfig, toolType ToolType) string {
	// 优先选择官方镜像源
	for name := range availableMirrors {
		mirror := availableMirrors[name]
		if mirror.ToolType == toolType && name == DefaultMirrorName {
			return name
		}
	}

	// 其次选择同类型的第一个可用镜像源
	for name := range availableMirrors {
		mirror := availableMirrors[name]
		if mirror.ToolType == toolType {
			return name
		}
	}

	// 如果没有找到合适的选择，返回空字符串
	return ""
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
	case StrategyLocal:
		return cr.resolveWithLocalPriority(resolvedConfig, resolution)
	case StrategyRemote:
		return cr.resolveWithRemotePriority(resolvedConfig, resolution)
	case StrategyMerge:
		return cr.resolveWithMerge(resolvedConfig, resolution)
	default:
		return nil, fmt.Errorf("不支持的冲突解决策略: %s", strategy)
	}
}

// resolveWithLocalPriority 以本地配置为准解决冲突.
func (cr *ConflictResolver) resolveWithLocalPriority(config *SystemConfig, resolution *ConflictResolution) (*SystemConfig, error) {
	// 本地优先：保持本地配置不变，只添加远程新增的镜像源
	for i := range resolution.Conflicts {
		conflict := &resolution.Conflicts[i]
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
func (cr *ConflictResolver) resolveWithRemotePriority(config *SystemConfig, _ *ConflictResolution) (*SystemConfig, error) {
	// 远程优先：使用远程配置，但保留本地的API密钥
	localKeys := make(map[string]string)
	for i := range cr.localConfig.Mirrors {
		mirror := &cr.localConfig.Mirrors[i]
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
			mirror.EnvKey = AnthropicAuthTokenEnv
		}
	}

	// 使用远程的当前激活源
	config.CurrentCodex = cr.remoteData.CurrentCodex
	config.CurrentClaude = cr.remoteData.CurrentClaude

	return config, nil
}

// resolveWithMerge 合并本地和远程配置.
func (cr *ConflictResolver) resolveWithMerge(config *SystemConfig, resolution *ConflictResolution) (*SystemConfig, error) {
	mergedMirrors := cr.initializeLocalMirrors()
	remoteDeletedMirrors := cr.createMirrorMap(cr.remoteData.DeletedMirrors)

	// 合并远程镜像源
	cr.mergeRemoteMirrors(mergedMirrors, remoteDeletedMirrors, resolution)

	// 处理云端已删除的镜像源
	cr.handleRemoteDeletedMirrors(mergedMirrors, remoteDeletedMirrors, resolution)

	// 转换为数组并设置当前激活源
	cr.finalizeMergeConfig(config, mergedMirrors)

	return config, nil
}

// initializeLocalMirrors 初始化本地镜像源映射.
func (cr *ConflictResolver) initializeLocalMirrors() map[string]MirrorConfig {
	mergedMirrors := make(map[string]MirrorConfig)
	for i := range cr.localConfig.Mirrors {
		mirror := &cr.localConfig.Mirrors[i]
		// 跳过明确标记为删除的本地镜像源
		if !mirror.Deleted || mirror.DeletedAt.IsZero() {
			mergedMirrors[mirror.Name] = *mirror
		}
	}
	return mergedMirrors
}

// mergeRemoteMirrors 合并远程镜像源.
func (cr *ConflictResolver) mergeRemoteMirrors(mergedMirrors map[string]MirrorConfig, remoteDeletedMirrors map[string]*MirrorConfig, resolution *ConflictResolution) {
	for i := range cr.remoteData.Mirrors {
		remoteMirror := &cr.remoteData.Mirrors[i]
		hasConflict := cr.hasDeleteConflict(remoteMirror.Name, resolution)

		if localMirror, exists := mergedMirrors[remoteMirror.Name]; exists {
			cr.mergeExistingMirror(mergedMirrors, remoteMirror, localMirror)
		} else {
			cr.mergeNewMirror(mergedMirrors, remoteMirror, remoteDeletedMirrors, hasConflict)
		}
	}
}

// hasDeleteConflict 检查是否有删除冲突.
func (cr *ConflictResolver) hasDeleteConflict(mirrorName string, resolution *ConflictResolution) bool {
	for _, conflict := range resolution.Conflicts {
		if conflict.Name == mirrorName && conflict.Type == ConflictTypeDeletedMirror {
			return true
		}
	}
	return false
}

// mergeExistingMirror 合并已存在的镜像源.
func (cr *ConflictResolver) mergeExistingMirror(mergedMirrors map[string]MirrorConfig, remoteMirror *MirrorConfig, localMirror MirrorConfig) {
	merged := localMirror // 使用本地配置作为基础，保留本地所有修改
	// 如果本地没有API密钥但远程有（且是加密的），保持本地为空（需要用户重新配置）
	cr.setEnvKey(&merged)
	mergedMirrors[remoteMirror.Name] = merged
}

// mergeNewMirror 合并新的镜像源.
func (cr *ConflictResolver) mergeNewMirror(mergedMirrors map[string]MirrorConfig, remoteMirror *MirrorConfig, remoteDeletedMirrors map[string]*MirrorConfig, hasConflict bool) {
	if hasConflict && cr.shouldKeepDeleted(remoteMirror.Name, remoteDeletedMirrors) {
		return // 保持删除状态
	}

	newMirror := *remoteMirror
	newMirror.APIKey = "" // 清空API密钥
	cr.setEnvKey(&newMirror)
	mergedMirrors[remoteMirror.Name] = newMirror
}

// shouldKeepDeleted 检查是否应该保持删除状态.
func (cr *ConflictResolver) shouldKeepDeleted(mirrorName string, remoteDeletedMirrors map[string]*MirrorConfig) bool {
	localDeletedMirror := cr.findLocalDeletedMirror(mirrorName)
	if localDeletedMirror != nil && cr.isIntentionalDeletion(localDeletedMirror, remoteDeletedMirrors) {
		fmt.Printf("🗑️  智能合并：保持删除状态 '%s'（本地主动删除）\n", mirrorName)
		return true
	}

	if remoteDeleted, wasRemoteDeleted := remoteDeletedMirrors[mirrorName]; wasRemoteDeleted && cr.isRecentlyDeleted(remoteDeleted) {
		fmt.Printf("🔄 智能合并：恢复镜像源 '%s'（云端删除后重新添加）\n", mirrorName)
	}

	return false
}

// setEnvKey 设置环境变量key.
func (cr *ConflictResolver) setEnvKey(mirror *MirrorConfig) {
	switch mirror.ToolType {
	case ToolTypeCodex:
		mirror.EnvKey = CodexSwitchAPIKeyEnv
	case ToolTypeClaude:
		mirror.EnvKey = AnthropicAuthTokenEnv
	}
}

// handleRemoteDeletedMirrors 处理云端已删除的镜像源.
func (cr *ConflictResolver) handleRemoteDeletedMirrors(mergedMirrors map[string]MirrorConfig, remoteDeletedMirrors map[string]*MirrorConfig, resolution *ConflictResolution) {
	for _, conflict := range resolution.Conflicts {
		if conflict.Type == ConflictTypeDeletedMirror && conflict.LocalMirror != nil {
			mirrorName := conflict.LocalMirror.Name
			if _, existsInMerged := mergedMirrors[mirrorName]; existsInMerged {
				if remoteDeleted, wasRemoteDeleted := remoteDeletedMirrors[mirrorName]; wasRemoteDeleted && cr.isRecentlyDeleted(remoteDeleted) {
					fmt.Printf("🗑️  智能合并：同步删除 '%s'（云端已删除）\n", mirrorName)
					delete(mergedMirrors, mirrorName)
				}
			}
		}
	}
}

// finalizeMergeConfig 完成合并配置.
func (cr *ConflictResolver) finalizeMergeConfig(config *SystemConfig, mergedMirrors map[string]MirrorConfig) {
	// 转换为数组
	config.Mirrors = make([]MirrorConfig, 0, len(mergedMirrors))
	for name := range mergedMirrors {
		mirror := mergedMirrors[name]
		config.Mirrors = append(config.Mirrors, mirror)
	}

	// 智能选择当前激活源
	cr.selectCurrentMirrors(config, mergedMirrors)
}

// selectCurrentMirrors 选择当前激活的镜像源.
func (cr *ConflictResolver) selectCurrentMirrors(config *SystemConfig, mergedMirrors map[string]MirrorConfig) {
	// 对于 Codex 镜像源
	if cr.localConfig.CurrentCodex != "" {
		// 优先保留本地的激活源
		if _, exists := mergedMirrors[cr.localConfig.CurrentCodex]; exists {
			config.CurrentCodex = cr.localConfig.CurrentCodex
		} else if cr.remoteData.CurrentCodex != "" {
			// 如果本地激活源不存在，尝试使用云端的激活源
			if _, exists := mergedMirrors[cr.remoteData.CurrentCodex]; exists {
				config.CurrentCodex = cr.remoteData.CurrentCodex
			} else {
				config.CurrentCodex = cr.selectDefaultMirror(mergedMirrors, ToolTypeCodex)
			}
		} else {
			config.CurrentCodex = cr.selectDefaultMirror(mergedMirrors, ToolTypeCodex)
		}
	} else if cr.remoteData.CurrentCodex != "" {
		// 如果本地没有激活源，使用云端的激活源
		if _, exists := mergedMirrors[cr.remoteData.CurrentCodex]; exists {
			config.CurrentCodex = cr.remoteData.CurrentCodex
		} else {
			config.CurrentCodex = cr.selectDefaultMirror(mergedMirrors, ToolTypeCodex)
		}
	} else {
		// 都没有的话选择默认的
		config.CurrentCodex = cr.selectDefaultMirror(mergedMirrors, ToolTypeCodex)
	}

	// 对于 Claude 镜像源
	if cr.localConfig.CurrentClaude != "" {
		// 优先保留本地的激活源
		if _, exists := mergedMirrors[cr.localConfig.CurrentClaude]; exists {
			config.CurrentClaude = cr.localConfig.CurrentClaude
		} else if cr.remoteData.CurrentClaude != "" {
			// 如果本地激活源不存在，尝试使用云端的激活源
			if _, exists := mergedMirrors[cr.remoteData.CurrentClaude]; exists {
				config.CurrentClaude = cr.remoteData.CurrentClaude
			} else {
				config.CurrentClaude = cr.selectDefaultMirror(mergedMirrors, ToolTypeClaude)
			}
		} else {
			config.CurrentClaude = cr.selectDefaultMirror(mergedMirrors, ToolTypeClaude)
		}
	} else if cr.remoteData.CurrentClaude != "" {
		// 如果本地没有激活源，使用云端的激活源
		if _, exists := mergedMirrors[cr.remoteData.CurrentClaude]; exists {
			config.CurrentClaude = cr.remoteData.CurrentClaude
		} else {
			config.CurrentClaude = cr.selectDefaultMirror(mergedMirrors, ToolTypeClaude)
		}
	} else {
		// 都没有的话选择默认的
		config.CurrentClaude = cr.selectDefaultMirror(mergedMirrors, ToolTypeClaude)
	}
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
