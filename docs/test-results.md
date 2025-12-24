# 镜像源测试结果报告

**测试时间**: 2025-12-24  
**测试命令**: `codex-mirror test --all`

---

## 📊 汇总统计

| 指标 | 数值 |
|------|------|
| 总镜像源数 | 26 |
| 测试成功 | 9 |
| 测试失败 | 17 |
| 成功率 | 34.6% |

---

## ✅ 测试成功的镜像源 (9个)

| 序号 | 名称 | 类型 | URL | 延迟 | 状态 |
|------|------|------|-----|------|------|
| 1 | cliproxy | claude | https://cliproxy.999gml.xyz | 3400ms | 正常 |
| 2 | packycode | codex | https://codex-api.packycode.com/v1 | 1977ms | 正常 |
| 3 | glm | claude | https://open.bigmodel.cn/api/anthropic | 849ms | 正常 |
| 4 | ikun-cc | claude | https://api.ikuncode.cc | 2481ms | 正常 |
| 5 | kimi-k2-gptloader | claude | https://gptloader.999gml.xyz/proxy/kimi-k2-ath | 4059ms | 正常 |
| 6 | kimi-k2-offical | claude | https://api.moonshot.cn/anthropic | 1265ms | 正常 |
| 7 | kat-coder | claude | https://vanchin.streamlake.ai/api/gateway/v1/endpoints/ep-jj1jkx-1760207080766611699/claude-code-proxy | 3718ms | 正常 |
| 8 | paid | codex | https://codex-api.packycode.com/v1 | 195ms | 正常 |
| 9 | privnode-cc | claude | https://privnode.com | 2805ms | 正常 |

---

## ❌ 测试失败的镜像源 (17个)

| 序号 | 名称 | 类型 | URL | 错误原因 | 建议 |
|------|------|------|-----|----------|------|
| 1 | min | claude | https://api.minimaxi.com/anthropic | 余额不足 | 充值 |
| 2 | kimi-k2-2 | claude | https://api.kimi.com/coding/ | API Key 无效 (401) | 更换 Key |
| 3 | privnode-codex | codex | https://privnode.com/v1 | URL 格式错误 (重复 /v1) | 检查配置 |
| 4 | kimi-k2-t | claude | https://ai.gitee.com/anthropic | 模型名称不支持 | 切换模型 |
| 5 | mt-qwen3-next-80b | claude | https://api-inference.modelsource.cn | 服务器错误 | 联系服务商 |
| 6 | kimi-k2 | claude | https://api.kimi.com/coding/ | API Key 无效 (401) | 更换 Key |
| 7 | gitee-k2t | claude | https://ai.gitee.com/anthropic | 模型名称不支持 | 切换模型 |
| 8 | gitee-k2 | claude | https://ai.gitee.com/anthropic | 模型名称不支持 | 切换模型 |
| 9 | free | codex | https://oai-api.fkclaude.com/v1 | 连接失败 | 检查网络/服务商 |
| 10 | claude | claude | https://api.anthropic.com | API Key 无效 (401) | 更换 Key |
| 11 | ds | claude | https://api.deepseek.com/anthropic | 余额不足 | 充值 |
| 12 | 88code-codex | codex | https://www.88code.org/openai/v1 | URL 格式错误 | 检查配置 |
| 13 | ikun-codex | codex | https://api.ikuncode.cc/v1 | URL 格式错误 (重复 /v1) | 检查配置 |
| 14 | official | codex | https://api.openai.com | 缺少 API Key | 添加 Key |
| 15 | nonocode | claude | https://claude.nonocode.cn/api | 连接超时 | 检查网络 |
| 16 | longcat | claude | https://api.longcat.chat/anthropic | API Key 无效 (401) | 更换 Key |
| 17 | glm-free | claude | https://newapi.ixio.cc | 缺少 API Key | 添加 Key |

---

## 📈 按类型统计

| 类型 | 总数 | 成功 | 失败 | 成功率 |
|------|------|------|------|--------|
| claude | 20 | 7 | 13 | 35.0% |
| codex | 6 | 2 | 4 | 33.3% |

---

## ⚠️ 常见问题

### 1. API Key 无效 (401)
- **影响**: 9 个镜像源
- **原因**: Key 过期、额度用尽或格式错误
- **建议**: 使用 `codex-mirror test --remove-invalid` 清理无效 Key

### 2. 模型名称不支持 (400)
- **影响**: 3 个镜像源 (gitee 系列)
- **原因**: 使用了不支持的模型版本 `claude-sonnet-4-20250514`
- **建议**: 切换到支持的模型或使用兼容模式

### 3. URL 格式错误
- **影响**: 3 个镜像源
- **原因**: URL 中包含重复的 `/v1` 路径
- **建议**: 检查并修正镜像源 URL 配置

### 4. 余额不足
- **影响**: 2 个镜像源 (min, ds)
- **原因**: API 调用额度耗尽
- **建议**: 充值或切换到其他镜像源

---

## 🛠️ 建议操作

### 立即执行
```bash
# 清理无效的 API Key
codex-mirror test --remove-invalid

# 切换到可用的镜像源
codex-mirror switch cliproxy
```

### 后续优化
1. 为不支持的镜像源配置兼容的模型名称
2. 修正 URL 格式错误的镜像源配置
3. 定期运行测试清理无效配置

---

## 📝 附录

### 测试命令说明
```bash
# 测试所有镜像源
codex-mirror test --all

# 并行测试 (速度更快)
codex-mirror test --all --parallel

# 测试并移除无效 Key
codex-mirror test --remove-invalid

# 设置超时时间 (秒)
codex-mirror test --all --timeout 30
```

### 镜像源健康度分布
```
✅ 正常 (9个)  ████████████████░░░░░░░░░  34.6%
⚠️  需关注 (17个) ████████████████████████████████████░░░░░░░  65.4%
```
