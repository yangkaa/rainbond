# Rainbond 插件体系问题清单

> 版本: v1.0
> 更新时间: 2026-01-08
> 关联任务: M1-1

---

## 概述

本文档是 M1（商业可用）里程碑的首要交付物，记录当前插件体系中发现的所有问题。问题按严重程度和所属组件分类，每个问题都包含具体文件路径和修复建议。

### 问题统计

| 严重等级 | 数量 | 说明 |
|----------|------|------|
| P0 (Critical) | 5 | 必须立即修复，影响安全或核心功能 |
| P1 (High) | 12 | 高优先级，影响用户体验或稳定性 |
| P2 (Medium) | 15 | 中等优先级，影响可维护性或性能 |
| P3 (Low) | 8 | 低优先级，代码质量问题 |

---

## 一、Critical (P0) 问题

### P0-1: SQL 注入漏洞

**组件**: rainbond-console (Python)
**文件**: `/www/services/plugin.py:67-72`

```python
# 问题代码
query_sql = """SELECT * from plugin_build_version WHERE
  id in (SELECT max(id) from plugin_build_version WHERE
  tenant_id="{0}" and region="{1}" GROUP BY plugin_id)
""".format(tenant.tenant_id, region, tenant.tenant_id)
```

**风险**: 攻击者可注入任意 SQL，窃取或修改插件数据

**修复方案**:
```python
# 使用参数化查询
from django.db import connection
with connection.cursor() as cursor:
    cursor.execute("""
        SELECT * from plugin_build_version WHERE
        id in (SELECT max(id) from plugin_build_version WHERE
        tenant_id=%s and region=%s GROUP BY plugin_id)
    """, [tenant.tenant_id, region])
```

---

### P0-2: 插件状态判断 Bug（字符串比较）

**组件**: rainbond-ui (React)
**文件**: `/src/utils/pulginUtils.js:12`

```javascript
// 问题代码：enable_status 是布尔值，但用字符串比较
&& item.enable_status === 'true'  // 永远不匹配！
```

**影响**: 所有已启用的插件都会被过滤掉，插件功能完全不可用

**修复方案**:
```javascript
// 使用布尔值比较
&& item.enable_status === true
// 或兼容两种情况
&& (item.enable_status === true || item.enable_status === 'true')
```

**涉及文件**:
- `/src/utils/pulginUtils.js:12`
- `/src/pages/Extension/pluginCapacity/pluginTable.js:288, 402`

---

### P0-3: 插件加载缓存机制失效

**组件**: rainbond-ui (React)
**文件**: `/src/utils/importPlugins.js:64-73`

```javascript
// 问题代码：计算了 bust 值但没有使用
export function locateWithCache(load, defaultBust = initializedAt) {
  const { address } = load;
  const path = extractPath(address);
  const version = cache[path];
  const bust = version || defaultBust;
  return `${address}`;  // BUG: 应该是 `${address}?v=${bust}`
}
```

**影响**: 插件版本更新后，用户浏览器继续使用旧版本缓存

**修复方案**:
```javascript
return `${address}?v=${bust}`;
```

---

### P0-4: 插件加载无错误处理

**组件**: rainbond-ui (React)
**文件**: `/src/utils/importPlugins.js:47-62`

```javascript
// 问题代码：无 catch 块，错误会导致页面崩溃
export async function importAppPagePlugin(meta, regionName, type) {
  const xu = await importPluginModule(meta, regionName).then(function (pluginExports) {
    // ...
  });
  // 缺少 .catch() 处理
  return xu
}
```

**影响**: 插件加载失败时整个页面崩溃，无法恢复

**修复方案**:
```javascript
export async function importAppPagePlugin(meta, regionName, type) {
  try {
    const module = await importPluginModule(meta, regionName);
    const plugin = module.plugin || (type === 'enterprise'
      ? new RainbondEnterprisePagePlugin()
      : new RainbondRootPagePlugin());
    plugin.init(meta);
    plugin.meta = meta;
    return plugin;
  } catch (error) {
    console.error(`Failed to load plugin: ${meta.name}`, error);
    throw new Error(`插件 ${meta.display_name || meta.name} 加载失败`);
  }
}
```

