# goInsight 迁移到 go-noah 完整方案

## 一、项目结构对比

### goInsight 结构（源项目）
```
goinsight/
├── bootstrap/          # 初始化配置、日志、数据库
├── config/             # 配置结构定义
├── global/             # 全局变量（DB、Redis、JWT、Cron等）
├── middleware/         # 中间件（JWT、日志、OTP、权限）
├── pkg/                # 工具包
├── routers/            # 路由注册
└── internal/
    ├── common/         # 环境管理、数据库配置
    │   ├── forms/      # 表单验证
    │   ├── models/     # 数据模型
    │   ├── routers/    # 路由定义
    │   ├── services/   # 业务逻辑
    │   ├── tasks/      # 定时任务
    │   └── views/      # 控制器（Handler）
    ├── das/            # 数据访问服务（SQL执行）
    ├── inspect/        # SQL语法审核
    ├── orders/         # 工单系统
    └── users/          # 用户管理
```

### go-noah 结构（目标项目）
```
go-noah/
├── cmd/                # 应用入口
├── config/             # 配置文件
├── pkg/                # 公共包（config、jwt、log、ldap等）
└── internal/
    ├── handler/        # HTTP处理器（对应 views）
    ├── service/        # 业务逻辑层
    ├── repository/     # 数据访问层
    ├── model/          # 数据模型
    ├── middleware/     # 中间件
    ├── router/         # 路由定义
    ├── server/         # 服务器启动
    ├── job/            # 后台任务
    └── task/           # 定时任务
```

## 二、模块功能清单

### 1. 用户模块 (users) - 已部分完成
| goInsight | 功能 | go-noah状态 |
|-----------|------|------------|
| InsightUsers | 用户表 | ✅ AdminUser |
| InsightRoles | 角色表 | ✅ Role |
| InsightOrganizations | 组织树 | ❌ 待迁移 |
| InsightOrganizationsUsers | 用户组织关联 | ❌ 待迁移 |

### 2. 通用模块 (common) - 待迁移
| goInsight | 功能 | go-noah状态 |
|-----------|------|------------|
| InsightDBEnvironments | 环境管理 | ❌ 待迁移 |
| InsightDBConfig | 数据库实例配置 | ❌ 待迁移 |
| InsightDBSchemas | 数据库Schema | ❌ 待迁移 |

### 3. DAS模块 (das) - 待迁移
| goInsight | 功能 | go-noah状态 |
|-----------|------|------------|
| InsightDASUserSchemaPermissions | 库权限 | ❌ 待迁移 |
| InsightDASUserTablePermissions | 表权限 | ❌ 待迁移 |
| InsightDASAllowedOperations | 允许的SQL操作 | ❌ 待迁移 |
| InsightDASRecords | 执行记录 | ❌ 待迁移 |
| InsightDASFavorites | 收藏夹 | ❌ 待迁移 |

### 4. 审核模块 (inspect) - 待迁移
| goInsight | 功能 | go-noah状态 |
|-----------|------|------------|
| InsightInspectParams | 审核参数配置 | ❌ 待迁移 |
| checker/* | 语法检查器 | ❌ 待迁移 |
| controllers/* | 审核控制器 | ❌ 待迁移 |

### 5. 工单模块 (orders) - 待迁移
| goInsight | 功能 | go-noah状态 |
|-----------|------|------------|
| InsightOrderRecords | 工单记录 | ❌ 待迁移 |
| InsightOrderTasks | 工单任务 | ❌ 待迁移 |
| InsightOrderOpLogs | 操作日志 | ❌ 待迁移 |
| InsightOrderMessages | 消息推送 | ❌ 待迁移 |
| scheduler/* | 定时执行 | ❌ 待迁移 |
| api/mysql/* | MySQL执行器 | ❌ 待迁移 |
| api/tidb/* | TiDB执行器 | ❌ 待迁移 |

## 三、迁移步骤

### 阶段一：基础模块迁移（环境与数据库配置）

#### 1.1 创建 Model 文件
```
go-noah/internal/model/
├── environment.go      # 环境配置
├── dbconfig.go         # 数据库实例配置
└── schema.go           # Schema信息
```

#### 1.2 配置结构扩展
在 `go-noah/config/local.yml` 中添加：
```yaml
# goInsight 配置（新增）
goinsight:
  remote_db:
    username: "readonly"
    password: "password"
  das:
    max_execution_time: 30
    default_return_rows: 100
    max_return_rows: 1000
    allowed_useragents: ["Mozilla", "Chrome"]
  ghost:
    path: "/usr/bin/gh-ost"
    args: ["--assume-rbr"]
  notify:
    notice_url: "https://example.com/notify"
    wechat:
      enable: false
      webhook: ""
    mail:
      enable: false
      username: ""
      password: ""
      host: ""
      port: 25
    dingtalk:
      enable: false
      webhook: ""
      keywords: ""
  crontab:
    sync_db_metas: "0 */2 * * *"
