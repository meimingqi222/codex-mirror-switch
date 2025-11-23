package internal

import (
	"bufio"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SyncManager 云同步管理器.
type SyncManager struct {
	mirrorManager *MirrorManager
	provider      SyncProvider
	config        *SyncConfig
}

// NewSyncManager 创建新的同步管理器.
func NewSyncManager(mirrorManager *MirrorManager) *SyncManager {
	return &SyncManager{
		mirrorManager: mirrorManager,
	}
}

// InitSync 初始化云同步.
func (sm *SyncManager) InitSync(providerType, endpoint, token string) error {
	return sm.InitSyncWithOptions(providerType, endpoint, token, false)
}

// InitSyncWithPassword 使用密码初始化云同步.
func (sm *SyncManager) InitSyncWithPassword(providerType, endpoint, token, password string) error {
	return sm.InitSyncWithPasswordAndGist(providerType, endpoint, token, password, "")
}

// InitSyncWithPasswordAndGist 使用密码和可选的Gist ID初始化云同步.
func (sm *SyncManager) InitSyncWithPasswordAndGist(providerType, endpoint, token, password, gistID string) error {
	// 生成设备ID
	deviceID := generateDeviceID()

	// 创建同步配置
	syncConfig := &SyncConfig{
		Enabled:       true,
		Provider:      providerType,
		Endpoint:      endpoint,
		Token:         token,
		EncryptKey:    "", // 不再使用随机密钥
		AutoSync:      false,
		SyncInterval:  30,
		DeviceID:      deviceID,
		LastSync:      time.Time{},
		SyncAPIKeys:   true,     // 默认总是同步API密钥
		EncryptionPwd: password, // 使用用户提供的密码
		GistID:        gistID,   // 可选的现有Gist ID
	}

	// 创建提供商实例
	provider, err := sm.createProvider(syncConfig)
	if err != nil {
		return fmt.Errorf("创建同步提供商失败: %w", err)
	}

	sm.config = syncConfig
	sm.provider = provider

	// 如果提供商自动发现了Gist ID，更新配置
	if gistProvider, ok := provider.(*GistProvider); ok {
		if discoveredID := gistProvider.GetGistID(); discoveredID != "" && syncConfig.GistID == "" {
			syncConfig.GistID = discoveredID
			fmt.Printf("🔍 自动发现现有配置 Gist: %s\n", discoveredID)

			// 验证密码是否正确
			if err := sm.validatePassword(); err != nil {
				return fmt.Errorf("密码验证失败: %w\n\n💡 可能原因:\n   - 密码输入错误\n   - 此Gist使用了不同的密码\n\n🔧 解决方法:\n   - 检查密码是否正确\n   - 或使用 --gist-id 参数指定新的Gist", err)
			}
			fmt.Printf("✅ 密码验证成功，可以正常同步现有配置\n")
		}
	}

	// 保存同步配置到系统配置
	sm.mirrorManager.config.Sync = syncConfig
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存同步配置失败: %w", err)
	}

	fmt.Printf("✅ 云同步初始化成功\n")
	fmt.Printf("   提供商: %s\n", providerType)
	fmt.Printf("   设备ID: %s\n", deviceID)
	fmt.Printf("   端点: %s\n", endpoint)
	fmt.Printf("   全量同步: 启用\n")

	if syncConfig.GistID != "" {
		fmt.Printf("   Gist ID: %s\n", syncConfig.GistID)
		fmt.Printf("   💡 可以直接使用 'codex-mirror sync pull' 拉取现有配置\n")
	} else {
		fmt.Printf("   💡 使用 'codex-mirror sync push' 创建新的云端配置\n")
	}

	return nil
}

// InitSyncWithOptions 初始化云同步（带选项）- 保持向后兼容.
func (sm *SyncManager) InitSyncWithOptions(providerType, endpoint, token string, syncAPIKeys bool) error {
	// 生成设备ID
	deviceID := generateDeviceID()

	// 生成加密密钥
	encryptKey, err := generateEncryptKey()
	if err != nil {
		return fmt.Errorf("生成加密密钥失败: %w", err)
	}

	// 创建同步配置
	syncConfig := &SyncConfig{
		Enabled:      true,
		Provider:     providerType,
		Endpoint:     endpoint,
		Token:        token,
		EncryptKey:   encryptKey,
		AutoSync:     false,
		SyncInterval: 30,
		DeviceID:     deviceID,
		LastSync:     time.Time{},
		SyncAPIKeys:  syncAPIKeys,
	}

	// 创建提供商实例
	provider, err := sm.createProvider(syncConfig)
	if err != nil {
		return fmt.Errorf("创建同步提供商失败: %w", err)
	}

	sm.config = syncConfig
	sm.provider = provider

	// 保存同步配置到系统配置
	sm.mirrorManager.config.Sync = syncConfig
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存同步配置失败: %w", err)
	}

	fmt.Printf("✅ 云同步初始化成功\n")
	fmt.Printf("   提供商: %s\n", providerType)
	fmt.Printf("   设备ID: %s\n", deviceID)
	fmt.Printf("   端点: %s\n", endpoint)
	if syncAPIKeys {
		fmt.Printf("   API密钥同步: 是\n")
	} else {
		fmt.Printf("   API密钥同步: 否\n")
	}

	return nil
}

