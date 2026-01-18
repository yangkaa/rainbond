# Rainbond 插件体系技术现状报告

> 生成时间: 2026-01-08
> 目的: 为插件体系重构提供技术现状基线

---

## 一、系统架构总览

### 1.1 请求链路

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Rainbond 插件体系架构                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐     ┌──────────────────┐     ┌──────────────────────┐    │
│  │  rainbond-ui │────▶│ rainbond-console │────▶│    rainbond (Go)     │    │
│  │   (React)    │     │    (Python)      │     │     rbd-api          │    │
│  └──────────────┘     └──────────────────┘     └──────────────────────┘    │
│         │                     │                         │                   │
│         │                     │                         ▼                   │
│         │                     │              ┌──────────────────────┐       │
│         │                     │              │   Kubernetes API     │       │
│         │                     │              │   RBDPlugin CRD      │       │
│         │                     │              └──────────────────────┘       │
│         │                     │                         │                   │
│         ▼                     ▼                         ▼                   │
│  ┌─────────────────────────────────────────────────────────────────┐       │
│  │                     企业插件 (JSInject)                          │       │
│  │  ┌────────────────────┐    ┌────────────────────┐               │       │
│  │  │ enterprise-base    │    │  rainbill          │               │       │
│  │  │ (备份/权限/设置)    │    │  (计量计费)         │               │       │
│  │  └────────────────────┘    └────────────────────┘               │       │
│  └─────────────────────────────────────────────────────────────────┘       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 代码库清单

| 代码库 | 路径 | 技术栈 | 职责 |
|--------|------|--------|------|
| rainbond-ui | `/Users/yangk/python3/rainbond-ui` | React + Ant Design | 前端渲染、插件加载 |
| rainbond-console | `/Users/yangk/python/rainbond-console-cloud-copy` | Python Django | API网关、业务逻辑 |
| rainbond | `/Users/yangk/go/src/github.com/goodrain/rainbond-backup-restore` | Go | 核心API、K8s交互 |
| enterprise-base | `/Users/yangk/go/src/goodrain.com/rainbond-enterprise-base` | React | 企业基础插件 |
| rainbill | `/Users/yangk/go/src/goodrain.com/rainbill` | Go + React | 计量计费插件 |

---

## 二、插件类型与机制

### 2.1 插件分类

| 类型 | 技术实现 | 视图范围 | 示例 |
|------|----------|----------|------|
| **平台插件 (RBDPlugin)** | JSInject / Iframe | Platform, Team, Application, Component | 企业基础、计量计费 |
| **团队插件 (TenantPlugin)** | 容器镜像 | Team级 | 网络治理、性能分析 |

### 2.2 RBDPlugin CRD 结构

```yaml
apiVersion: rainbond.io/v1alpha1
kind: RBDPlugin
metadata:
  name: rainbond-enterprise-base
  labels:
    app_id: 5cf0a95fa9dd4f6188d52d1147a434c9
    plugin.rainbond.io/enable: "true"
    plugin.rainbond.io/name: rainbond-enterprise-base
spec:
  display_name: 企业基础功能插件
  description: 包含应用备份、个性化设置等功能
  plugin_type: JSInject           # JSInject | Iframe
  plugin_views:                   # 支持的视图
    - Platform
  fronted_path: https://xxx/main.js  # 前端JS地址
  backend: ""                      # 后端服务地址(Iframe用)
  icon: https://xxx/logo.svg
  version: v1.0
  authors:
    - name: rainbond
      email: admin@demo.com
```

### 2.3 插件加载机制

**JSInject 类型** (前端动态加载):
```
1. 前端调用 GET /console/enterprise/{eid}/regions/{region}/plugins
2. 获取插件元数据列表
3. 通过 SystemJS.import(fronted_path) 加载插件JS
4. 实例化 RainbondEnterprisePagePlugin
5. 调用 plugin.init(meta) 初始化
6. 渲染插件React组件，注入baseInfo
```

