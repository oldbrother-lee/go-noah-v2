# Service 初始化方式对比

## 一、两种方式对比

### 1.1 gin-vue-admin 方式（全局变量）

```go
// service/user.go
type UserService struct{}

var UserServiceApp = new(UserService)  // Service 直接定义为全局变量

func (s *UserService) GetUser(id uint) (*User, error) {
    // 直接使用 global.DB（global 中只有基础设施，不包含 Service）
    var user User
    return &user, global.DB.First(&user, id).Error
}

// pkg/global/global.go（只包含基础设施）
var (
    DB     *gorm.DB      // 数据库
    Logger *log.Logger   // 日志
    Redis  *redis.Client // Redis
    // 注意：不包含 Service 或 Repository
)
```

**特点：**
- ✅ 简洁：Service 直接定义为全局变量，不需要 New 函数
- ✅ 直接使用：`UserServiceApp.GetUser(id)`
- ✅ global 包只包含基础设施（DB、Logger 等），不包含业务层
- ❌ Service 依赖全局变量（global.DB）
- ❌ 测试时需要设置全局变量

### 1.2 go-noah 方式（依赖注入）

```go
// service/admin.go
type AdminService struct {
    *Service
    adminRepository *repository.AdminRepository
}

func NewAdminService(
    service *Service,
    adminRepository *repository.AdminRepository,
) *AdminService {
    return &AdminService{
        Service:         service,
        adminRepository: adminRepository,
    }
}
```

**特点：**
- ✅ 依赖注入：依赖通过参数传入
- ✅ 便于测试：可以传入 mock 对象
- ✅ 无全局变量：更符合函数式编程
- ❌ 需要 New 函数：代码稍多

## 二、为什么 go-noah 要这样设计？

### 2.1 依赖注入的优势

1. **明确的依赖关系**
   ```go
   // 一眼就能看出依赖什么
   adminSvc := service.NewAdminService(svc, adminRepo)
   ```

2. **便于测试**
   ```go
   // 测试时可以传入 mock
   mockRepo := &MockAdminRepository{}
   service := NewAdminService(mockService, mockRepo)
   ```

3. **无全局状态**
   - 不依赖全局变量
   - 更符合函数式编程思想
   - 避免并发问题

### 2.2 gin-vue-admin 的方式

**gin-vue-admin 使用全局变量：**
```go
// global/global.go
var (
    DB     *gorm.DB
    Redis  *redis.Client
    // ...
)

// service/user.go
func (s *UserService) GetUser(id uint) (*User, error) {
    var user User
    return &user, global.DB.First(&user, id).Error
}
```

**优点：**
- ✅ 代码简洁
- ✅ 使用方便

**缺点：**
- ❌ 依赖全局变量
- ❌ 测试需要设置全局变量
- ❌ 可能有并发问题

## 三、简化方案

### 方案一：保持依赖注入（推荐）

**当前方式已经很好，保持即可：**
```go
type AdminService struct {
    *Service
    adminRepository *repository.AdminRepository
}

func NewAdminService(
    service *Service,
    adminRepository *repository.AdminRepository,
) *AdminService {
    return &AdminService{
        Service:         service,
        adminRepository: adminRepository,
    }
}
```

**优点：**
- ✅ 依赖明确
- ✅ 便于测试
- ✅ 无全局变量

### 方案二：改为 gin-vue-admin 风格（可选）

**如果要改成 gin-vue-admin 风格，需要：**

1. **创建 global 包（只定义基础设施）**
```go
// pkg/global/global.go
package global

import (
    "go-noah/pkg/jwt"
    "go-noah/pkg/log"
    "gorm.io/gorm"
)

var (
    DB     *gorm.DB
    Logger *log.Logger
    JWT    *jwt.JWT
)
```

2. **简化 Service（直接使用 global.DB，不需要 Repository 字段）**
```go
// service/admin.go
type AdminService struct{}

var AdminServiceApp = new(AdminService)

func (s *AdminService) GetAdminUser(ctx context.Context, uid uint) (*v1.GetAdminUserResponseData, error) {
    // 直接使用 global.DB，或者内部创建 Repository
    var user model.AdminUser
    err := global.DB.WithContext(ctx).Where("id = ?", uid).First(&user).Error
    // 或者
    // repo := repository.NewAdminRepository(repository.NewRepository(global.Logger, global.DB, e))
    // user, err := repo.GetAdminUser(ctx, uid)
    // ...
}
```

3. **初始化全局变量**
```go
// pkg/noah/noah.go
func NewServerApp(conf *viper.Viper, logger *log.Logger) (*app.App, func(), error) {
    // 初始化全局变量（只初始化基础设施）
    global.DB = repository.NewDB(conf, logger)
    global.Logger = logger
    global.JWT = jwt.NewJwt(conf)
    
    // Service 不需要初始化（使用全局变量）
    // adminSvc := service.NewAdminService(...) // 不需要了
    
    // Handler 直接使用全局 Service
    adminHandler := handler.NewAdminHandler(h, service.AdminServiceApp)
    // ...
}
```