// LoadSync 加载同步配置.
func (sm *SyncManager) LoadSync() error {
	if sm.mirrorManager.config.Sync == nil {
		return fmt.Errorf("未配置云同步")
	}

	sm.config = sm.mirrorManager.config.Sync

	// 创建提供商实例
	provider, err := sm.createProvider(sm.config)
	if err != nil {
		return fmt.Errorf("创建同步提供商失败: %w", err)
	}

	sm.provider = provider
	return nil
}

// Push 推送配置到云端.
func (sm *SyncManager) Push() error {
	return sm.PushWithStrategy("auto")
}

// PushWithStrategy 使用指定策略推送配置到云端.
func (sm *SyncManager) PushWithStrategy(strategy string) error {
	if err := sm.LoadSync(); err != nil {
		return err
	}

	// 推送前自动备份
	if err := sm.createBackupWithPrefix("pre-push"); err != nil {
		fmt.Printf("⚠️  创建备份失败: %v（继续推送）\n", err)
	}

	fmt.Printf("📤 正在推送配置到云端...\n")

	// 首先检查是否存在云端配置，如果存在则进行冲突检查
	filename := ConfigFileName
	if encryptedRemoteData, err := sm.provider.Download(filename); err == nil {
		fmt.Printf("🔍 检查云端配置冲突...\n")
		// 解密远程数据
		if remoteData, err := sm.decryptData(encryptedRemoteData); err == nil {
			var remoteSyncData SyncData
			if err := json.Unmarshal(remoteData, &remoteSyncData); err == nil {
				// 解密所有远程镜像源的 APIKey（在冲突检测之前）
				if err := sm.decryptSyncDataAPIKeys(&remoteSyncData); err != nil {
					fmt.Printf("⚠️  解密远程 API 密钥失败: %v（继续推送）\n", err)
				}

				// 检测冲突
				resolver := NewConflictResolver(sm.mirrorManager.config, &remoteSyncData)
				conflicts := resolver.DetectConflicts()

				if len(conflicts.Conflicts) > 0 {
					// 有冲突，根据策略处理
					return sm.handlePushConflicts(resolver, conflicts, strategy, &remoteSyncData)
				} else {
					fmt.Printf("✅ 无配置冲突，直接推送\n")
				}
			}
		}
	} else {
		fmt.Printf("💡 云端暂无配置，首次推送\n")
	}

	// 没有冲突或首次推送，直接上传
	return sm.performPush(filename)
}