```

#### 1.3 创建目录结构
```bash
mkdir -p go-noah/internal/model/insight
mkdir -p go-noah/internal/handler/insight
mkdir -p go-noah/internal/service/insight
mkdir -p go-noah/internal/repository/insight
mkdir -p go-noah/pkg/insight
```

### 阶段二：用户与组织模块完善

#### 2.1 组织管理 Model
```go
// go-noah/internal/model/organization.go
type Organization struct {
    ID        uint64         `gorm:"primaryKey"`
    Name      string         `gorm:"type:varchar(32);not null;uniqueIndex:uniq_name"`
    ParentID  uint64         `gorm:"not null;default:0;uniqueIndex:uniq_name"`
    Key       string         `gorm:"type:varchar(256);uniqueIndex:uniq_key"`
    Level     uint64         `gorm:"not null;default:1"`
    Path      datatypes.JSON `gorm:"type:json"`
    Creator   string         `gorm:"type:varchar(64)"`
    Updater   string         `gorm:"type:varchar(64)"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

type OrganizationUser struct {
    gorm.Model
    UID             uint64 `gorm:"not null;uniqueIndex"`
    OrganizationKey string `gorm:"type:varchar(256);index"`
}
```

#### 2.2 API路由
```
GET    /api/v1/admin/organizations          # 获取组织树
POST   /api/v1/admin/organizations/root     # 创建根节点
POST   /api/v1/admin/organizations/child    # 创建子节点
PUT    /api/v1/admin/organizations/:id      # 更新节点
DELETE /api/v1/admin/organizations/:id      # 删除节点
GET    /api/v1/admin/organizations/users    # 获取组织用户
POST   /api/v1/admin/organizations/users    # 绑定用户
DELETE /api/v1/admin/organizations/users    # 解绑用户
```

### 阶段三：通用模块迁移

#### 3.1 环境管理
```go
// go-noah/internal/model/insight/environment.go
type DBEnvironment struct {
    gorm.Model
    Name string `gorm:"type:varchar(32);not null;uniqueIndex"`
}
```

#### 3.2 数据库配置
```go
// go-noah/internal/model/insight/dbconfig.go
type DBConfig struct {
    gorm.Model
    InstanceID       uuid.UUID      `gorm:"type:char(36);uniqueIndex"`
    Hostname         string         `gorm:"type:varchar(128);not null;uniqueIndex:uniq_hostname"`
    Port             int            `gorm:"type:int;not null;default:3306;uniqueIndex:uniq_hostname"`
    UserName         string         `gorm:"type:varchar(128);not null"`
    Password         string         `gorm:"type:varchar(256);not null"` // 需加密存储
    UseType          string         `gorm:"type:varchar(20);default:'工单';uniqueIndex:uniq_hostname"` // 查询/工单
    DbType           string         `gorm:"type:varchar(20);default:'MySQL'"` // MySQL/TiDB/ClickHouse
    Environment      int            `gorm:"type:int"`
    InspectParams    datatypes.JSON `gorm:"type:json"`
    OrganizationKey  string         `gorm:"type:varchar(256);index"`
    OrganizationPath datatypes.JSON `gorm:"type:json"`
    Remark           string         `gorm:"type:varchar(256)"`
}
```

#### 3.3 API路由
```
# 环境管理
GET    /api/v1/admin/environments           # 获取环境列表
POST   /api/v1/admin/environments           # 创建环境
PUT    /api/v1/admin/environments/:id       # 更新环境
DELETE /api/v1/admin/environments/:id       # 删除环境

# 数据库配置
GET    /api/v1/admin/dbconfigs              # 获取数据库配置列表
POST   /api/v1/admin/dbconfigs              # 创建数据库配置
PUT    /api/v1/admin/dbconfigs/:id          # 更新数据库配置
DELETE /api/v1/admin/dbconfigs/:id          # 删除数据库配置
```

### 阶段四：DAS模块迁移（数据访问服务）

#### 4.1 Model 定义
```go
// go-noah/internal/model/insight/das.go
// 用户库权限
type DASUserSchemaPermission struct {
    gorm.Model
    Username   string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_schema"`
    Schema     string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_schema"`
    InstanceID uuid.UUID `gorm:"type:char(36);uniqueIndex:uniq_schema;index"`
}

