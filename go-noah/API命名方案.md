# API 命名方案

## 当前结构
```
api/
└── v1/
    ├── v1.go
    ├── admin.go
    └── errors.go
```

## 命名方案

### 方案一：直接放在 api 根目录（推荐）
**优点：** 简洁，适合单版本 API
**缺点：** 如果未来需要多版本，需要重构

```
api/
├── api.go          # 原 v1.go，重命名为 api.go
├── admin.go
└── errors.go
```

**包名：** `package api`
**导入：** `import "go-noah/api"`
**使用：** `api.HandleSuccess(ctx, data)`

### 方案二：使用语义化名称
**优点：** 语义清晰，便于理解
**缺点：** 如果未来需要多版本，需要重新命名

```
api/
└── stable/         # 或 current, latest
    ├── stable.go
    ├── admin.go
    └── errors.go
```

**包名：** `package stable`
**导入：** `import "go-noah/api/stable"`
**使用：** `stable.HandleSuccess(ctx, data)`

### 方案三：使用业务相关名称
**优点：** 符合业务语义
**缺点：** 不够通用

```
api/
└── rest/           # RESTful API
    ├── rest.go
    ├── admin.go
    └── errors.go
```

**包名：** `package rest`
**导入：** `import "go-noah/api/rest"`
**使用：** `rest.HandleSuccess(ctx, data)`

### 方案四：使用日期版本
**优点：** 版本清晰，便于管理
**缺点：** 命名较长

```
api/
└── 2024/           # 或 2024-01
    ├── api.go
    ├── admin.go
    └── errors.go
```

**包名：** `package api2024`
**导入：** `import "go-noah/api/2024"`
**使用：** `api2024.HandleSuccess(ctx, data)`

### 方案五：使用语义版本（不带 v 前缀）
**优点：** 版本号清晰
**缺点：** 仍然是版本号

```
api/
└── 1/              # 版本 1
    ├── api.go
    ├── admin.go
    └── errors.go
```

**包名：** `package api1`
**导入：** `import "go-noah/api/1"`
**使用：** `api1.HandleSuccess(ctx, data)`

## 推荐方案

### 推荐：方案一（直接放在 api 根目录）

**理由：**
1. ✅ 最简洁，代码最少
2. ✅ 适合单版本 API（大多数项目都是这样）
3. ✅ 包名 `api` 清晰明了
4. ✅ 如果未来需要多版本，可以再迁移到 `api/v1/` 等

**迁移步骤：**
1. 将 `api/v1/` 目录下的文件移到 `api/` 根目录
2. 修改包名从 `package v1` 改为 `package api`
3. 全局替换 `v1.` 为 `api.`
4. 全局替换 `"go-noah/api/v1"` 为 `"go-noah/api"`

## 各方案对比

| 方案 | 包名 | 导入路径 | 使用示例 | 复杂度 | 推荐度 |
|------|------|---------|---------|--------|--------|
| 方案一 | `api` | `go-noah/api` | `api.HandleSuccess()` | ⭐ | ⭐⭐⭐⭐⭐ |
| 方案二 | `stable` | `go-noah/api/stable` | `stable.HandleSuccess()` | ⭐⭐ | ⭐⭐⭐ |
| 方案三 | `rest` | `go-noah/api/rest` | `rest.HandleSuccess()` | ⭐⭐ | ⭐⭐⭐ |
| 方案四 | `api2024` | `go-noah/api/2024` | `api2024.HandleSuccess()` | ⭐⭐⭐ | ⭐⭐ |
| 方案五 | `api1` | `go-noah/api/1` | `api1.HandleSuccess()` | ⭐⭐ | ⭐⭐ |

## 注意事项

1. **包名不能是数字开头**：如果使用方案四或方案五，包名需要特殊处理（如 `api2024`、`api1`）
2. **Go 模块路径**：确保导入路径符合 Go 模块规范
3. **全局替换**：修改后需要全局替换所有引用