// handlePushConflicts 处理推送时的配置冲突.
func (sm *SyncManager) handlePushConflicts(resolver *ConflictResolver, conflicts *ConflictResolution, strategy string, remoteSyncData *SyncData) error {
	fmt.Printf("⚠️  检测到推送冲突\n\n")
	fmt.Printf("🔍 云端配置信息:\n")
	fmt.Printf("   来源设备: %s\n", remoteSyncData.DeviceID)
	fmt.Printf("   配置时间: %s\n", remoteSyncData.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("   镜像源数量: %d\n\n", len(remoteSyncData.Mirrors))
	fmt.Printf("%s", resolver.FormatConflicts(conflicts))

	switch strategy {
	case "auto", StrategyMerge:
		return sm.handlePushAutoResolve(resolver, conflicts, remoteSyncData)
	case "force":
		fmt.Printf("🚀 强制推送模式，覆盖云端配置...\n")
		return sm.performPush(ConfigFileName)
	case "manual":
		return fmt.Errorf("检测到配置冲突，请选择解决策略:\n\n" +
			"  codex-mirror sync push --strategy=force  # 强制覆盖云端配置\n" +
			"  codex-mirror sync push --strategy=merge  # 智能合并后推送\n" +
			"  codex-mirror sync pull --strategy=merge  # 先拉取合并，再推送\n\n" +
			"💡 建议先使用 pull --strategy=merge 合并云端配置，再推送")
	default:
		return fmt.Errorf("不支持的推送策略: %s", strategy)
	}
}

// handlePushAutoResolve 自动解决推送冲突.
func (sm *SyncManager) handlePushAutoResolve(resolver *ConflictResolver, conflicts *ConflictResolution, _ *SyncData) error {
	fmt.Printf("🔄 自动合并本地和云端配置...\n")

	// 使用合并策略解决冲突
	resolvedConfig, err := resolver.ResolveConflicts(conflicts, StrategyMerge)
	if err != nil {
		return fmt.Errorf("自动解决冲突失败: %w", err)
	}

	// 创建备份
	if err := sm.createBackup(); err != nil {
		fmt.Printf("警告: 创建备份失败: %v\n", err)
	}

	// 应用解决后的配置
	sm.mirrorManager.config = resolvedConfig
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存解决后的配置失败: %w", err)
	}

	fmt.Printf("✅ 冲突已自动解决（智能合并）\n")
	fmt.Printf("   - 优先保留本地配置修改（URL、模型名等）\n")
	fmt.Printf("   - 保留了本地API密钥\n")
	fmt.Printf("   - 合并了云端新增的镜像源\n\n")

	// 现在推送合并后的配置
	return sm.performPush(ConfigFileName)
}

// performPush 执行实际的推送操作.
func (sm *SyncManager) performPush(filename string) error {
	// 导出同步数据
	syncData := sm.exportSyncData()

	// 序列化数据
	data, err := json.MarshalIndent(syncData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化同步数据失败: %w", err)
	}

	// 加密数据
	encryptedData, err := sm.encryptData(data)
	if err != nil {
		return fmt.Errorf("加密数据失败: %w", err)
	}

	// 上传到云端
	if err := sm.provider.Upload(encryptedData, filename); err != nil {
		return fmt.Errorf("上传配置失败: %w", err)
	}

	// 保存 Gist ID（如果是新创建的）
	if gistProvider, ok := sm.provider.(*GistProvider); ok {
		if gistID := gistProvider.GetGistID(); gistID != "" && sm.config.GistID == "" {
			sm.config.GistID = gistID
			sm.mirrorManager.config.Sync.GistID = gistID
		}
	}

	// 更新最后同步时间
	sm.config.LastSync = time.Now()
	sm.mirrorManager.config.Sync = sm.config
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存同步时间失败: %w", err)
	}

	fmt.Printf("✅ 配置已推送到云端\n")
	fmt.Printf("   文件: %s\n", filename)
	fmt.Printf("   时间: %s\n", sm.config.LastSync.Format("2006-01-02 15:04:05"))
	fmt.Printf("   镜像源数量: %d\n", len(sm.mirrorManager.config.Mirrors))
	fmt.Printf("   数据已加密: 是\n")

	return nil
}

// Pull 从云端拉取配置.
func (sm *SyncManager) Pull() error {
	return sm.PullWithStrategy("auto")
}

// PullWithStrategy 使用指定策略从云端拉取配置.
func (sm *SyncManager) PullWithStrategy(strategy string) error {
	if err := sm.LoadSync(); err != nil {
		return err
	}

	// 拉取前自动备份
	if err := sm.createBackupWithPrefix("pre-pull"); err != nil {
		fmt.Printf("⚠️  创建备份失败: %v（继续拉取）\n", err)
	}

	// 直接使用标准配置文件名
	filename := ConfigFileName
	fmt.Printf("📥 正在从云端拉取配置...\n")

	// 下载数据
	encryptedData, err := sm.provider.Download(filename)
	if err != nil {
		return fmt.Errorf("下载配置失败: %w", err)
	}

	// 解密数据
	data, err := sm.decryptData(encryptedData)
	if err != nil {
		return fmt.Errorf("解密数据失败: %w", err)
	}

	// 解析同步数据
	var syncData SyncData
	if err := json.Unmarshal(data, &syncData); err != nil {
		return fmt.Errorf("解析同步数据失败: %w", err)
	}

	// 解密所有远程镜像源的 APIKey（在冲突检测之前）
	if err := sm.decryptSyncDataAPIKeys(&syncData); err != nil {
		return fmt.Errorf("解密远程 API 密钥失败: %w", err)
	}

	// 检测冲突
	fmt.Printf("🔍 检查配置冲突...\n")
	resolver := NewConflictResolver(sm.mirrorManager.config, &syncData)
	conflicts := resolver.DetectConflicts()

	if len(conflicts.Conflicts) > 0 {
		// 有冲突，根据策略处理
		return sm.handleConflicts(resolver, conflicts, strategy, &syncData)
	} else {
		fmt.Printf("✅ 无配置冲突，直接应用\n")
	}

	// 没有冲突，直接应用
	if err := sm.applySyncData(&syncData); err != nil {
		return fmt.Errorf("应用同步数据失败: %w", err)
	}

	// 更新最后同步时间
	sm.config.LastSync = time.Now()
	sm.mirrorManager.config.Sync = sm.config
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存同步时间失败: %w", err)
	}

	fmt.Printf("✅ 配置已从云端拉取并应用\n")
	fmt.Printf("   来源设备: %s\n", syncData.DeviceID)
	fmt.Printf("   配置时间: %s\n", syncData.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("   镜像源数量: %d\n", len(syncData.Mirrors))
	fmt.Printf("   数据已解密: 是\n")

	return nil
}

