# 修复 AppService 初始化并发竞态条件导致 Pod 信息丢失

## 问题描述

在生产环境中发现，GetAppPods 接口偶尔返回空的 Pod 列表（0 个 Pods），即使日志显示 Pod 已经成功添加到 AppService。这个问题在服务重启或大量服务同时启动时更容易出现。

### 问题表现

```
时间轴：
03:44:02 - [Pod新增] 成功添加到 AppService (当前Pod数量=1)     ✅
03:44:02 - [Deployment新增] 成功设置到 AppService (CurrentPodCount=0) ⚠️
03:45:02 - GetAppPods 返回 0 个 pods                         ❌
```

**关键症状：**
- Pod 添加日志显示成功（Pod数量=1）
- 但随后的状态显示 Pod 数量为 0
- GetAppPods 持续返回空列表，直到下一次 resync（30分钟后）

---

## 根本原因

### 1. 并发竞态条件

在 `worker/appm/store/store.go` 的 `getAppService` 函数中存在经典的 **Check-Then-Act** 竞态条件：

```go
// 旧代码（有问题）
func (a *appRuntimeStore) getAppService(serviceID, version, createrID string, creator bool) (*v1.AppService, error) {
    appservice = a.GetAppService(serviceID)  // ← 检查

    if appservice == nil && creator {
        // ← 多个协程可能同时到达这里！
        appservice, err = conversion.InitCacheAppService(...)  // ← 创建新实例
        a.RegistAppService(appservice)  // ← 注册（后注册的会覆盖先注册的）
    }
    return appservice, nil
}
```

### 2. 问题触发流程

当 Kubernetes Informer 同时触发 Pod Add 和 Deployment Add 事件时：

```
┌─────────────────────────────────────────────────────────────────┐
│ Deployment Add 协程          Pod Add 协程                        │
├─────────────────────────────────────────────────────────────────┤
│ ① 检查内存 → nil                                                │
│ ② 创建 AppService A                                             │
│                              ③ 检查内存 → nil (几乎同时)         │
│                              ④ 创建 AppService B                │
│ ⑤ 注册 A 到内存                                                 │
│                              ⑥ 注册 B 到内存 (覆盖 A) ❌        │
│ ⑦ A.SetPods(pod)                                                │
│    → A.pods = [pod1]                                            │
│    → 但 A 已经被 B 覆盖了！                                     │
│                                                                  │
│ 内存中实际保存的是 B，而 B.pods = [] (空)                       │
│ GetAppPods 返回 B.pods → 空列表 ❌                              │
└─────────────────────────────────────────────────────────────────┘
```

### 3. 为什么之前没有发现？

**关键提交：** `773bb6883` (2025-09-18)

```diff
- infFactory := informers.NewSharedInformerFactoryWithOptions(..., 10*time.Second)
+ infFactory := informers.NewSharedInformerFactoryWithOptions(..., 30*time.Minute)

- store.informers.Pod.AddEventHandlerWithResyncPeriod(..., time.Second*10)
+ store.informers.Pod.AddEventHandlerWithResyncPeriod(..., 0)
```

**之前（10秒 resync）的"自愈"机制：**
- 竞态条件产生 → Pod 信息丢失
- **10 秒后** resync 触发 Update 事件
- Update 事件重新同步所有资源到同一个 AppService
- **问题自动修复**，用户几乎感知不到

**现在（30分钟 resync）的问题暴露：**
- 竞态条件产生 → Pod 信息丢失
- **需要等 30 分钟** 才会 resync 修复
- 在这 30 分钟内，GetAppPods 持续返回错误结果

**结论：** 30 分钟 resync 是正确的性能优化（减少 API Server 压力 180 倍），但暴露了代码中隐藏的并发 bug。

---

## 影响范围

### 触发条件

1. **服务重启时**：所有 Informer 同时接收大量 Add 事件
2. **大规模集群**：服务越多，并发事件越多，触发概率越高
3. **新服务部署**：同一服务的多个资源（Pod、Deployment、Service）同时创建

### 影响程度

- **轻微场景**：小集群（<100 服务），偶尔出现，30 分钟后自动恢复
- **严重场景**：大集群（>500 服务），频繁出现，影响监控和运维

---

## 解决方案

### 方案选择

| 方案 | 优点 | 缺点 | 是否采用 |
|------|------|------|---------|
| 回滚到 10 秒 resync | 简单 | 性能倒退 180 倍 | ❌ |
| 使用全局锁 | 实现简单 | 所有 serviceID 串行，性能差 | ❌ |
| 使用 LoadOrStore | 无锁，高性能 | 对象替换导致数据丢失 | ❌ |
| **双重检查锁（DCL）** | 并发安全，性能高 | 需要管理锁 Map | ✅ |