// 用户表权限
type DASUserTablePermission struct {
    gorm.Model
    Username   string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_table"`
    Schema     string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_table"`
    Table      string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_table"`
    InstanceID uuid.UUID `gorm:"type:char(36);uniqueIndex:uniq_table;index"`
    Rule       string    `gorm:"type:varchar(10);default:'allow'"` // allow/deny
}

// 允许的操作
type DASAllowedOperation struct {
    gorm.Model
    Name     string `gorm:"type:varchar(128);not null;uniqueIndex"`
    IsEnable bool   `gorm:"default:false"`
    Remark   string `gorm:"type:varchar(1024)"`
}

// 执行记录
type DASRecord struct {
    gorm.Model
    Username   string    `gorm:"type:varchar(128);index"`
    InstanceID uuid.UUID `gorm:"type:char(36);index"`
    Schema     string    `gorm:"type:varchar(128)"`
    SQL        string    `gorm:"type:text"`
    Duration   int64     // 执行时长(ms)
    RowCount   int64     // 返回行数
    Error      string    `gorm:"type:text"`
}

// 收藏夹
type DASFavorite struct {
    gorm.Model
    Username string `gorm:"type:varchar(128);index"`
    Title    string `gorm:"type:varchar(256)"`
    SQL      string `gorm:"type:text"`
}
```

#### 4.2 核心服务
- **MySQL执行器**: 复用 `goinsight/internal/das/dao/db.go`
- **ClickHouse执行器**: 复用 `goinsight/internal/das/dao/clickhouse.go`
- **SQL解析器**: 复用 `goinsight/internal/das/parser/*`

#### 4.3 API路由
```
# 数据查询
GET    /api/v1/das/environments             # 获取可用环境
GET    /api/v1/das/schemas                  # 获取可访问的Schema
GET    /api/v1/das/tables                   # 获取表列表
POST   /api/v1/das/execute/mysql            # 执行MySQL查询
POST   /api/v1/das/execute/clickhouse       # 执行ClickHouse查询
GET    /api/v1/das/table-info               # 获取表结构
GET    /api/v1/das/dbdict                   # 获取数据字典
GET    /api/v1/das/history                  # 查询历史
GET    /api/v1/das/favorites                # 收藏夹列表
POST   /api/v1/das/favorites                # 添加收藏
PUT    /api/v1/das/favorites/:id            # 更新收藏
DELETE /api/v1/das/favorites/:id            # 删除收藏
GET    /api/v1/das/user/grants              # 用户授权列表

# 管理员接口
GET    /api/v1/admin/das/schemas/grant      # 获取库授权
POST   /api/v1/admin/das/schemas/grant      # 创建库授权
DELETE /api/v1/admin/das/schemas/grant/:id  # 删除库授权
GET    /api/v1/admin/das/tables/grant       # 获取表授权
POST   /api/v1/admin/das/tables/grant       # 创建表授权
DELETE /api/v1/admin/das/tables/grant/:id   # 删除表授权
```

### 阶段五：SQL审核模块迁移

#### 5.1 Model 定义
```go
// go-noah/internal/model/insight/inspect.go
type InspectParams struct {
    gorm.Model
    Params datatypes.JSON `gorm:"type:json"`
    Remark string         `gorm:"type:varchar(256);uniqueIndex"`
}
```

#### 5.2 核心组件迁移
```
goinsight/internal/inspect/ → go-noah/pkg/insight/
├── checker/          # 检查器
├── config/           # 审核配置
└── controllers/      # 审核控制器
    ├── dao/          # 数据访问
    ├── extract/      # 表提取
    ├── logics/       # 逻辑处理
    ├── parser/       # SQL解析
    ├── process/      # 处理流程
    ├── rules/        # 审核规则
    └── traverses/    # AST遍历
```

#### 5.3 API路由
```
# 管理员接口
GET    /api/v1/admin/inspect/params         # 获取审核参数
PUT    /api/v1/admin/inspect/params/:id     # 更新审核参数

# 语法审核（内部调用）
POST   /api/v1/inspect/syntax               # 语法检查
```

### 阶段六：工单模块迁移

#### 6.1 Model 定义
```go
// go-noah/internal/model/insight/order.go
// 工单记录
type OrderRecord struct {
    gorm.Model
    Title            string         `gorm:"type:varchar(128);index"`
    OrderID          uuid.UUID      `gorm:"type:char(36);uniqueIndex"`
    HookOrderID      uuid.UUID      `gorm:"type:char(36);index"`
    Remark           string         `gorm:"type:varchar(1024)"`
    IsRestrictAccess bool           `gorm:"default:false"`
    DBType           string         `gorm:"type:varchar(20);default:'MySQL'"`
    SQLType          string         `gorm:"type:varchar(20);default:'DML'"`
    Environment      int            `gorm:"index"`
    Applicant        string         `gorm:"type:varchar(32);index"`
    Organization     string         `gorm:"type:varchar(256);index"`
    Approver         datatypes.JSON `gorm:"type:json"`
    Executor         datatypes.JSON `gorm:"type:json"`
    Reviewer         datatypes.JSON `gorm:"type:json"`
    CC               datatypes.JSON `gorm:"type:json"`
    InstanceID       uuid.UUID      `gorm:"type:char(36);index"`
    Schema           string         `gorm:"type:varchar(128)"`
    Progress         string         `gorm:"type:varchar(20);default:'待审核'"`
    ExecuteResult    string         `gorm:"type:varchar(32)"`
    ScheduleTime     *time.Time
    FixVersion       string         `gorm:"type:varchar(128);index"`
    Content          string         `gorm:"type:text"`
    ExportFileFormat string         `gorm:"type:varchar(10);default:'XLSX'"`
}

// 工单任务
type OrderTask struct {
    gorm.Model
    OrderID  uuid.UUID      `gorm:"type:char(36);index"`
    TaskID   uuid.UUID      `gorm:"type:char(36);index"`
    DBType   string         `gorm:"type:varchar(20)"`
    SQLType  string         `gorm:"type:varchar(20)"`
    Executor string         `gorm:"type:varchar(128)"`
    SQL      string         `gorm:"type:text"`
    Progress string         `gorm:"type:varchar(20);default:'未执行'"`
    Result   datatypes.JSON `gorm:"type:json"`
}

// 操作日志
type OrderOpLog struct {
    gorm.Model
    Username string    `gorm:"type:varchar(32);index"`
    OrderID  uuid.UUID `gorm:"type:char(36);index"`
    Msg      string    `gorm:"type:varchar(1024)"`
}

// 消息记录
type OrderMessage struct {
    gorm.Model
    OrderID  uuid.UUID      `gorm:"type:char(36);index"`
    Receiver datatypes.JSON `gorm:"type:json"`
    Response string         `gorm:"type:text"`
}
```

#### 6.2 核心组件迁移
```
goinsight/internal/orders/api/ → go-noah/pkg/insight/executor/
├── base/             # 基础组件
├── execute/          # 执行器
├── file/             # 文件处理
├── mysql/            # MySQL执行
└── tidb/             # TiDB执行
```

#### 6.3 API路由
```
# 工单相关
GET    /api/v1/orders/environments          # 获取环境
GET    /api/v1/orders/instances             # 获取实例
POST   /api/v1/orders/syntax-inspect        # 语法审核
GET    /api/v1/orders/schemas               # 获取Schema
GET    /api/v1/orders/users                 # 获取用户列表
POST   /api/v1/orders/commit                # 提交工单
GET    /api/v1/orders/list                  # 工单列表
GET    /api/v1/orders/detail/:order_id      # 工单详情
GET    /api/v1/orders/detail/oplogs         # 操作日志

# 工单操作
PUT    /api/v1/orders/operate/approve       # 审批
PUT    /api/v1/orders/operate/feedback      # 反馈
PUT    /api/v1/orders/operate/review        # 复核
PUT    /api/v1/orders/operate/close         # 关闭
PUT    /api/v1/orders/operate/schedule      # 更新计划时间

# 任务执行
POST   /api/v1/orders/tasks/generate        # 生成任务
GET    /api/v1/orders/tasks/:order_id       # 获取任务列表
GET    /api/v1/orders/tasks/preview         # 预览任务
POST   /api/v1/orders/tasks/execute-single  # 执行单个任务
POST   /api/v1/orders/tasks/execute-all     # 执行所有任务
GET    /api/v1/orders/download/:task_id     # 下载导出文件

# Hook接口
POST   /api/v1/orders/hook                  # 外部Hook

# WebSocket
GET    /ws/:channel                         # 实时进度
```

### 阶段七：定时任务与后台任务

#### 7.1 定时任务迁移
```go
// go-noah/internal/task/insight.go
type InsightTask struct {
    *Task
    repo *repository.InsightRepository
}

// 同步数据库元数据
func (t *InsightTask) SyncDBMetas() error {
    // 从 goInsight/internal/common/tasks/tasks.go 迁移
}
```

#### 7.2 调度器迁移
```go
// go-noah/internal/service/insight/scheduler.go
// 从 goInsight/internal/orders/scheduler/scheduler.go 迁移
```

## 四、配置文件合并

### 完整配置文件结构
```yaml
# go-noah/config/local.yml
env: local

http:
  host: 127.0.0.1
  port: 8000

security:
  api_sign:
    app_key: 123456
    app_security: 123456
  jwt:
    key: QQYnRFerJTSEcrfB89fw8prOaObmrch8

data:
  db:
    user:
      driver: mysql
      dsn: root:password@tcp(127.0.0.1:3306)/noah?charset=utf8mb4&parseTime=True&loc=Local
  redis:
    addr: 127.0.0.1:6379
    password: ""
    db: 0

log:
  log_level: debug
  encoding: console
  log_file_name: "./storage/logs/server.log"

ldap:
  enable: true
  host: ldap.example.com
  port: 389
  use_ssl: false
  base_dn: "dc=example,dc=com"
  bind_dn: "cn=admin,dc=example,dc=com"
  bind_pass: "password"
  user_filter: "(&(objectClass=inetOrgPerson)(uid=%s))"
  attributes:
    nickname: "sn"
    email: "mail"
    mobile: "mobile"

# ===== goInsight 配置 =====
app:
  title: "goInsight"

crontab:
  sync_db_metas: "0 */2 * * *"

