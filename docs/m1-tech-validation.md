# M1 技术验证点

> 版本: v1.0
> 更新时间: 2026-01-13
> 用途: 记录需要技术验证的关键点

---

## 一、验证点总览

| 编号 | 验证点 | 优先级 | 复杂度 | 状态 |
|------|--------|--------|--------|------|
| T1 | RSA 签名验证机制 | P0 | 中 | 待验证 |
| T2 | 前端 JS 嵌入 Go 二进制 | P0 | 中 | 待验证 |
| T3 | ConfigMap 读写与权限 | P0 | 低 | 待验证 |
| T4 | 插件启动时获取授权信息 | P0 | 中 | 待验证 |
| T5 | Cluster UID 获取方式 | P1 | 低 | 待验证 |
| T6 | AccessKey 与镜像仓库凭证绑定 | P1 | 高 | 待验证 |
| T7 | 插件依赖检测机制 | P1 | 中 | 待验证 |

---

## 二、详细验证内容

### T1: RSA 签名验证机制

**目标**：验证使用 RSA 非对称加密进行授权签名的可行性

**验证内容**：

1. **密钥生成**
   - 生成 RSA 密钥对（建议 2048 位或 4096 位）
   - 私钥保存在授权服务（云端）
   - 公钥嵌入到插件代码中

2. **签名流程**
   ```go
   // 授权服务端 - 签名
   func SignLicense(license *LicenseToken, privateKey *rsa.PrivateKey) (string, error) {
       // 1. 将授权信息序列化为 JSON
       data, _ := json.Marshal(license)

       // 2. 计算 SHA256 哈希
       hash := sha256.Sum256(data)

       // 3. 使用私钥签名
       signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])

       // 4. Base64 编码
       return base64.StdEncoding.EncodeToString(signature), err
   }
   ```

3. **验证流程**
   ```go
   // 插件端 - 验证
   func VerifyLicense(license *LicenseToken, signature string, publicKey *rsa.PublicKey) error {
       // 1. 将授权信息序列化为 JSON（与签名时相同）
       data, _ := json.Marshal(license)

       // 2. 计算 SHA256 哈希
       hash := sha256.Sum256(data)

       // 3. Base64 解码签名
       sig, _ := base64.StdEncoding.DecodeString(signature)

       // 4. 使用公钥验证
       return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], sig)
   }
   ```

**验证步骤**：

- [ ] 编写密钥生成工具
- [ ] 编写签名工具（模拟授权服务）
- [ ] 编写验证代码（模拟插件端）
- [ ] 测试正常签名验证
- [ ] 测试篡改数据后验证失败
- [ ] 测试使用错误公钥验证失败

**预期产出**：

- 可复用的签名/验证代码库
- 密钥管理方案

---

### T2: 前端 JS 嵌入 Go 二进制

**目标**：验证将前端静态资源嵌入 Go 二进制并动态返回的可行性

**验证内容**：

1. **使用 Go embed 包**
   ```go
   package main

   import (
       "embed"
       "net/http"
   )

   //go:embed static/*
   var staticFiles embed.FS

   func main() {
       http.Handle("/static/", http.FileServer(http.FS(staticFiles)))
       http.ListenAndServe(":8080", nil)
   }
   ```

2. **带授权验证的资源返回**
   ```go
   func handleStaticWithAuth(w http.ResponseWriter, r *http.Request) {
       // 1. 验证授权
       if !verifyLicense() {
           http.Error(w, "Unauthorized", http.StatusForbidden)
           return
       }

       // 2. 读取嵌入的文件
       content, err := staticFiles.ReadFile("static/main.js")
       if err != nil {
           http.Error(w, "Not found", http.StatusNotFound)
           return
       }

       // 3. 设置正确的 Content-Type
       w.Header().Set("Content-Type", "application/javascript")
       w.Write(content)
   }
   ```

**验证步骤**：

- [ ] 创建示例 Go 项目
- [ ] 嵌入示例前端文件（JS、CSS）
- [ ] 编译为单一二进制
- [ ] 验证文件可正常访问
- [ ] 验证授权检查后才返回文件
- [ ] 测试文件大小和加载性能

**关注点**：

- 嵌入后二进制大小增加多少？
- 是否影响加载性能？
- 如何处理前端构建产物的更新？

**预期产出**：

- 示例代码
- 性能测试报告
- 构建流程方案

---

### T3: ConfigMap 读写与权限

**目标**：验证插件如何安全地读写授权信息 ConfigMap

**验证内容**：

