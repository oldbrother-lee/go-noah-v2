# API 命名改造完成（方案一）

## 改造内容

按照方案一（直接放在 api 根目录）完成了 API 命名改造。

## 完成的变更

### 1. 文件移动和重命名

- ✅ 将 `api/v1/v1.go` 移动到 `api/api.go`
- ✅ 将 `api/v1/admin.go` 移动到 `api/admin.go`
- ✅ 将 `api/v1/errors.go` 移动到 `api/errors.go`
- ✅ 删除 `api/v1/` 目录

### 2. 包名修改

- ✅ `package v1` → `package api`
- ✅ 所有文件包名已统一为 `package api`

### 3. 导入路径修改

- ✅ `"go-noah/api/v1"` → `"go-noah/api"`
- ✅ 移除了所有别名导入（`v1 "go-noah/api"` → `"go-noah/api"`）

### 4. 代码引用修改

- ✅ `v1.HandleSuccess()` → `api.HandleSuccess()`
- ✅ `v1.HandleError()` → `api.HandleError()`
- ✅ `v1.ErrXXX` → `api.ErrXXX`
- ✅ `v1.XXXRequest` → `api.XXXRequest`
- ✅ `v1.XXXResponse` → `api.XXXResponse`

### 5. 变量名冲突修复

- ✅ 修复了 `service/admin.go` 中循环变量 `api` 与包名冲突的问题
- ✅ 将循环变量改为 `item`

## 修改的文件清单

### API 包文件
- ✅ `api/api.go` - 新建（原 `api/v1/v1.go`）
- ✅ `api/admin.go` - 新建（原 `api/v1/admin.go`）
- ✅ `api/errors.go` - 新建（原 `api/v1/errors.go`）

### 引用 API 包的文件
- ✅ `internal/handler/admin.go`
- ✅ `internal/service/admin.go`
- ✅ `internal/repository/admin.go`
- ✅ `internal/router/router.go`
- ✅ `internal/middleware/rbac.go`
- ✅ `internal/middleware/jwt.go`
- ✅ `internal/middleware/sign.go`
- ✅ `internal/server/migration.go`

## 最终结构

```
api/
├── api.go          # 响应处理和错误定义
├── admin.go        # 管理员相关请求/响应结构体
└── errors.go       # 错误定义
```

## 使用示例

### 导入
```go
import "go-noah/api"
```

### 使用
```go
// 成功响应
api.HandleSuccess(ctx, data)

// 错误响应
api.HandleError(ctx, http.StatusBadRequest, api.ErrBadRequest, nil)

// 请求结构体
var req api.LoginRequest

// 响应结构体
resp := api.LoginResponseData{...}
```

## 注意事项

1. **路由路径保持不变**：路由路径 `/v1/...` 保持不变，因为路由路径和包名可以不同
2. **向后兼容**：API 路径保持不变，不影响前端调用
3. **包名清晰**：`api` 包名更简洁明了

## 完成状态

✅ 所有改造已完成，代码已通过 linter 检查，无语法错误。