**Iframe 类型** (嵌入式):
```
1. 获取插件元数据
2. 直接渲染 <iframe src={backend}> 加载后端服务
```

---

## 三、现有功能清单

### 3.1 企业基础插件 (rainbond-enterprise-base)

| 功能模块 | 状态 | 说明 |
|----------|------|------|
| 应用备份管理 | ✅ 完成 | 备份、恢复、迁移、导入导出 |
| 企业个性化设置 | ✅ 完成 | Logo、标题、登录页定制 |
| 权限管理 | ✅ 完成 | 角色创建、权限分配 |
| 登录日志 | ✅ 完成 | 用户登录记录查询 |
| 操作日志 | ✅ 完成 | 企业操作审计 |
| 语言包上传 | ✅ 完成 | 多语言构建包管理 |
| 资源超分配置 | ✅ 完成 | CPU/内存超分比例 |

### 3.2 计量计费插件 (rainbill)

| 功能模块 | 状态 | 说明 |
|----------|------|------|
| 资源采集 | ✅ 完成 | 从Prometheus采集CPU/内存/存储/网络 |
| 费用计算 | ✅ 完成 | 按资源使用量计费 |
| 账户管理 | ✅ 完成 | 余额、交易记录 |
| 充值支付 | ✅ 完成 | 微信支付集成 |
| 余额监控 | ✅ 完成 | 自动告警、停用 |
| 发票管理 | ✅ 完成 | 发票创建、下载 |

### 3.3 平台侧插件能力

| 功能 | 前端 | Console | Go后端 | 状态 |
|------|------|---------|--------|------|
| 插件列表展示 | ✅ | ✅ | ✅ | 完成 |
| 插件启用/禁用 | ✅ | ✅ | ✅ | 完成 |
| JSInject渲染 | ✅ | - | - | 完成 |
| Iframe渲染 | ✅ | - | - | 完成 |
| 插件后端代理 | - | ✅ | - | 完成 |
| 多视图支持 | ✅ | ✅ | ✅ | 完成 |

---

## 四、授权机制现状

### 4.1 License 验证流程

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        License 验证流程                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Console (Python)                    Go Backend (rbd-api)               │
│  ┌─────────────────┐                ┌─────────────────────────────┐    │
│  │ API Request     │                │ LicenseVerification()       │    │
│  │ (带Header)      │───────────────▶│                             │    │
│  │ enterprise_id   │                │ 1. 检查 rainbond-enterprise │    │
│  │ info_body       │                │    -base 插件是否存在       │    │
│  │ actual_cluster  │                │                             │    │
│  │ actual_node     │                │ 2. 若不存在 → 开源版,跳过   │    │
│  │ actual_memory   │                │                             │    │
│  └─────────────────┘                │ 3. 若存在 → 验证License     │    │
│                                     │    - 解析 info_body         │    │
│                                     │    - 检查过期时间            │    │
│                                     │    - 检查资源配额            │    │
│                                     │                             │    │
│                                     │ 4. 返回验证结果             │    │
│                                     └─────────────────────────────┘    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 4.2 License 数据结构

```go
// Go后端 (util/license/license.go)
type LicInfo struct {
    LicKey     string   `json:"license_key"`
    Code       string   `json:"code"`       // 企业代码
    Company    string   `json:"company"`    // 企业名称
    Node       int64    `json:"node"`       // 授权节点数
    CPU        int64    `json:"cpu"`        // 授权CPU
    Memory     int64    `json:"memory"`     // 授权内存
    Tenant     int64    `json:"tenant"`     // 授权租户数
    EndTime    string   `json:"end_time"`   // 到期时间
    StartTime  string   `json:"start_time"` // 开始时间
    DataCenter int64    `json:"data_center"`// 数据中心数
    ModuleList []string `json:"module_list"`// 模块列表
}
```

### 4.3 企业版判断逻辑