**关键点：** 
- ✅ **global 包只包含基础设施组件**（DB、Logger、Redis 等），**不包含 Service 或 Repository**
- ✅ **Service 直接定义为全局变量**（在 service 包中：`var UserServiceApp = new(UserService)`），**不在 global 中注册**
- ✅ **Service 内部使用 global.DB** 等基础设施组件访问数据库，**不通过 Repository 层**

**实际 gin-vue-admin 代码结构：**
```go
// pkg/global/global.go（只包含基础设施组件）
var (
    DB     *gorm.DB      // 数据库连接
    Logger *log.Logger   // 日志组件
    Redis  *redis.Client // Redis 连接
    // 注意：global 中不包含 Service 或 Repository
)

// service/user.go（Service 定义为全局变量，不在 global 中）
type UserService struct{}

var UserServiceApp = new(UserService)  // 在 service 包中定义

func (s *UserService) GetUser(id uint) (*User, error) {
    var user User
    // Service 内部使用 global.DB（基础设施组件）
    return &user, global.DB.First(&user, id).Error
}
```

**goInsight 也是类似方式：**
```go
// global/app.go（只包含基础设施组件）
var App = new(Application)  // Application 包含 DB、Redis、Logger 等

// services/users.go（Service 不在 global 中，直接使用全局变量）
type GetUsersServices struct {
    *forms.GetUsersForm
    C *gin.Context
}

func (s *GetUsersServices) Run() {
    // Service 内部使用 global.App.DB（基础设施组件）
    tx := global.App.DB.Table("insight_users a")...
}
```

## 四、推荐方案

### 4.1 保持当前方式（推荐）

**理由：**
1. ✅ **依赖注入更清晰**：一眼看出依赖关系
2. ✅ **便于测试**：可以传入 mock
3. ✅ **无全局变量**：避免并发问题
4. ✅ **符合现代 Go 实践**：依赖注入是推荐方式

**当前代码已经很简洁了：**
```go
// 只需要 10 行代码
type AdminService struct {
    *Service
    adminRepository *repository.AdminRepository
}

func NewAdminService(...) *AdminService {
    return &AdminService{...}
}
```

### 4.2 如果一定要简化

**可以去掉 New 函数，但需要全局变量：**

```go
// service/admin.go
type AdminService struct {
    // 依赖通过方法内部获取，而不是字段
}

var AdminServiceApp = new(AdminService)

func (s *AdminService) GetAdminUser(ctx context.Context, uid uint) (*v1.GetAdminUserResponseData, error) {
    // 需要从 global 获取依赖
    repo := global.AdminRepository
    user, err := repo.GetAdminUser(ctx, uid)
    // ...
}
```

**但这样会：**
- ❌ 引入全局变量
- ❌ 依赖关系不明确
- ❌ 测试更困难

## 五、总结

### 5.1 两种方式对比

| 维度 | gin-vue-admin（全局变量） | go-noah（依赖注入） |
|------|-------------------------|-------------------|
| **代码量** | 更少（2行） | 稍多（10行） |
| **依赖关系** | 不明确（隐藏在 global） | 明确（参数传入） |
| **测试** | 需要设置全局变量 | 可以传入 mock |
| **并发安全** | 可能有问题 | 更安全 |
| **推荐度** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |

### 5.2 建议

**保持当前的依赖注入方式：**
- ✅ 代码已经很简洁（只需 10 行）
- ✅ 依赖关系清晰
- ✅ 便于测试和维护
- ✅ 符合现代 Go 最佳实践

**如果觉得 New 函数麻烦，可以：**
- 使用代码生成工具
- 或者接受这 10 行代码（相比全局变量的缺点，这点代码量是值得的）

## 六、实际使用对比

### gin-vue-admin 方式
```go
// Service 定义（在 service/user.go 中）
type UserService struct{}

var UserServiceApp = new(UserService)  // 直接定义为全局变量

func (s *UserService) GetUser(id uint) (*User, error) {
    // 直接使用 global.DB（global 中只有基础设施）
    var user User
    return &user, global.DB.First(&user, id).Error
}

// global 包中只包含基础设施（不包含 Service）
// pkg/global/global.go
var (
    DB     *gorm.DB      // 数据库
    Logger *log.Logger   // 日志
    Redis  *redis.Client // Redis
    // 不包含 Service 或 Repository
)

// 使用
UserServiceApp.GetUser(id)  // 直接调用全局 Service

// 测试时需要设置 global.DB
```

### go-noah 方式
```go
// 使用（在 noah.go 中初始化一次）
adminSvc := service.NewAdminService(svc, adminRepo)
adminHandler := handler.NewAdminHandler(h, adminSvc)

// 依赖明确，测试时可以传入 mock
```

**结论：** 虽然 go-noah 方式代码稍多，但依赖关系更清晰，更便于测试和维护。这 10 行代码是值得的。