// FetchRemoteSyncData 仅获取云端同步数据（不应用到本地）。
func (sm *SyncManager) FetchRemoteSyncData() (*SyncData, error) {
	// 确保提供商已初始化
	if err := sm.LoadSync(); err != nil {
		return nil, err
	}

	filename := ConfigFileName

	// 下载远端数据
	encryptedData, err := sm.provider.Download(filename)
	if err != nil {
		return nil, fmt.Errorf("下载配置失败: %w", err)
	}

	// 解密
	data, err := sm.decryptData(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("解密数据失败: %w", err)
	}

	// 解析 JSON
	var syncData SyncData
	if err := json.Unmarshal(data, &syncData); err != nil {
		return nil, fmt.Errorf("解析同步数据失败: %w", err)
	}

	return &syncData, nil
}

// handleConflicts 处理配置冲突.
func (sm *SyncManager) handleConflicts(resolver *ConflictResolver, conflicts *ConflictResolution, strategy string, syncData *SyncData) error {
	fmt.Printf("⚠️  检测到配置冲突\n\n")
	fmt.Printf("%s", resolver.FormatConflicts(conflicts))

	var resolvedConfig *SystemConfig
	var err error

	switch strategy {
	case "auto", StrategyMerge:
		fmt.Printf("🔄 使用智能合并策略解决冲突...\n")
		resolvedConfig, err = resolver.ResolveConflicts(conflicts, StrategyMerge)
		if err != nil {
			return fmt.Errorf("自动解决冲突失败: %w", err)
		}

		fmt.Printf("✅ 冲突已自动解决（智能合并）\n")
		fmt.Printf("   - 保留了本地API密钥\n")
		fmt.Printf("   - 合并了镜像源配置\n")
		fmt.Printf("   - 新增镜像源需要手动配置API密钥\n")

	case "local":
		fmt.Printf("🏠 使用本地优先策略解决冲突...\n")
		resolvedConfig, err = resolver.ResolveConflicts(conflicts, "local")
		if err != nil {
			return fmt.Errorf("本地优先解决冲突失败: %w", err)
		}

		fmt.Printf("✅ 冲突已解决（本地优先）\n")
		fmt.Printf("   - 保持本地配置不变\n")
		fmt.Printf("   - 添加了云端新增的镜像源\n")

	case "remote":
		fmt.Printf("☁️  使用远程优先策略解决冲突...\n")
		resolvedConfig, err = resolver.ResolveConflicts(conflicts, "remote")
		if err != nil {
			return fmt.Errorf("远程优先解决冲突失败: %w", err)
		}

		fmt.Printf("✅ 冲突已解决（远程优先）\n")
		fmt.Printf("   - 使用云端配置\n")
		fmt.Printf("   - 保留了本地API密钥\n")

	case "manual":
		return sm.handleManualConflictResolution(resolver, conflicts, syncData)

	default:
		return fmt.Errorf("不支持的冲突解决策略: %s", strategy)
	}

	// 创建备份
	if err := sm.createBackup(); err != nil {
		fmt.Printf("警告: 创建备份失败: %v\n", err)
	}

	// 应用解决后的配置
	sm.mirrorManager.config = resolvedConfig
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存解决后的配置失败: %w", err)
	}

	// 更新最后同步时间
	sm.config.LastSync = time.Now()
	sm.mirrorManager.config.Sync = sm.config
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存同步时间失败: %w", err)
	}

	fmt.Printf("\n📊 同步完成统计:\n")
	fmt.Printf("   来源设备: %s\n", syncData.DeviceID)
	fmt.Printf("   配置时间: %s\n", syncData.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("   镜像源数量: %d\n", len(resolvedConfig.Mirrors))
	fmt.Printf("   解决冲突: %d个\n", len(conflicts.Conflicts))

	return nil
}