1. **ConfigMap 结构设计**
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: rbd-license-info
     namespace: rbd-system
   data:
     license_token: |
       {
         "cluster_id": "xxx",
         "tier": "advanced",
         "expire_at": 1735689600,
         "allowed_plugins": ["enterprise-base", "rainbill"],
         "signature": "base64-encoded-signature..."
       }
   ```

2. **权限配置**
   ```yaml
   # 授权插件需要写权限
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: license-plugin-role
     namespace: rbd-system
   rules:
   - apiGroups: [""]
     resources: ["configmaps"]
     resourceNames: ["rbd-license-info"]
     verbs: ["get", "update", "patch"]

   # 其他插件只需要读权限
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: enterprise-plugin-role
     namespace: rbd-system
   rules:
   - apiGroups: [""]
     resources: ["configmaps"]
     resourceNames: ["rbd-license-info"]
     verbs: ["get"]
   ```

3. **读取代码**
   ```go
   func GetLicenseFromConfigMap(clientset *kubernetes.Clientset) (*LicenseToken, error) {
       cm, err := clientset.CoreV1().ConfigMaps("rbd-system").Get(
           context.TODO(), "rbd-license-info", metav1.GetOptions{})
       if err != nil {
           return nil, err
       }

       var token LicenseToken
       err = json.Unmarshal([]byte(cm.Data["license_token"]), &token)
       return &token, err
   }
   ```

**验证步骤**：

- [ ] 创建 ConfigMap 和 RBAC 配置
- [ ] 验证授权插件可以写入
- [ ] 验证其他插件可以读取
- [ ] 验证未授权 Pod 无法读取
- [ ] 测试 ConfigMap 更新后的读取

**预期产出**：

- RBAC 配置模板
- 读写代码示例

---

### T4: 插件启动时获取授权信息

**目标**：验证企业插件如何在启动时获取并验证授权信息

**验证内容**：

1. **启动流程设计**
   ```go
   func main() {
       // 1. 初始化 K8s 客户端
       config, _ := rest.InClusterConfig()
       clientset, _ := kubernetes.NewForConfig(config)

       // 2. 获取授权信息
       license, err := GetLicenseFromConfigMap(clientset)
       if err != nil {
           log.Fatal("Failed to get license:", err)
       }

       // 3. 验证签名
       if err := VerifyLicense(license, license.Signature, embeddedPublicKey); err != nil {
           log.Fatal("License verification failed:", err)
       }

       // 4. 检查插件权限
       if !contains(license.AllowedPlugins, "enterprise-base") {
           log.Fatal("Plugin not authorized")
       }

       // 5. 检查有效期
       if time.Now().Unix() > license.ExpireAt {
           log.Fatal("License expired")
       }

       // 6. 启动服务
       startServer()

       // 7. 启动定时重验证
       go periodicVerification(clientset)
   }
   ```

2. **定时重验证**
   ```go
   func periodicVerification(clientset *kubernetes.Clientset) {
       ticker := time.NewTicker(1 * time.Hour)
       for range ticker.C {
           license, err := GetLicenseFromConfigMap(clientset)
           if err != nil || !isValid(license) {
               // 停止服务或返回 403
               stopService()
           }
       }
   }
   ```

**验证步骤**：

- [ ] 编写启动时验证代码
- [ ] 测试正常启动流程
- [ ] 测试无授权信息时的行为
- [ ] 测试签名无效时的行为
- [ ] 测试插件不在授权列表时的行为
- [ ] 测试过期后的行为
- [ ] 测试定时重验证机制

**预期产出**：

- 启动验证代码模板
- 各种错误场景的处理方案

---

### T5: Cluster UID 获取方式

**目标**：验证获取集群唯一标识的方法

**验证内容**：

1. **使用 kube-system namespace UID**
   ```go
   func GetClusterUID(clientset *kubernetes.Clientset) (string, error) {
       ns, err := clientset.CoreV1().Namespaces().Get(
           context.TODO(), "kube-system", metav1.GetOptions{})
       if err != nil {
           return "", err
       }
       return string(ns.UID), nil
   }
   ```

2. **命令行获取**
   ```bash
   kubectl get namespace kube-system -o jsonpath='{.metadata.uid}'
   ```

**验证步骤**：

- [ ] 在不同 K8s 集群测试 UID 唯一性
- [ ] 验证集群重建后 UID 是否变化
- [ ] 验证普通用户是否能修改此 UID
- [ ] 测试代码获取 UID

**关注点**：

- kube-system namespace 的 UID 是否足够稳定？
- 是否需要组合多个标识增强唯一性？

**预期产出**：

- Cluster UID 获取代码
- 稳定性评估报告

---

### T6: AccessKey 与镜像仓库凭证绑定

**目标**：验证如何将 AccessKey 与镜像仓库拉取凭证绑定

**验证内容**：

1. **应用市场侧**
   - AccessKey 生成时关联镜像仓库用户名/密码
   - 验证 AccessKey 时返回对应凭证
   - AccessKey 过期时凭证同步失效

2. **Rainbond 平台侧**
   ```go
   // 获取镜像拉取凭证
   func GetImagePullSecret(accessKey string) (*ImageCredential, error) {
       // 调用应用市场 API
       resp, err := http.Get(fmt.Sprintf("%s/api/v1/credentials?ak=%s",
           marketURL, accessKey))
       // ...
   }

   // 创建 K8s Secret
   func CreateImagePullSecret(cred *ImageCredential) error {
       secret := &corev1.Secret{
           ObjectMeta: metav1.ObjectMeta{
               Name:      "rbd-plugin-registry-secret",
               Namespace: "rbd-system",
           },
           Type: corev1.SecretTypeDockerConfigJson,
           Data: map[string][]byte{
               ".dockerconfigjson": createDockerConfig(cred),
           },
       }
       // ...
   }
   ```

**验证步骤**：

- [ ] 设计 AccessKey 与凭证的关联方案
- [ ] 测试凭证过期后镜像拉取失败
- [ ] 测试凭证更新机制
- [ ] 验证 Rainbond 应用部署如何使用 imagePullSecrets

**关注点**：

- 应用市场需要做哪些改动？
- 凭证刷新机制如何设计？
- 如何处理凭证轮换？

**预期产出**：

- AccessKey-凭证绑定方案
- 应用市场接口设计

---

### T7: 插件依赖检测机制

**目标**：验证企业插件如何检测授权插件是否已安装

**验证内容**：

1. **方案 A：检测 ConfigMap 存在性**
   ```go
   func IsLicensePluginInstalled(clientset *kubernetes.Clientset) bool {
       _, err := clientset.CoreV1().ConfigMaps("rbd-system").Get(
           context.TODO(), "rbd-license-info", metav1.GetOptions{})
       return err == nil
   }
   ```

2. **方案 B：检测特定 Label 的 Service**
   ```go
   func IsLicensePluginRunning(clientset *kubernetes.Clientset) bool {
       services, err := clientset.CoreV1().Services("").List(
           context.TODO(), metav1.ListOptions{
               LabelSelector: "rainbond.io/plugin-type=license",
           })
       return err == nil && len(services.Items) > 0
   }
   ```

3. **方案 C：检测 RBDPlugin CRD**
   ```go
   func IsLicensePluginInstalled(rbdClient *rbdclientset.Clientset) bool {
       _, err := rbdClient.RainbondV1alpha1().RBDPlugins("").Get(
           context.TODO(), "rainbond-license", metav1.GetOptions{})
       return err == nil
   }
   ```

**验证步骤**：

- [ ] 测试各方案的可行性
- [ ] 评估哪种方案最可靠
- [ ] 测试授权插件未安装时的处理
- [ ] 测试授权插件安装但未授权时的处理

**预期产出**：

- 推荐的检测方案
- 检测代码示例

---

## 三、验证优先级

### 第一周重点

| 验证点 | 原因 |
|--------|------|
| T1 RSA 签名验证 | 核心安全机制，必须先验证 |
| T2 前端嵌入二进制 | 影响插件架构设计 |
| T3 ConfigMap 读写 | 基础能力，其他验证依赖 |

### 第二周补充

| 验证点 | 原因 |
|--------|------|
| T4 启动时验证 | 基于 T1/T3 的集成验证 |
| T5 Cluster UID | 相对简单，可快速完成 |
| T7 依赖检测 | 确定最佳方案 |

### 后续跟进

| 验证点 | 原因 |
|--------|------|
| T6 AccessKey 绑定 | 需要应用市场配合，复杂度高 |

---

## 四、验证产出模板

每个验证点完成后，请记录：

```markdown
## T{n}: {验证点名称}

### 验证结论
- [ ] 可行 / [ ] 不可行 / [ ] 需要调整

### 验证过程
{描述验证步骤和结果}

### 代码示例
{可复用的代码}

### 问题与风险
{发现的问题}

### 建议方案
{最终推荐的实现方案}
```

---

*文档结束*