### 采用方案：双重检查锁模式

**核心思想：**
1. 第一次检查（无锁）：快速路径，已存在则直接返回
2. 获取锁：每个 serviceID 一个独立的锁
3. 第二次检查（持有锁）：避免重复初始化
4. 初始化：只有一个协程能执行
5. 释放锁：其他协程获取已初始化的对象

### 实现代码

```go
// 添加字段
type appRuntimeStore struct {
    // ... 其他字段
    initLocks sync.Map // map[serviceID]*sync.Mutex for AppService initialization
}

// getAppService - 修复后
func (a *appRuntimeStore) getAppService(serviceID, version, createrID string, creator bool) (*v1.AppService, error) {
    key := v1.GetCacheKeyOnlyServiceID(serviceID)

    // 第一次检查（无锁，快速路径）
    if app, ok := a.appServices.Load(key); ok {
        return app.(*v1.AppService), nil
    }

    if !creator {
        return nil, nil
    }

    // 获取该 serviceID 的专属锁
    // 使用 LoadOrStore 确保每个 serviceID 只有一个锁实例
    lockInterface, _ := a.initLocks.LoadOrStore(serviceID, &sync.Mutex{})
    mu := lockInterface.(*sync.Mutex)

    // 加锁，确保同一 serviceID 的初始化串行执行
    mu.Lock()
    defer mu.Unlock()

    // 第二次检查（持有锁）
    // 可能在等待锁期间，其他协程已经完成了初始化
    if app, ok := a.appServices.Load(key); ok {
        logrus.Debugf("[getAppService] 并发场景：其他协程已完成初始化 AppService (serviceID=%s)", serviceID)
        return app.(*v1.AppService), nil
    }

    // 当前协程获得初始化权，开始真正的初始化
    logrus.Infof("[getAppService] 内存中未找到 AppService (serviceID=%s)，开始从数据库初始化", serviceID)

    appservice, err := conversion.InitCacheAppService(a.dbmanager, serviceID, createrID)
    if err != nil {
        logrus.Warnf("[getAppService] 从数据库初始化 AppService 失败 (serviceID=%s, createrID=%s): %s",
            serviceID, createrID, err.Error())
        return nil, err
    }

    // 存储到缓存
    a.appServices.Store(key, appservice)
    atomic.AddInt32(&a.appCount, 1)

    logrus.Infof("[getAppService] 成功从数据库初始化 AppService (serviceID=%s, TenantID=%s, ServiceAlias=%s)",
        serviceID, appservice.TenantID, appservice.ServiceAlias)

    return appservice, nil
}
```

### 修复后的执行流程

```
┌──────────────────────────────────────────────────────────────────────┐
│ Deployment Add 协程              Pod Add 协程                         │
├──────────────────────────────────────────────────────────────────────┤
│ ① Load(key) → nil                                                    │
│ ② 获取 serviceID 的锁                                                │
│ ③ Lock() 🔒                                                          │
│ ④ Load(key) → nil (二次检查)                                         │
│ ⑤ InitCacheAppService()          ⑥ Load(key) → nil                  │
│    → 正在初始化...                ⑦ 获取同一个锁                     │
│                                   ⑧ Lock() → 阻塞等待... ⏳          │
│ ⑨ Store(realApp)                                                     │
│ ⑩ Unlock() 🔓                                                        │
│                                   ⑪ Lock() 成功 🔒                   │
│                                   ⑫ Load(key) → realApp ✅           │
│                                      (二次检查，已初始化)             │
│                                   ⑬ Unlock() 🔓                      │
│                                   ⑭ realApp.SetPods(pod1) ✅         │
│                                                                       │
│ 最终：realApp.pods = [pod1]  ✅ 所有协程操作同一对象                 │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 性能分析

### 内存开销

```
锁的数量 = 服务数量
1000 个服务 × sizeof(sync.Mutex) ≈ 1000 × 16 bytes = 16KB