---

### P0-5: 插件路径属性无验证

**组件**: rainbond-ui (React)
**文件**: `/src/utils/importPlugins.js:49`

```javascript
// 问题代码：直接使用 meta.fronted_path，无空值检查
const path = meta.fronted_path
const module = await SystemJS.import(path);
```

**影响**: 如果 `fronted_path` 为空，SystemJS 尝试加载 undefined URL，导致静默失败

**修复方案**:
```javascript
const path = meta?.fronted_path;
if (!path) {
  throw new Error(`Plugin ${meta?.name || 'unknown'} missing fronted_path`);
}
const module = await SystemJS.import(path);
```

---

## 二、High (P1) 问题

### P1-1: K8s 清单中硬编码凭据

**组件**: rainbill
**文件**: `/rainbond-enterprise-billing-server/deploy/billing-server.yaml:28-65`

```yaml
# 问题：敏感信息直接写在 YAML 中
- name: WECHAT_MCHID
  value: "1900009191"
- name: MYSQL_HOST
  value: "10.0.0.2"
```

**风险**:
- 凭据泄露到版本控制
- 不同环境需要手动修改 YAML

**修复方案**:
- 使用 Kubernetes Secret 存储敏感信息
- 使用 ConfigMap 存储配置
- 考虑引入 Helm Chart 参数化

---

### P1-2: 插件依赖无检查机制

**组件**: rainbond (Go)
**文件**: `/pkg/apis/rainbond/v1alpha1/rbdplugin_types.go`

**问题**: RBDPlugin CRD 没有 `dependencies` 字段，无法表达插件依赖关系

**影响**: 无法强制 rainbill 依赖 enterprise-base，用户可能安装顺序错误

**修复方案**: 扩展 CRD
```yaml
spec:
  dependencies:
    - name: rainbond-enterprise-base
      version: ">=1.0.0"
      required: true
```

---

### P1-3: 插件启用/禁用 API 错误处理不当

**组件**: rainbond-ui (React)
**文件**: `/src/pages/Extension/pluginCapacity/pluginTable.js:281-301`

```javascript
// 问题：成功和失败都执行相同的 handlePluginList()
callback: res => {
    this.handlePluginList()
    notification.success({ message: '...' })
},
handleError: err => {
    this.handlePluginList()  // 失败也刷新？
    notification.error({ message: '...' })
}
```

**影响**:
- 用户无法区分操作是否真正成功
- 失败后 UI 状态可能不一致

**修复方案**:
```javascript
callback: res => {
    this.handlePluginList()
    notification.success({ message: '操作成功' })
},
handleError: err => {
    // 不刷新，保持当前状态，让用户重试
    notification.error({ message: `操作失败: ${err.message}` })
}
```

---

### P1-4: Console API 响应未验证

**组件**: rainbond-console (Python)
**文件**: `/www/services/plugin.py:895, 916`

```python
# 问题：直接访问字典键，无存在性检查
return body["list"]  # 如果 body 没有 "list" 键？
status = body["bean"]["status"]  # 嵌套访问更危险
```

**影响**: API 响应结构变化时服务崩溃

**修复方案**:
```python
return body.get("list", [])
status = body.get("bean", {}).get("status", "unknown")
```

---

### P1-5: 异常被静默吞掉

**组件**: rainbond-console (Python)
**文件**: `/www/tenantservice/baseservice.py:41`

```python
try:
    pbv = PluginBuildVersion.objects.get(...)
    memory += pbv.min_memory
except Exception:
    pass  # 所有异常都被忽略！
```

**影响**:
- 内存计算可能返回错误值
- 数据库错误无法被发现
- 调试困难

**修复方案**:
```python
try:
    pbv = PluginBuildVersion.objects.get(...)
    memory += pbv.min_memory
except PluginBuildVersion.DoesNotExist:
    logger.warning(f"Plugin build version not found: {tpr.plugin_id}")
except Exception as e:
    logger.error(f"Error getting plugin resource: {e}")
```