// createBackup 创建配置备份.
func (sm *SyncManager) createBackup() error {
	return sm.createBackupWithPrefix("backup")
}

// createBackupWithPrefix 使用指定前缀创建配置备份.
func (sm *SyncManager) createBackupWithPrefix(prefix string) error {
	// 使用 MirrorManager 的配置路径，而不是硬编码的系统路径
	configPath := sm.mirrorManager.GetConfigPath()
	configDir := filepath.Dir(configPath)
	backupDir := filepath.Join(configDir, "backup")

	if err := EnsureDir(backupDir); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 生成备份文件名
	timestamp := time.Now().Format("20060102-150405")
	backupFileName := fmt.Sprintf("%s-%s.toml", prefix, timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	// 复制当前配置文件
	if err := copyFile(configPath, backupPath); err != nil {
		return fmt.Errorf("创建备份失败: %w", err)
	}

	fmt.Printf("💾 已备份配置: %s\n", backupPath)

	// 清理旧备份（保留最近10个）
	sm.cleanOldBackups(backupDir, prefix, 10)

	return nil
}

// cleanOldBackups 清理旧备份文件，保留指定数量的最新备份.
func (sm *SyncManager) cleanOldBackups(backupDir, prefix string, keepCount int) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	// 筛选匹配前缀的备份文件
	var backupFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix+"-") && strings.HasSuffix(entry.Name(), ".toml") {
			backupFiles = append(backupFiles, entry)
		}
	}

	// 如果备份数量超过限制，删除最旧的
	if len(backupFiles) > keepCount {
		// 按名称排序（时间戳格式保证字典序等于时间序）
		// 删除最旧的
		for i := 0; i < len(backupFiles)-keepCount; i++ {
			oldFile := backupDir + "/" + backupFiles[i].Name()
			_ = os.Remove(oldFile)
		}
	}
}

// GetStatus 获取同步状态.
func (sm *SyncManager) GetStatus() (*SyncStatus, error) {
	if sm.mirrorManager.config.Sync == nil {
		return &SyncStatus{
			Enabled: false,
			Message: "未配置云同步",
		}, nil
	}

	config := sm.mirrorManager.config.Sync
	status := &SyncStatus{
		Enabled:      config.Enabled,
		Provider:     config.Provider,
		Endpoint:     config.Endpoint,
		DeviceID:     config.DeviceID,
		AutoSync:     config.AutoSync,
		SyncInterval: config.SyncInterval,
		LastSync:     config.LastSync,
	}

	if config.LastSync.IsZero() {
		status.Message = "尚未进行过同步"
	} else {
		duration := time.Since(config.LastSync)
		status.Message = fmt.Sprintf("上次同步: %s 前", formatDuration(duration))
	}

	return status, nil
}

// exportSyncData 导出同步数据.
func (sm *SyncManager) exportSyncData() *SyncData {
	var mirrors []MirrorConfig
	var deletedMirrors []MirrorConfig

	// 总是包含API密钥（加密后）
	for i := range sm.mirrorManager.config.Mirrors {
		mirror := &sm.mirrorManager.config.Mirrors[i]
		exportMirror := *mirror

		// 如果有API密钥，进行加密
		if mirror.APIKey != "" {
			encryptedKey, err := sm.encryptAPIKey(mirror.APIKey)
			if err != nil {
				fmt.Printf("警告: 加密API密钥失败 (%s): %v\n", mirror.Name, err)
				// 如果加密失败，不包含API密钥
				exportMirror.APIKey = ""
			} else {
				exportMirror.APIKey = encryptedKey
			}
		}

		// 确保时间戳不为零值
		if exportMirror.CreatedAt.IsZero() {
			exportMirror.CreatedAt = time.Now()
		}
		if exportMirror.LastModified.IsZero() {
			exportMirror.LastModified = exportMirror.CreatedAt
		}

		// 分离已删除和活跃的镜像源
		if exportMirror.Deleted && !exportMirror.DeletedAt.IsZero() {
			// 已删除的镜像源
			deletedMirrors = append(deletedMirrors, exportMirror)
		} else {
			// 活跃的镜像源
			mirrors = append(mirrors, exportMirror)
		}
	}

	// 计算数据校验和
	data, _ := json.Marshal(mirrors)
	checksum := calculateChecksum(data)

	return &SyncData{
		Mirrors:        mirrors,
		CurrentCodex:   sm.mirrorManager.config.CurrentCodex,
		CurrentClaude:  sm.mirrorManager.config.CurrentClaude,
		Timestamp:      time.Now(),
		DeviceID:       sm.config.DeviceID,
		Version:        "3.1", // 支持删除追踪的新版本
		Checksum:       checksum,
		HasAPIKeys:     true,           // 总是为true
		DeletedMirrors: deletedMirrors, // 包含已删除的镜像源信息
	}
}