可忽略不计 ✅
```

### 性能特征

| 场景 | 性能表现 | 说明 |
|------|---------|------|
| 不同 serviceID 并发初始化 | **完全并行** | 各自持有不同的锁，互不影响 |
| 同一 serviceID 并发初始化 | **串行执行** | 必须的，避免重复初始化和数据竞争 |
| 已初始化的访问 | **无锁快速返回** | 第一次检查就返回，零开销 |
| 锁等待时间 | **< 100ms** | 仅数据库查询时间，只在启动时发生 |

### 性能对比

| 指标 | 修复前 | 修复后 |
|------|--------|--------|
| 并发安全性 | ❌ 数据丢失 | ✅ 完全安全 |
| 重复初始化 | ❌ 可能重复 | ✅ 只执行一次 |
| 数据库查询 | N 次（并发数） | 1 次 |
| 内存占用 | 正常 | +16KB (1000服务) |
| CPU 开销 | 正常 | 几乎相同 |
| 已初始化访问 | 无锁 | 无锁 ✅ |

---

## 测试验证

### 1. 单元测试场景

```go
// 测试并发初始化
func TestConcurrentGetAppService(t *testing.T) {
    store := NewStore(dbmanager)
    serviceID := "test-service-id"

    var wg sync.WaitGroup
    results := make([]*v1.AppService, 10)

    // 10 个协程同时初始化同一个 serviceID
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()
            app, err := store.getAppService(serviceID, "v1", "creator1", true)
            assert.NoError(t, err)
            results[index] = app
        }(i)
    }

    wg.Wait()

    // 验证所有协程获得的是同一个对象
    for i := 1; i < 10; i++ {
        assert.True(t, results[0] == results[i], "应该是同一个对象实例")
    }
}
```

### 2. 集成测试

**部署后观察日志：**

```bash
# 正常情况
INFO [getAppService] 内存中未找到 AppService (serviceID=xxx)，开始从数据库初始化
INFO [getAppService] 成功从数据库初始化 AppService (serviceID=xxx)

# 并发情况（关键验证点）
DEBUG [getAppService] 并发场景：其他协程已完成初始化 AppService (serviceID=xxx)
# ↑ 这条日志表示双重检查锁生效

# GetAppPods 应该返回正确结果
INFO GetAppPods called for service_id: xxx
INFO App service found for service_id: xxx
INFO Got 1 pods for service_id: xxx  ✅ 正确
```

### 3. 压力测试

**场景：** 同时启动 1000 个服务，观察：
- 是否有重复初始化（数据库查询次数 = 服务数量）
- GetAppPods 返回是否正确
- 是否有数据竞争警告（使用 `go run -race`）

---

## 相关改动

### 修改的文件

**`worker/appm/store/store.go`**

1. **添加 initLocks 字段** (line 159)
   ```go
   initLocks sync.Map // map[serviceID]*sync.Mutex for AppService initialization
   ```

2. **导入 sync/atomic 包** (line 35)
   ```go
   "sync/atomic"
   ```

3. **重写 getAppService 函数** (line 794-841)
   - 实现双重检查锁模式
   - 添加并发场景日志

4. **修复 appCount 并发安全** (line 1140, 1162)
   ```go
   atomic.AddInt32(&a.appCount, 1)  // 替代 a.appCount++
   ```

### 向后兼容性

✅ **完全兼容**
- 不改变 API 接口
- 不改变数据结构（除了添加内部字段）
- 不改变外部行为（只修复 bug）

---

## 部署建议

### 1. 升级步骤

```bash
# 1. 备份当前版本
cp rainbond-worker rainbond-worker.backup

# 2. 构建新版本
make build

# 3. 滚动重启 worker
# - 先重启一个实例，观察日志
# - 确认无误后，逐步重启其他实例

# 4. 观察关键指标
# - GetAppPods 错误率
# - AppService 初始化时间
# - 数据库查询次数
```

### 2. 回滚方案

如果出现问题，可以立即回滚到备份版本：

```bash
cp rainbond-worker.backup rainbond-worker
systemctl restart rainbond-worker
```

### 3. 监控指标

```
关注以下指标 1 小时：
- GetAppPods 返回空列表的次数 → 应该降为 0
- AppService 初始化并发日志 → 应该能看到 "并发场景" 日志
- 数据库查询 QPS → 应该明显降低（避免重复查询）
```

---

## 总结

### 问题本质

Kubernetes Informer 并发事件 + Check-Then-Act 竞态条件 → AppService 对象被覆盖 → Pod 信息丢失

### 修复效果

✅ **彻底解决并发竞态问题**
✅ **保留 30 分钟 resync 的性能优化**
✅ **提升系统并发安全性**
✅ **几乎零性能损耗**
✅ **完全向后兼容**

### 长期价值

1. **稳定性提升**：消除了隐藏的并发 bug
2. **可扩展性**：大规模集群下更稳定
3. **可维护性**：代码更清晰，遵循最佳实践
4. **性能优化**：避免重复数据库查询

---

## 参考资料

- [Kubernetes Informer 机制](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes)
- [Double-Checked Locking Pattern](https://en.wikipedia.org/wiki/Double-checked_locking)
- [Go sync.Map 文档](https://pkg.go.dev/sync#Map)
- 相关 Issue: #2327 (performance issues caused by resync)
- 相关 Commit: 773bb6883 (fix: performance issues caused by resync)