remote_db:
  username: "readonly"
  password: "password"

das:
  max_execution_time: 30
  default_return_rows: 100
  max_return_rows: 1000
  allowed_useragents:
    - "Mozilla"
    - "Chrome"

ghost:
  path: "/usr/bin/gh-ost"
  args:
    - "--assume-rbr"

notify:
  notice_url: "https://example.com/notify"
  wechat:
    enable: false
    webhook: ""
  mail:
    enable: false
    username: ""
    password: ""
    host: ""
    port: 25
  dingtalk:
    enable: false
    webhook: ""
    keywords: ""
```

## 五、迁移文件清单

⚠️ **重要说明**：以下文件清单只是列出了需要迁移的文件，但**不是直接复制**。所有文件都需要：
1. 修改导入路径（`goInsight` → `go-noah`）
2. 适配新框架架构（上下文、依赖注入等）
3. 适配配置系统
4. 适配返回格式

### ⚠️ SQL审核模块迁移状态

**当前状态**：只实现了基础框架，缺少**53条审核规则**和**完整的规则引擎架构**！

**详细缺失清单**：请参考 `go-noah/docs/SQL审核模块完整迁移清单.md`

### 需要迁移的核心文件

#### 1. SQL解析器（已迁移，需要验证）
```
goinsight/internal/das/parser/* → go-noah/internal/inspect/parser/ ✅
goinsight/internal/inspect/controllers/parser/* → go-noah/internal/inspect/parser/ ✅
```

#### 2. SQL审核规则（⚠️ 严重缺失，需要完整迁移）
```
goinsight/internal/inspect/checker/* → go-noah/internal/inspect/checker/ ⚠️ 部分实现
goinsight/internal/inspect/controllers/rules/* → go-noah/internal/inspect/rules/ ❌ 缺失
goinsight/internal/inspect/controllers/logics/* → go-noah/internal/inspect/logics/ ❌ 缺失
goinsight/internal/inspect/controllers/traverses/* → go-noah/internal/inspect/traverses/ ❌ 缺失
goinsight/internal/inspect/controllers/process/* → go-noah/internal/inspect/process/ ❌ 缺失
goinsight/internal/inspect/controllers/extract/* → go-noah/internal/inspect/extract/ ❌ 缺失
```

**缺失统计**：
- AlterTable 规则：缺失 18/19 条
- CreateTable 规则：缺失 18/19 条
- DML 规则：缺失 8/8 条
- 其他规则：缺失 5/5 条

#### 3. 数据库执行器（直接复制）
```
goinsight/internal/das/dao/* → go-noah/pkg/insight/dao/
goinsight/internal/orders/api/mysql/* → go-noah/pkg/insight/executor/mysql/
goinsight/internal/orders/api/tidb/* → go-noah/pkg/insight/executor/tidb/
goinsight/internal/orders/api/base/* → go-noah/pkg/insight/executor/base/
goinsight/internal/orders/api/file/* → go-noah/pkg/insight/executor/file/
```

#### 4. 工具包（直接复制）
```
goinsight/pkg/kv/* → go-noah/pkg/kv/
goinsight/pkg/notifier/* → go-noah/pkg/notifier/
goinsight/pkg/pagination/* → go-noah/pkg/pagination/
goinsight/pkg/parser/* → go-noah/pkg/parser/
goinsight/pkg/query/* → go-noah/pkg/query/
goinsight/pkg/response/* → go-noah/pkg/response/
goinsight/pkg/utils/* → go-noah/pkg/utils/
```

### 需要重写/适配的文件

#### 1. Handler层（重写，适配go-noah架构）
```
goinsight/internal/*/views/* → go-noah/internal/handler/insight/
```

#### 2. Service层（重写，适配go-noah架构）
```
goinsight/internal/*/services/* → go-noah/internal/service/insight/
```

#### 3. Repository层（新建）
```
go-noah/internal/repository/insight/
├── environment.go
├── dbconfig.go
├── das.go
├── inspect.go
├── order.go
└── organization.go
```

## 六、数据库迁移

### 1. 表映射关系
| goInsight 表名 | go-noah 表名 |
|---------------|-------------|
| insight_users | admin_users (已存在，需扩展) |
| insight_roles | roles (已存在) |
| insight_organizations | organizations |
| insight_organizations_users | organization_users |
| insight_db_environments | db_environments |
| insight_db_config | db_configs |
| insight_db_schemas | db_schemas |
| insight_das_user_schema_permissions | das_user_schema_permissions |
| insight_das_user_table_permissions | das_user_table_permissions |
| insight_das_allowed_operations | das_allowed_operations |
| insight_das_records | das_records |
| insight_das_favorites | das_favorites |
| insight_inspect_params | inspect_params |
| insight_order_records | order_records |
| insight_order_tasks | order_tasks |
| insight_order_oplogs | order_op_logs |
| insight_order_messages | order_messages |

### 2. 数据迁移脚本
```sql
-- 迁移脚本示例（需要根据实际情况调整）
-- 用户数据迁移
INSERT INTO admin_users (username, nickname, password, email, phone, created_at, updated_at)
SELECT username, nick_name, password, email, mobile, date_joined, updated_at
FROM insight_users;

-- 其他表类似...
```

## 七、前端适配

### API路径变更
goInsight 的 API 路径格式为 `/api/v1/xxx`，go-noah 保持兼容。

### 需要注意的变更
1. 认证头格式：`Authorization: JWT <token>` → `Authorization: Bearer <token>`
2. 响应格式统一

## 八、执行顺序

1. **第一周：基础设施**
   - 配置文件合并
   - 数据库模型定义
   - 表结构迁移

2. **第二周：用户与组织**
   - 组织管理功能
   - 用户-组织关联

3. **第三周：通用模块**
   - 环境管理
   - 数据库配置管理

4. **第四周：DAS模块**
   - SQL执行器
   - 权限管理
   - 历史记录和收藏

5. **第五周：审核模块**
   - 审核规则迁移
   - 审核参数管理

6. **第六周：工单模块**
   - 工单CRUD
   - 任务执行
   - 消息通知

7. **第七周：定时任务与测试**
   - 定时任务迁移
   - 集成测试
   - 性能优化

## 九、注意事项

1. **包名修改**：所有 `goInsight` 的导入路径需改为 `go-noah`
2. **全局变量**：goInsight 使用 `global.App.*`，需改为 `global.*`
3. **错误处理**：统一使用 go-noah 的错误处理方式
4. **日志记录**：统一使用 zap logger
5. **权限控制**：所有新增路由需要在 Casbin 中配置相应策略
6. **密码加密**：数据库连接密码需使用加密存储

## 十、验收标准

1. 所有 goInsight 功能在 go-noah 中正常运行
2. API 响应格式一致
3. 权限控制正常工作
4. 定时任务正常执行
5. 数据完整性验证通过
6. 性能测试通过