// applySyncData 应用同步数据.
func (sm *SyncManager) applySyncData(syncData *SyncData) error {
	// 验证校验和
	data, _ := json.Marshal(syncData.Mirrors)
	if calculateChecksum(data) != syncData.Checksum {
		return fmt.Errorf("数据校验和不匹配，可能数据已损坏")
	}

	// 备份当前配置
	backupMirrors := make([]MirrorConfig, len(sm.mirrorManager.config.Mirrors))
	copy(backupMirrors, sm.mirrorManager.config.Mirrors)

	// 应用新的镜像源配置
	var newMirrors []MirrorConfig
	for i := range syncData.Mirrors {
		mirror := &syncData.Mirrors[i]
		newMirror := *mirror

		// 解密API密钥
		if mirror.APIKey != "" {
			decryptedKey, err := sm.decryptAPIKey(mirror.APIKey)
			if err != nil {
				return fmt.Errorf("解密API密钥失败 (%s): %w", mirror.Name, err)
			}
			newMirror.APIKey = decryptedKey
		}

		// 设置环境变量key
		switch mirror.ToolType {
		case ToolTypeCodex:
			newMirror.EnvKey = CodexSwitchAPIKeyEnv
		case ToolTypeClaude:
			newMirror.EnvKey = AnthropicAuthTokenEnv
		}

		newMirrors = append(newMirrors, newMirror)
	}

	// 更新配置
	sm.mirrorManager.config.Mirrors = newMirrors

	// 优先保留本地激活源配置，只有在本地没有设置时才使用云端的
	if sm.mirrorManager.config.CurrentCodex == "" && syncData.CurrentCodex != "" {
		sm.mirrorManager.config.CurrentCodex = syncData.CurrentCodex
	}
	if sm.mirrorManager.config.CurrentClaude == "" && syncData.CurrentClaude != "" {
		sm.mirrorManager.config.CurrentClaude = syncData.CurrentClaude
	}

	// 保存配置
	if err := sm.mirrorManager.saveConfig(); err != nil {
		// 恢复备份
		sm.mirrorManager.config.Mirrors = backupMirrors
		return fmt.Errorf("保存配置失败: %w", err)
	}

	return nil
}

// createProvider 创建同步提供商.
func (sm *SyncManager) createProvider(config *SyncConfig) (SyncProvider, error) {
	switch config.Provider {
	case "gist":
		return NewGistProvider(config.Token, config.GistID)
	default:
		return nil, fmt.Errorf("不支持的同步提供商: %s", config.Provider)
	}
}

// encryptData 加密数据.
func (sm *SyncManager) encryptData(data []byte) ([]byte, error) {
	// 优先使用用户设置的密码，否则使用随机密钥
	password := sm.config.EncryptionPwd
	if password == "" {
		password = sm.config.EncryptKey
	}

	if password == "" {
		return nil, fmt.Errorf("未设置加密密码")
	}

	crypto := NewCryptoManager(password)
	return crypto.Encrypt(data)
}

// decryptData 解密数据.
func (sm *SyncManager) decryptData(encryptedData []byte) ([]byte, error) {
	// 优先使用用户设置的密码，否则使用随机密钥
	password := sm.config.EncryptionPwd
	if password == "" {
		password = sm.config.EncryptKey
	}

	if password == "" {
		return nil, fmt.Errorf("未设置加密密码")
	}

	crypto := NewCryptoManager(password)
	return crypto.Decrypt(encryptedData)
}

// encryptAPIKey 加密单个API密钥.
func (sm *SyncManager) encryptAPIKey(apiKey string) (string, error) {
	encrypted, err := sm.encryptData([]byte(apiKey))
	if err != nil {
		return "", err
	}
	// 转换为hex字符串便于存储
	return fmt.Sprintf("enc:%x", encrypted), nil
}

