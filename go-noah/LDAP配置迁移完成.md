# LDAP 配置迁移完成

已成功将 goInsight 项目中的 LDAP 配置迁移到 go-noah 项目。

## 迁移的配置

### goInsight 原始配置（config.yaml）
```yaml
ldap:
  enable: true
  host: "192.168.100.51"
  port: 389
  use_ssl: false
  base_dn: "dc=qixiangyun,dc=com"
  bind_dn: "cn=admin,dc=qixiangyun,dc=com"
  bind_pass: "qxy906906"
  user_filter: "(&(objectClass=inetOrgPerson)(uid=%s))"
  attributes:
    nickname: "sn"
    email: "mail"
    mobile: "mobile"
```

### go-noah 迁移后配置（config/local.yml）
```yaml
ldap:
  enable: true               # 已启用
  host: 192.168.100.51       # LDAP 服务器地址
  port: 389                  # LDAP 端口
  use_ssl: false             # 不使用 SSL/TLS
  base_dn: "dc=qixiangyun,dc=com"  # 基础 DN
  bind_dn: "cn=admin,dc=qixiangyun,dc=com"  # 管理员绑定 DN
  bind_pass: "qxy906906"     # 管理员密码
  user_filter: "(&(objectClass=inetOrgPerson)(uid=%s))"  # 用户搜索过滤器
  attributes:
    nickname: "sn"           # 昵称属性名
    email: "mail"           # 邮箱属性名
    mobile: "mobile"        # 手机号属性名
```

## 配置说明

### 配置项对比

| 配置项 | goInsight | go-noah | 说明 |
|--------|-----------|---------|------|
| `enable` | `true` | `true` | LDAP 认证已启用 |
| `host` | `192.168.100.51` | `192.168.100.51` | LDAP 服务器地址 |
| `port` | `389` | `389` | LDAP 端口（非 SSL） |
| `use_ssl` | `false` | `false` | 不使用 SSL/TLS |
| `base_dn` | `dc=qixiangyun,dc=com` | `dc=qixiangyun,dc=com` | 基础 DN |
| `bind_dn` | `cn=admin,dc=qixiangyun,dc=com` | `cn=admin,dc=qixiangyun,dc=com` | 管理员绑定 DN |
| `bind_pass` | `qxy906906` | `qxy906906` | 管理员密码 |
| `user_filter` | `(&(objectClass=inetOrgPerson)(uid=%s))` | `(&(objectClass=inetOrgPerson)(uid=%s))` | 用户搜索过滤器 |
| `attributes.nickname` | `sn` | `sn` | 昵称属性（使用 sn 而不是 cn） |
| `attributes.email` | `mail` | `mail` | 邮箱属性 |
| `attributes.mobile` | `mobile` | `mobile` | 手机号属性 |

### 关键差异

1. **用户过滤器**：
   - goInsight 使用：`(&(objectClass=inetOrgPerson)(uid=%s))`
   - 这是一个复合过滤器，要求：
     - `objectClass` 必须是 `inetOrgPerson`
     - `uid` 必须匹配输入的用户名

2. **昵称属性**：
   - 使用 `sn`（surname）而不是 `cn`（common name）
   - 这表示昵称从 LDAP 的 `sn` 属性获取

## 测试 LDAP 登录

### 1. 确保服务器已启动
```bash
cd go-noah
go run ./cmd/server
```

### 2. 测试 LDAP 登录
```bash
# 使用测试脚本
./test_login.sh ldap_username ldap_password

# 或使用 curl
curl -X POST http://127.0.0.1:8000/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your_ldap_username",
    "password": "your_ldap_password"
  }'
```

### 3. 预期行为

**LDAP 认证成功：**
- 系统会先尝试 LDAP 认证
- 如果 LDAP 认证成功：
  - 自动创建或更新本地用户记录（JIT）
  - 返回 JWT Token
  - 用户信息从 LDAP 同步到本地数据库

**LDAP 认证失败：**
- 记录警告日志
- 自动回退到本地密码认证
- 如果本地用户存在且密码正确，仍然可以登录

### 4. 查看日志

服务器日志会显示：
- LDAP 连接状态
- LDAP 认证结果
- JIT 用户同步情况
- 认证失败原因（如果失败）

## 注意事项

1. **网络连接**：
   - 确保服务器可以访问 `192.168.100.51:389`
   - 检查防火墙规则

2. **LDAP 服务器状态**：
   - 确保 LDAP 服务器正常运行
   - 验证管理员账号 `cn=admin,dc=qixiangyun,dc=com` 和密码 `qxy906906` 有效

3. **用户过滤器**：
   - 确保 LDAP 中的用户 `objectClass` 包含 `inetOrgPerson`
   - 确保用户有 `uid` 属性

4. **属性映射**：
   - 昵称从 `sn` 属性获取（不是 `cn`）
   - 邮箱从 `mail` 属性获取
   - 手机号从 `mobile` 属性获取

5. **安全性**：
   - 密码以明文形式存储在配置文件中
   - 生产环境建议使用环境变量或密钥管理服务

## 配置验证

可以使用以下 LDAP 查询工具验证配置：

```bash
# 使用 ldapsearch 测试（如果已安装）
ldapsearch -x -H ldap://192.168.100.51:389 \
  -D "cn=admin,dc=qixiangyun,dc=com" \
  -w "qxy906906" \
  -b "dc=qixiangyun,dc=com" \
  "(&(objectClass=inetOrgPerson)(uid=testuser))"
```

## 故障排查

### 问题 1: LDAP 连接失败
```
LDAP connect failed: dial tcp 192.168.100.51:389: connect: connection refused
```
**解决**：
- 检查 LDAP 服务器是否运行
- 检查网络连接
- 检查防火墙规则

### 问题 2: LDAP 管理员绑定失败
```
LDAP admin bind failed: Invalid Credentials
```
**解决**：
- 检查 `bind_dn` 和 `bind_pass` 是否正确
- 验证管理员账号是否有效

### 问题 3: 用户未找到
```
LDAP user not found or multiple entries
```
**解决**：
- 检查 `user_filter` 是否正确
- 验证用户是否存在
- 检查 `base_dn` 是否正确

### 问题 4: 属性获取失败
**解决**：
- 检查 LDAP 用户是否有对应的属性（sn, mail, mobile）
- 验证属性名称是否正确

## 下一步

1. 启动服务器测试 LDAP 登录
2. 验证 JIT 用户同步功能
3. 测试 LDAP 失败后的本地认证回退
4. 根据实际使用情况调整配置

配置已成功迁移，可以直接使用！