---

### P1-6: 数据库查询无异常处理

**组件**: rainbond-console (Python)
**文件**: `/www/services/plugin.py:76, 187, 199`

```python
# 可能抛出 DoesNotExist 异常
plugin = TenantPlugin.objects.get(plugin_id=d.plugin_id)
oldRelation = TenantServicePluginRelation.objects.get(...)
```

**影响**: 记录不存在时返回 HTTP 500 错误

**修复方案**:
```python
try:
    plugin = TenantPlugin.objects.get(plugin_id=d.plugin_id)
except TenantPlugin.DoesNotExist:
    return Response({"error": "Plugin not found"}, status=404)
```

---

### P1-7: 插件安装无一键流程

**组件**: 全局
**现状**:
1. 从应用市场找到插件
2. 安装应用到指定团队
3. 等待应用启动
4. 手动创建 RBDPlugin CRD
5. 配置插件参数
6. 刷新页面

**影响**: 新用户难以完成安装，流失率高

**修复方案**:
- 设计一键安装 API
- 前端插件市场 UI
- 自动创建 CRD

---

### P1-8: 离线安装无标准流程

**组件**: 全局
**现状**: 无标准离线包格式，无离线安装脚本

**影响**: 离线客户安装困难

**修复方案**:
- 定义离线包规范 (见 roadmap M3)
- 提供离线包制作脚本
- 提供离线安装脚本

---

### P1-9: Rainbill 配置复杂度过高

**组件**: rainbill
**文件**: `/rainbond-enterprise-billing-server/internal/config/config.go`

**问题**: 需要配置 20+ 环境变量
- MySQL 连接 (5个)
- 微信支付 (6个)
- 短信服务 (4个)
- 邮件服务 (5个)

**影响**: 安装配置繁琐，易出错

**修复方案**:
- 提供配置模板
- 减少必填配置
- 提供配置向导

---

### P1-10: Enterprise-Base 无安装文档

**组件**: enterprise-base
**文件**: `/README.md` (空文件)

**影响**: 开发者不知道如何安装和配置

**修复方案**: 编写完整的安装文档

---

### P1-11: License 过期行为未定义

**组件**: rainbond (Go)
**文件**: `/api/middleware/middleware.go`

**问题**: License 过期后系统行为不明确
- 是提示还是强制？
- 已运行应用是否停止？
- 宽限期多长？

**影响**: 用户体验不一致，商业风险

**修复方案**: 定义清晰的过期状态机 (见 roadmap M2)

---

### P1-12: 松散相等比较 (== vs ===)

**组件**: rainbond-ui, enterprise-base
**文件**: 多处

| 文件 | 行号 | 问题代码 |
|------|------|----------|
| `/src/pages/Extension/pluginCapacity/pluginTable.js` | 288 | `val.enable_status == 'true'` |
| `/enterprise-base/src/utils/logSocket.js` | 58 | `evt.data != 'ok'` |
| `/enterprise-base/src/moudle/appBackUpPage.js` | 35 | `currentLocale == 'zh'` |

**影响**: 类型强转可能导致意外行为

**修复方案**: 全局替换为严格相等 (`===`, `!==`)

---

## 三、Medium (P2) 问题

### P2-1: 硬编码插件黑名单

**组件**: rainbond-ui
**文件**: `/src/utils/pulginUtils.js:7-10`

```javascript
&& item.name != 'rainbond-enterprise-base'
&& item.name != 'rainbond-bill'
&& item.name !='rainbond-observability'  // 还有空格问题
```

**问题**: 添加新插件需要改代码

**修复方案**: 移到配置文件或后端 API 返回

---

### P2-2: Debug console.log 残留

**组件**: rainbond-ui, enterprise-base
**文件**:

| 位置 | 行号 |
|------|------|
| `/src/pages/Extension/pluginCapacity/pluginTable.js` | 421 |
| `/enterprise-base/src/utils/logSocket.js` | 23, 52 |
| `/enterprise-base/src/page/PackageUploadPage/index.js` | 37 |
| `/enterprise-base/src/utils/request.js` | 40 |

**修复方案**: 删除或替换为 logger

---

### P2-3: 全局 window 对象污染

**组件**: enterprise-base
**文件**: `/src/components/WatchMore/index.js:22-25`

```javascript
window.diff_match_patch = DiffMatchPatch;
window.DIFF_DELETE = -1;
```

**问题**: 可能与其他代码冲突

**修复方案**: 使用模块导入代替全局挂载

---

### P2-4: Axios 基础 URL 为空

**组件**: enterprise-base
**文件**: `/src/utils/request.js:5`

```javascript
const http = axios.create({
  baseURL: '',  // 空！
  timeout: 10000,
});
```

**问题**: 完全依赖调用方传入完整 URL

**修复方案**: 从环境变量或 baseInfo 配置

---

### P2-5: 硬编码区域特殊逻辑

**组件**: rainbond-console
**文件**: `/www/services/plugin.py:689`

```python
if region == "ali-hz":
    min_cpu = min_cpu * 2
```

**问题**: 区域特殊处理硬编码在代码中

**修复方案**: 配置化

---

### P2-6: 敏感信息日志输出

**组件**: rainbond-console
**文件**: `/www/services/plugin.py:847, 621`

```python
logger.debug("=====> build_data {0}".format(build_data))
```

**问题**: 可能包含敏感配置信息

**修复方案**: 脱敏处理后再记录

---

### P2-7: 数据库更新非原子操作

**组件**: rainbond-console
**文件**: `/www/services/plugin.py:186-192`

```python
oldRelation = TenantServicePluginRelation.objects.get(...)
oldRelation.build_version = build_version
oldRelation.save()
TenantServicePluginAttr.objects.filter(...).delete()
# 如果 delete 失败，relation 已经改了
```

**修复方案**: 使用 `@transaction.atomic`

---

### P2-8: 插件列表无分页

**组件**: rainbond-console
**文件**: `/www/services/plugin.py:88-96`

```python
def get_tenant_plugins(self, region, tenant):
    plugins = TenantPlugin.objects.filter(...)
    return plugins  # 返回所有，无分页
```

**修复方案**: 添加分页参数

---

### P2-9: 注释中的硬编码 URL

**组件**: rainbond-console
**文件**: `/www/apiclient/marketclient.py:63, 72`

```python
# url = "http://5000.grcd3008.goodrain.ali-hz.goodrain.net:10080"
```

**问题**: 暴露内部基础设施信息

**修复方案**: 删除注释代码

---

### P2-10: 插件类型无验证

**组件**: rainbond-ui
**文件**: `/src/components/RBDPluginsCom/index.js:118-124`

```javascript
plugins?.plugin_type === 'JSInject' ? this.rbdPluginsRender() : this.iframeRender()
```

**问题**: 未知类型默认 iframe 渲染

**修复方案**: 添加类型白名单验证

---

### P2-11: URL 正则匹配不完整

**组件**: rainbond-ui
**文件**: `/src/utils/importPlugins.js:75-85`

```javascript
const match = /\/public\/(plugins\/.+\/module)\.js/i.exec(address);
```

**问题**: 只匹配 `module.js`，其他文件名会失败

**修复方案**: 扩展正则或使用更灵活的方式

---

### P2-12: 输入验证不完整

**组件**: rainbond-console
**文件**: `/console/views/plugin/service_plugin.py:43-46`

**问题**: 只验证了 category 参数，其他参数无验证

**修复方案**: 使用 DRF Serializer 全面验证

---

### P2-13: Rainbill TODO 未完成

**组件**: rainbill
**文件**: `/internal/service/account.go`

- Line 144: `// TODO: 关闭所有应用`
- Line 192: `// TODO：清理所有资源`
- `/internal/service/cost.go:27`: `// TODO: 实现费用查询逻辑`

---

### P2-14: 超时配置固定