// decryptAPIKey 解密单个API密钥.
func (sm *SyncManager) decryptAPIKey(encryptedKey string) (string, error) {
	// 检查是否是加密的密钥
	if !strings.HasPrefix(encryptedKey, "enc:") {
		return encryptedKey, nil // 未加密的密钥直接返回
	}

	// 移除前缀并解码hex
	hexData := strings.TrimPrefix(encryptedKey, "enc:")
	encrypted, err := hex.DecodeString(hexData)
	if err != nil {
		return "", fmt.Errorf("解码加密数据失败: %w", err)
	}

	// 解密
	decrypted, err := sm.decryptData(encrypted)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// decryptSyncDataAPIKeys 解密同步数据中所有镜像源的 APIKey.
func (sm *SyncManager) decryptSyncDataAPIKeys(syncData *SyncData) error {
	// 解密活跃镜像源的 APIKey
	for i := range syncData.Mirrors {
		mirror := &syncData.Mirrors[i]
		if mirror.APIKey != "" {
			decryptedKey, err := sm.decryptAPIKey(mirror.APIKey)
			if err != nil {
				return fmt.Errorf("解密镜像源 '%s' 的 API 密钥失败: %w", mirror.Name, err)
			}
			mirror.APIKey = decryptedKey
		}
	}

	// 解密已删除镜像源的 APIKey
	for i := range syncData.DeletedMirrors {
		mirror := &syncData.DeletedMirrors[i]
		if mirror.APIKey != "" {
			decryptedKey, err := sm.decryptAPIKey(mirror.APIKey)
			if err != nil {
				// 已删除的镜像源解密失败不影响主流程
				fmt.Printf("⚠️  解密已删除镜像源 '%s' 的 API 密钥失败: %v\n", mirror.Name, err)
				mirror.APIKey = ""
			} else {
				mirror.APIKey = decryptedKey
			}
		}
	}

	return nil
}

// validatePassword 验证用户密码是否能正确解密现有配置.
func (sm *SyncManager) validatePassword() error {
	if sm.provider == nil {
		return fmt.Errorf("同步提供商未初始化")
	}

	// 尝试下载现有配置
	filename := ConfigFileName
	encryptedData, err := sm.provider.Download(filename)
	if err != nil {
		// 如果下载失败，可能是第一次设置，不需要验证
		return nil
	}

	// 尝试解密数据来验证密码
	_, err = sm.decryptData(encryptedData)
	if err != nil {
		return fmt.Errorf("无法解密现有配置")
	}

	return nil
}

// SyncStatus 同步状态.
type SyncStatus struct {
	Enabled      bool      `json:"enabled"`
	Provider     string    `json:"provider"`
	Endpoint     string    `json:"endpoint"`
	DeviceID     string    `json:"device_id"`
	AutoSync     bool      `json:"auto_sync"`
	SyncInterval int       `json:"sync_interval"`
	LastSync     time.Time `json:"last_sync"`
	Message      string    `json:"message"`
}

// 辅助函数

// generateDeviceID 生成设备ID.
func generateDeviceID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	hash := md5.Sum([]byte(hostname + "codex-mirror-device-v1"))
	suffix := hex.EncodeToString(hash[:4]) // 使用前4字节作为8位十六进制后缀

	return fmt.Sprintf("%s-%s", hostname, suffix)
}

// generateEncryptKey 生成加密密钥.
func generateEncryptKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// calculateChecksum 计算校验和.
func calculateChecksum(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// formatDuration 格式化时间间隔.
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.0f秒", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%.0f分钟", d.Minutes())
	case d < 24*time.Hour:
		return fmt.Sprintf("%.1f小时", d.Hours())
	default:
		return fmt.Sprintf("%.1f天", d.Hours()/24)
	}
}