```go
// 关键代码 (api/middleware/middleware.go:442)
func LicenseVerification(r *http.Request, resourceValidation bool) *util.APIHandleError {
    // 检查 rainbond-enterprise-base 插件是否存在
    _, err := k8s.Default().RainbondClient.RainbondV1alpha1().
        RBDPlugins(metav1.NamespaceNone).
        Get(context.TODO(), "rainbond-enterprise-base", metav1.GetOptions{})
    if err != nil {
        // 插件不存在 = 开源版，跳过License验证
        return nil
    }
    // 插件存在 = 企业版，进行License验证
    // ...
}
```

### 4.4 现有授权机制问题

| 问题 | 描述 | 影响 |
|------|------|------|
| **无AccessKey机制** | 没有统一的API访问密钥 | 无法细粒度控制API访问 |
| **无插件级授权** | License只控制整体，不控制单个插件 | 无法按插件收费 |
| **无授权分级** | 没有basic/pro/enterprise等级别 | 商业化灵活性差 |
| **无依赖检查** | 插件间无依赖关系定义 | 无法强制基础插件优先 |
| **过期行为不明** | License过期后系统行为不清晰 | 用户体验差 |
| **离线授权不完善** | 缺少离线场景的授权验证机制 | 离线部署受限 |

---

## 五、安装与分发现状

### 5.1 当前安装方式

| 方式 | 流程 | 问题 |
|------|------|------|
| **应用市场安装** | 从云市拉取 → 部署应用 → 创建RBDPlugin CRD | 需要联网 |
| **手动安装** | 手动部署 → 手动创建CRD | 复杂易错 |
| **离线导入** | 导入镜像 → 部署 → 创建CRD | 流程不完整 |

### 5.2 镜像仓库

- 当前使用阿里云OSS存放插件前端JS
- 无统一的插件镜像仓库
- 无账号密码授权机制

### 5.3 缺失能力

| 缺失项 | 说明 |
|--------|------|
| 插件市场UI | 无独立的插件市场浏览页面 |
| 一键安装 | 无自动化安装流程 |
| 版本管理 | 无插件版本升级机制 |
| 离线包规范 | 无标准的离线安装包格式 |
| 依赖管理 | 无插件依赖自动安装 |

---

## 六、差距分析（对照需求）

### 6.1 需求对照表

| 需求 | 现状 | 差距 | 优先级 |
|------------------|------|------|--------|
| 插件支持在线安装 | 通过应用市场，流程复杂 | 需简化流程 | 高 |
| AccessKey与企业授权同步 | AccessKey不存在 | 需从零开发 | 高 |
| 基础插件→企业插件依赖 | 无依赖机制 | 需扩展CRD | 中 |
| 离线安装包规范 | 无规范 | 需设计 | 中 |
| 授权分级（basic/pro/enterprise） | 无分级 | 需设计 | 中 |
| 插件列表写死（版本预置） | 无此能力 | 需开发 | 中 |
| 用户自主试用授权 | 无此能力 | 需开发 | 低 |
| 镜像仓库账号密码授权 | 无此能力 | 需设计 | 低 |

### 6.2 技术债务

| 债务项 | 描述 | 建议处理方式 |
|--------|------|--------------|
| License验证逻辑分散 | Go/Python两端都有验证 | 统一到Go端 |
| 插件元数据硬编码 | 部分插件信息硬编码在前端 | 改为动态配置 |
| 缺乏插件规范文档 | 开发新插件无标准参考 | 输出规范文档 |
| 无插件健康检查 | 不知道插件是否正常工作 | 增加健康检查 |

---

## 七、关键代码位置索引

### 7.1 前端 (rainbond-ui)

| 功能 | 文件路径 |
|------|----------|
| 插件列表页 | `src/pages/RbdPlugins/index.js` |
| 插件渲染组件 | `src/components/RBDPluginsCom/index.js` |
| SystemJS加载 | `src/utils/importPlugins.js` |
| 插件API调用 | `src/services/api.js` |
| 扩展页面 | `src/pages/Extension/index.js` |
| 路由配置 | `config/router.config.js` |