**组件**: enterprise-base
**文件**: `/src/utils/request.js:6`

```javascript
timeout: 10000,  // 固定 10 秒，大文件上传可能超时
```

---

### P2-15: WebSocket 协议转换简单替换

**组件**: enterprise-base
**文件**: `/src/page/PackageUploadPage/index.js:192-200`

```javascript
replaceProtocol = (address) => {
  if (address.startsWith('ws:')) {
    return address.replace('ws:', 'http:');  // 简单替换
  }
}
```

---

## 四、Low (P3) 问题

### P3-1: PropTypes 验证关闭

**组件**: enterprise-base
**文件**: `/.eslintrc.js:19`

```javascript
'react/prop-types': 'off',
```

---

### P3-2: 方法实现不完整

**组件**: rainbond-console
**文件**: `/www/services/plugin.py:456-461`

```python
def metaTypeAttrs(self, tenant_id, service_id, attrList, metaType):
    if metaType == ConstKey.DOWNSTREAM_PORT:
        # 只处理了一种情况
```

---

### P3-3: 轮询间隔硬编码

**组件**: enterprise-base
**文件**: `/src/page/AppBackUpPage/index.js:82`

```javascript
this.timer = setTimeout(() => {
  this.startLoopStatus();
}, 10000);  // 固定 10 秒
```

---

### P3-4: 错误信息静默

**组件**: enterprise-base
**文件**: `/src/utils/logSocket.js:20-24`

```javascript
try {
  this.init();
} catch (e) {
  console.log(e);  // 只打印，不处理
}
```

---

### P3-5: 无安全页面刷新确认

**组件**: enterprise-base
**文件**: `/src/components/MigrationBackup/index.js`

使用 `window.location.reload()` 无用户确认

---

### P3-6: 单副本部署无 HA

**组件**: rainbill
**文件**: `/deploy/billing-server.yaml`

```yaml
replicas: 1  # 无高可用
```

---

### P3-7: 实例变量代替 state

**组件**: rainbond-ui
**文件**: `/src/pages/RbdPlugins/index.js`

```javascript
this.importingPlugin = meta.name;  // 应该用 setState
```

---

### P3-8: 授权 token 代码注释

**组件**: enterprise-base
**文件**: `/src/utils/request.js:13-15`

```javascript
// config.headers['Authorization'] = '你的token';
```

---

## 五、修复优先级建议

### 第一批（本周）- 核心安全和功能

| ID | 问题 | 负责模块 | 预估工时 |
|----|------|----------|----------|
| P0-1 | SQL 注入 | console | 2h |
| P0-2 | 状态判断 Bug | ui | 1h |
| P0-3 | 缓存机制失效 | ui | 1h |
| P0-4 | 加载无错误处理 | ui | 2h |
| P0-5 | 路径属性验证 | ui | 1h |

### 第二批（下周）- 高优先级

| ID | 问题 | 负责模块 | 预估工时 |
|----|------|----------|----------|
| P1-3 | 启用/禁用错误处理 | ui | 2h |
| P1-4 | API 响应验证 | console | 2h |
| P1-5 | 异常静默吞掉 | console | 1h |
| P1-6 | 查询异常处理 | console | 2h |
| P1-12 | 松散相等比较 | ui, enterprise-base | 2h |

### 第三批（M1 期间）- 中等优先级

- P1-1: K8s 凭据改为 Secret
- P1-10: 编写安装文档
- P2-1 ~ P2-15: 中等优先级问题

### 第四批（M2 期间）- 架构改进

- P1-2: 插件依赖机制
- P1-7: 一键安装流程
- P1-8: 离线安装规范
- P1-11: License 过期行为

---

## 六、相关文档

- [插件体系技术现状报告](./plugin-system-status-report.md)
- [里程碑规划](./plugin-system-roadmap.md)

---

## 七、变更记录

| 日期 | 版本 | 内容 | 作者 |
|------|------|------|------|
| 2026-01-08 | v1.0 | 初始版本 | - |

---

*文档结束*