// handleManualConflictResolution 处理手动冲突解决.
func (sm *SyncManager) handleManualConflictResolution(resolver *ConflictResolver, conflicts *ConflictResolution, syncData *SyncData) error {
	fmt.Printf("🤔 需要手动解决冲突，请选择解决策略:\n\n")

	fmt.Printf("可用策略:\n")
	fmt.Printf("  1. merge  - 智能合并本地和云端配置（推荐）\n")
	fmt.Printf("  2. local  - 保持本地配置优先，仅添加云端新增项\n")
	fmt.Printf("  3. remote - 使用云端配置优先，保留本地API密钥\n")
	fmt.Printf("  4. abort  - 取消操作，不应用任何更改\n\n")

	strategy := sm.promptUserChoice()
	if strategy == "" {
		return fmt.Errorf("用户取消操作")
	}

	if strategy == StrategyAbort {
		fmt.Printf("❌ 操作已取消，本地配置未更改\n")
		return fmt.Errorf("用户取消同步操作")
	}

	// 使用用户选择的策略解决冲突
	resolvedConfig, err := resolver.ResolveConflicts(conflicts, strategy)
	if err != nil {
		return fmt.Errorf("解决冲突失败: %w", err)
	}

	// 显示将要应用的更改
	fmt.Printf("\n📋 将要应用的更改:\n")
	sm.showConfigChanges(sm.mirrorManager.config, resolvedConfig)

	if !sm.confirmChanges() {
		fmt.Printf("❌ 操作已取消，本地配置未更改\n")
		return fmt.Errorf("用户取消应用更改")
	}

	// 创建备份
	if err := sm.createBackup(); err != nil {
		fmt.Printf("警告: 创建备份失败: %v\n", err)
	}

	// 应用解决后的配置
	sm.mirrorManager.config = resolvedConfig
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存解决后的配置失败: %w", err)
	}

	// 更新最后同步时间
	sm.config.LastSync = time.Now()
	sm.mirrorManager.config.Sync = sm.config
	if err := sm.mirrorManager.saveConfig(); err != nil {
		return fmt.Errorf("保存同步时间失败: %w", err)
	}

	fmt.Printf("\n✅ 冲突已解决并应用\n")
	fmt.Printf("   来源设备: %s\n", syncData.DeviceID)
	fmt.Printf("   配置时间: %s\n", syncData.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("   解决策略: %s\n", strategy)

	return nil
}

// promptUserChoice 提示用户选择策略.
func (sm *SyncManager) promptUserChoice() string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("请选择策略 [1-4] 或 [merge/local/remote/abort]: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入错误: %v\n", err)
			continue
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "1", "merge":
			return StrategyMerge
		case "2", "local":
			return StrategyLocal
		case "3", "remote":
			return StrategyRemote
		case "4", "abort":
			return StrategyAbort
		default:
			fmt.Printf("❌ 无效输入，请输入 1-4 或对应的策略名称\n")
		}
	}
}

// showConfigChanges 显示配置更改.
func (sm *SyncManager) showConfigChanges(currentConfig, newConfig *SystemConfig) {
	fmt.Printf("   镜像源变化:\n")

	// 创建映射便于比较
	currentMirrors := make(map[string]MirrorConfig)
	for i := range currentConfig.Mirrors {
		mirror := &currentConfig.Mirrors[i]
		currentMirrors[mirror.Name] = *mirror
	}

	newMirrors := make(map[string]MirrorConfig)
	for i := range newConfig.Mirrors {
		mirror := &newConfig.Mirrors[i]
		newMirrors[mirror.Name] = *mirror
	}

	// 检查新增的镜像源
	for name := range newMirrors {
		newMirror := newMirrors[name]
		if _, exists := currentMirrors[name]; !exists {
			fmt.Printf("     + 新增: %s (%s)\n", name, newMirror.BaseURL)
		}
	}

	// 检查删除的镜像源
	for name := range currentMirrors {
		currentMirror := currentMirrors[name]
		if _, exists := newMirrors[name]; !exists {
			fmt.Printf("     - 删除: %s (%s)\n", name, currentMirror.BaseURL)
		}
	}

	// 检查修改的镜像源
	for name := range newMirrors {
		newMirror := newMirrors[name]
		if currentMirror, exists := currentMirrors[name]; exists {
			if currentMirror.BaseURL != newMirror.BaseURL {
				fmt.Printf("     ~ 修改: %s (%s -> %s)\n", name, currentMirror.BaseURL, newMirror.BaseURL)
			}
		}
	}

	// 检查当前激活源变化
	if currentConfig.CurrentCodex != newConfig.CurrentCodex {
		fmt.Printf("   当前Codex镜像: %s -> %s\n", currentConfig.CurrentCodex, newConfig.CurrentCodex)
	}
	if currentConfig.CurrentClaude != newConfig.CurrentClaude {
		fmt.Printf("   当前Claude镜像: %s -> %s\n", currentConfig.CurrentClaude, newConfig.CurrentClaude)
	}
}

// confirmChanges 确认是否应用更改.
func (sm *SyncManager) confirmChanges() bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n是否应用这些更改? [y/N]: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入错误: %v\n", err)
			continue
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y", "yes", "是":
			return true
		case "n", "no", "否", "":
			return false
		default:
			fmt.Printf("❌ 请输入 y (是) 或 n (否)\n")
		}
	}
}