### 7.2 Python后端 (rainbond-console)

| 功能 | 文件路径 |
|------|----------|
| 插件列表API | `console/views/rbd_plugin.py` |
| 插件代理API | `console/views/rbd_plugin.py` (RainbondPluginBackendView) |
| 团队插件服务 | `console/services/plugin/app_plugin.py` |
| 市场插件服务 | `console/services/market_plugin_service.py` |
| 插件数据模型 | `www/models/plugin.py` |

### 7.3 Go后端 (rainbond)

| 功能 | 文件路径 |
|------|----------|
| RBDPlugin CRD | `pkg/apis/rainbond/v1alpha1/rbdplugin_types.go` |
| 插件Handler | `api/handler/cluster.go` |
| License验证 | `api/middleware/middleware.go` |
| License工具 | `util/license/license.go` |

### 7.4 企业基础插件

| 功能 | 文件路径 |
|------|----------|
| 插件导出 | `src/moudle.js` |
| 主容器 | `src/page/index.js` |
| 备份功能 | `src/page/AppBackUpPage/` |
| Webpack配置 | `build/webpack.prod.config.js` |

---

## 八、后续行动建议

### 8.1 建议的里程碑

```
M1: 可用（商业化支撑）
├── 修复现有插件安装问题
├── 完善插件列表展示
├── 编写插件使用文档
└── 预期：现有功能稳定可用

M2: 可控（规则无漏洞）
├── 设计AccessKey机制
├── 实现授权分级
├── 明确License过期行为
├── 实现插件依赖检查
└── 预期：授权体系完善

M3: 可扩展（生态建设）
├── 插件开发规范
├── 离线安装包规范
├── 插件市场UI
├── 版本升级机制
└── 预期：第三方可开发插件
```

### 8.2 技术选型建议

| 领域 | 建议方案 |
|------|----------|
| AccessKey存储 | K8s Secret + Console数据库双存 |
| 授权分级 | 扩展License结构，增加tier字段 |
| 插件依赖 | 扩展RBDPlugin CRD，增加dependencies字段 |
| 离线包格式 | 参考Helm Chart + OCI Artifact |
| 插件规范 | 输出Markdown文档 + 示例项目 |

---

## 九、附录

### A. RBDPlugin CRD 字段建议扩展

```yaml
spec:
  # 现有字段...

  # 建议新增字段
  tier: enterprise              # basic | pro | enterprise
  requires_license: true        # 是否需要License
  dependencies:                 # 依赖的其他插件
    - name: rainbond-enterprise-base
      version: ">=1.0.0"
  offline_package:              # 离线包信息
    images:
      - registry.example.com/plugin:v1.0
    charts:
      - name: plugin-chart
        version: 1.0.0
```

### B. AccessKey 设计草案

```go
type AccessKey struct {
    ID           string    `json:"id"`
    EnterpriseID string    `json:"enterprise_id"`
    Key          string    `json:"key"`           // 访问密钥
    Secret       string    `json:"secret"`        // 密钥
    ExpireAt     time.Time `json:"expire_at"`     // 过期时间
    Tier         string    `json:"tier"`          // 授权级别
    AllowedPlugins []string `json:"allowed_plugins"` // 允许的插件
    CreatedAt    time.Time `json:"created_at"`
}
```

---

*报告结束*
第一阶段再设计一下。

1. 扩展插件页面需要实打实能展示插件和安装，如果是开源插件免费装，企业插件的话就根据我们给的授权来。
2. 跟插件对应的在文档系统有文档。
3. 要能给用户授权，要解决授权的问题，要求插件从下载（下载需要授权）和复制（复制了也不能在另一个环境直接用）得到控制，根据授权能限定两个版本，基础版和高级版（控制安装哪些插件）
4. 授权好之后，用户在列表中可以自助安装和升级
5. 授权本身需要优化一些提醒机制
6. 打包机制需要是完整打包机制，可以独立安装和独立升级。