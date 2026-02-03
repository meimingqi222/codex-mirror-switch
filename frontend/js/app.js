// 等待 Wails runtime 加载完成
function waitForWails() {
    return new Promise((resolve) => {
        if (window.go && window.go.main && window.go.main.App) {
            resolve();
        } else {
            const checkInterval = setInterval(() => {
                if (window.go && window.go.main && window.go.main.App) {
                    clearInterval(checkInterval);
                    resolve();
                }
            }, 50);
        }
    });
}

// Alpine.js 应用
function mirrorApp() {
    return {
        mirrors: [],
        status: {
            currentCodex: '',
            currentClaude: '',
            codexStatus: { exists: false, path: '' },
            claudeStatus: { exists: false, path: '' },
            vscodeStatus: { exists: false, path: '' },
            configPath: ''
        },
        loading: false,
        filterType: 'all',

        // 表单状态
        showForm: false,
        formMode: 'add',
        originalAPIKey: '', // 保存编辑时的原始API Key
        form: {
            name: '',
            base_url: '',
            api_key: '',
            tool_type: 'codex',
            model_name: '',
            extra_env: []
        },
        saving: false,
        errors: {},

        // 删除确认
        showDeleteConfirm: false,
        deleteTarget: null,

        // 通知
        toasts: [],
        toastId: 0,

        // 云同步状态
        syncStatus: {
            enabled: false,
            provider: '',
            deviceId: '',
            message: ''
        },
        syncing: false,
        showSyncModal: false,
        syncForm: {
            token: '',
            password: '',
            gistId: '',
            newPassword: '',
            newGistId: ''
        },

        async init() {
            await waitForWails();
            await this.refreshMirrors();
            await this.refreshStatus();
            await this.refreshSyncStatus();
        },

        // 获取过滤后的镜像列表
        get filteredMirrors() {
            if (this.filterType === 'all') {
                return this.mirrors;
            }
            return this.mirrors.filter(m => m.tool_type === this.filterType);
        },

        // 刷新镜像列表
        async refreshMirrors() {
            this.loading = true;
            try {
                const mirrors = await window.go.main.App.ListMirrors();
                this.mirrors = mirrors || [];
            } catch (error) {
                this.showToast('加载镜像列表失败: ' + error, 'error');
            } finally {
                this.loading = false;
            }
        },

        // 刷新状态
        async refreshStatus() {
            try {
                const status = await window.go.main.App.GetCurrentStatus();
                if (status) {
                    this.status = status;
                }
            } catch (error) {
                console.error('获取状态失败:', error);
            }
        },

        // 显示添加表单
        showAddForm() {
            this.formMode = 'add';
            this.originalAPIKey = ''; // 清空原始API Key
            this.form = {
                name: '',
                base_url: '',
                api_key: '',
                tool_type: 'codex',
                model_name: '',
                extra_env: []
            };
            this.errors = {};
            this.showForm = true;
        },

        // 添加扩展参数
        addExtraEnv() {
            this.form.extra_env.push({ key: '', value: '' });
        },

        // 删除扩展参数
        removeExtraEnv(index) {
            this.form.extra_env.splice(index, 1);
        },

        // 复制镜像配置（基于现有配置创建新的）
        async duplicateMirror(mirror) {
            this.formMode = 'add';
            this.originalAPIKey = ''; // 复制时不需要保留原始API Key
            // 获取完整的镜像信息
            try {
                const fullMirror = await window.go.main.App.GetMirror(mirror.name);
                // 转换 extra_env 对象为数组
                const extraEnvArray = [];
                if (fullMirror.extra_env) {
                    for (const [key, value] of Object.entries(fullMirror.extra_env)) {
                        extraEnvArray.push({ key, value });
                    }
                }
                // 预填表单，但名称加上"-副本"后缀
                this.form = {
                    name: fullMirror.name + '-副本',
                    base_url: fullMirror.base_url,
                    api_key: fullMirror.has_api_key ? '***' : '',
                    tool_type: fullMirror.tool_type,
                    model_name: fullMirror.model_name || '',
                    extra_env: extraEnvArray
                };
            } catch (error) {
                // 如果获取失败，使用列表中的数据
                const extraEnvArray = [];
                if (mirror.extra_env) {
                    for (const [key, value] of Object.entries(mirror.extra_env)) {
                        extraEnvArray.push({ key, value });
                    }
                }
                this.form = {
                    name: mirror.name + '-副本',
                    base_url: mirror.base_url,
                    api_key: '',
                    tool_type: mirror.tool_type,
                    model_name: mirror.model_name || '',
                    extra_env: extraEnvArray
                };
            }
            this.errors = {};
            this.showForm = true;
        },

        // 编辑镜像
        async editMirror(mirror) {
            this.formMode = 'edit';
            // 获取完整的镜像信息（包括未掩码的 API Key）
            try {
                const fullMirror = await window.go.main.App.GetMirror(mirror.name);
                // 保存原始API Key用于后续比较
                this.originalAPIKey = fullMirror.api_key || '';
                // 转换 extra_env 对象为数组
                const extraEnvArray = [];
                if (fullMirror.extra_env) {
                    for (const [key, value] of Object.entries(fullMirror.extra_env)) {
                        extraEnvArray.push({ key, value });
                    }
                }
                this.form = {
                    name: fullMirror.name,
                    base_url: fullMirror.base_url,
                    api_key: fullMirror.has_api_key ? '***' : '', // 显示掩码
                    tool_type: fullMirror.tool_type,
                    model_name: fullMirror.model_name || '',
                    extra_env: extraEnvArray
                };
            } catch (error) {
                this.originalAPIKey = '';
                this.form = {
                    name: mirror.name,
                    base_url: mirror.base_url,
                    api_key: '',
                    tool_type: mirror.tool_type,
                    model_name: mirror.model_name || '',
                    extra_env: []
                };
            }
            this.errors = {};
            this.showForm = true;
        },

        // 隐藏表单
        hideForm() {
            this.showForm = false;
            this.originalAPIKey = ''; // 清理原始API Key
            this.form = {
                name: '',
                base_url: '',
                api_key: '',
                tool_type: 'codex',
                model_name: '',
                extra_env: []
            };
            this.errors = {};
        },

        // 保存镜像
        async saveMirror() {
            this.saving = true;
            this.errors = {};

            // 验证表单
            if (!this.form.name) {
                this.errors.name = '请输入名称';
            }
            if (!this.form.base_url) {
                this.errors.base_url = '请输入 API 地址';
            } else {
                try {
                    await window.go.main.App.ValidateURL(this.form.base_url);
                } catch (error) {
                    this.errors.base_url = error || 'URL 格式无效';
                }
            }

            if (Object.keys(this.errors).length > 0) {
                this.saving = false;
                return;
            }

            // 处理 API Key 更新逻辑
            let apiKeyToUpdate = this.form.api_key;
            
            // 如果是编辑模式且API Key显示为掩码且用户没有修改，则使用原始API Key
            if (this.formMode === 'edit' && this.form.api_key === '***' && this.originalAPIKey) {
                apiKeyToUpdate = this.originalAPIKey; // 保持原来的API Key
            }
            // 如果用户清空了API Key字段，则传递空字符串
            else if (this.form.api_key === '') {
                apiKeyToUpdate = '';
            }
            // 如果用户输入了新的API Key，则使用新值

            // 将 extra_env 数组转换为对象
            const mirrorData = {
                ...this.form,
                api_key: apiKeyToUpdate, // 使用处理后的API Key
                extra_env: {}
            };
            
            // 过滤掉空 key 的扩展参数
            this.form.extra_env.forEach(env => {
                if (env.key && env.key.trim()) {
                    mirrorData.extra_env[env.key.trim()] = env.value || '';
                }
            });

            try {
                if (this.formMode === 'add') {
                    await window.go.main.App.AddMirror(mirrorData);
                    this.showToast('镜像源添加成功', 'success');
                } else {
                    await window.go.main.App.UpdateMirror(mirrorData);
                    this.showToast('镜像源更新成功', 'success');
                }

                await this.refreshMirrors();
                await this.refreshStatus();
                this.hideForm();
            } catch (error) {
                this.showToast(error || '保存失败', 'error');
            } finally {
                this.saving = false;
            }
        },

        // 切换镜像源
        async switchMirror(name) {
            try {
                await window.go.main.App.SwitchMirror(name);
                await this.refreshMirrors();
                await this.refreshStatus();
                this.showToast(`已切换到 ${name}`, 'success');
            } catch (error) {
                this.showToast(error || '切换失败', 'error');
            }
        },

        // 确认删除
        confirmDeleteMirror(mirror) {
            this.deleteTarget = mirror;
            this.showDeleteConfirm = true;
        },

        // 隐藏删除确认
        hideDeleteConfirm() {
            this.showDeleteConfirm = false;
            this.deleteTarget = null;
        },

        // 删除镜像
        async deleteMirror() {
            if (!this.deleteTarget) return;

            try {
                await window.go.main.App.RemoveMirror(this.deleteTarget.name);
                await this.refreshMirrors();
                await this.refreshStatus();
                this.showToast(`镜像源 ${this.deleteTarget.name} 已删除`, 'success');
            } catch (error) {
                this.showToast(error || '删除失败', 'error');
            } finally {
                this.hideDeleteConfirm();
            }
        },

        // 显示通知
        showToast(message, type = 'success') {
            const id = this.toastId++;
            this.toasts.push({ id, message, type });

            setTimeout(() => {
                this.toasts = this.toasts.filter(t => t.id !== id);
            }, 3000);
        },

        // 获取扩展参数的 tooltip 文本
        getExtraEnvTooltip(extraEnv) {
            if (!extraEnv || Object.keys(extraEnv).length === 0) {
                return '';
            }
            const lines = Object.entries(extraEnv).map(([key, value]) => {
                return `${key}=${value}`;
            });
            return '扩展参数:\n' + lines.join('\n');
        },

        // ============ 云同步相关方法 ============

        // 刷新同步状态
        async refreshSyncStatus() {
            try {
                const status = await window.go.main.App.GetSyncStatus();
                this.syncStatus = status;
            } catch (error) {
                console.error('获取同步状态失败:', error);
            }
        },

        // 显示同步设置弹窗
        showSyncSettings() {
            this.showSyncModal = true;
        },

        // 隐藏同步设置弹窗
        hideSyncModal() {
            this.showSyncModal = false;
            this.syncForm = { token: '', password: '', gistId: '', newPassword: '', newGistId: '' };
        },

        // 复制 Gist ID
        copyGistId() {
            if (!this.syncStatus.gistId) return;
            navigator.clipboard.writeText(this.syncStatus.gistId).then(() => {
                this.showToast('Gist ID 已复制', 'success');
            }).catch(() => {
                this.showToast('复制失败', 'error');
            });
        },

        // 更新同步设置
        async updateSyncSettings() {
            if (!this.syncForm.newPassword && !this.syncForm.newGistId) {
                this.showToast('请填写要更新的内容', 'error');
                return;
            }

            if (this.syncForm.newPassword && this.syncForm.newPassword.length < 8) {
                this.showToast('密码至少需要8位', 'error');
                return;
            }

            this.syncing = true;
            try {
                const result = await window.go.main.App.UpdateSyncSettings({
                    newPassword: this.syncForm.newPassword || '',
                    newGistId: this.syncForm.newGistId || ''
                });

                if (result.success) {
                    this.showToast(result.message, 'success');
                    await this.refreshSyncStatus();
                    this.syncForm.newPassword = '';
                    this.syncForm.newGistId = '';
                } else {
                    this.showToast(result.message, 'error');
                }
            } catch (error) {
                this.showToast(error || '更新失败', 'error');
            } finally {
                this.syncing = false;
            }
        },

        // 初始化云同步
        async initSync() {
            if (!this.syncForm.token || !this.syncForm.password) {
                this.showToast('请填写 GitHub Token 和加密密码', 'error');
                return;
            }

            if (this.syncForm.password.length < 8) {
                this.showToast('加密密码至少需要8位', 'error');
                return;
            }

            this.syncing = true;
            try {
                const result = await window.go.main.App.InitSync({
                    token: this.syncForm.token,
                    password: this.syncForm.password,
                    gistId: this.syncForm.gistId
                });

                if (result.success) {
                    this.showToast(result.message, 'success');
                    await this.refreshSyncStatus();
                    this.hideSyncModal();
                } else {
                    this.showToast(result.message, 'error');
                }
            } catch (error) {
                this.showToast(error || '初始化失败', 'error');
            } finally {
                this.syncing = false;
            }
        },

        // 推送配置到云端
        async syncPush() {
            this.syncing = true;
            try {
                const result = await window.go.main.App.SyncPush();
                this.showToast(result, 'success');
                await this.refreshSyncStatus();
                await this.refreshMirrors();
            } catch (error) {
                this.showToast(error || '推送失败', 'error');
            } finally {
                this.syncing = false;
            }
        },

        // 从云端拉取配置
        async syncPull() {
            this.syncing = true;
            try {
                const result = await window.go.main.App.SyncPull();
                this.showToast(result, 'success');
                await this.refreshSyncStatus();
                await this.refreshMirrors();
                await this.refreshStatus();
            } catch (error) {
                this.showToast(error || '拉取失败', 'error');
            } finally {
                this.syncing = false;
            }
        },

        // 禁用云同步
        async disableSync() {
            if (!confirm('确定要禁用云同步吗？')) {
                return;
            }

            this.syncing = true;
            try {
                await window.go.main.App.DisableSync();
                this.showToast('云同步已禁用', 'success');
                await this.refreshSyncStatus();
                this.hideSyncModal();
            } catch (error) {
                this.showToast(error || '禁用失败', 'error');
            } finally {
                this.syncing = false;
            }
        }
    };
}
