# inspect_params 数据迁移方案

## 问题描述

`inspect_params` 数据为空，从老项目迁移过来，老项目启动时应该自动初始化这些数据，新项目需要实现相同的功能。

## 现状分析

### 老项目（goinsight）的实现方式

**文件**: `goinsight/bootstrap/db.go`

**实现逻辑**:
1. 在 `initializeMySQLGorm()` 中，启动时调用 `initializeInspectParams(db)`
2. `initializeInspectParams()` 函数：
   - 定义了 80+ 条默认审核参数配置
   - 使用 `FirstOrCreate` 确保数据只初始化一次（不会重复创建）
   - 每条记录包含 `params` (JSON) 和 `remark` (备注)

**表结构**: `insight_inspect_params`（表名带 `insight_` 前缀）
```go
type InsightInspectParams struct {
    *models.Model  // ID, CreatedAt, UpdatedAt
    Params datatypes.JSON  // 语法审核参数（JSON格式）
    Remark string          // 备注
}
```

**数据格式**:
- 每条记录代表一个审核参数配置项
- `Params` 字段存储 JSON，格式如：`{"MAX_TABLE_NAME_LENGTH": 32}`
- `Remark` 字段存储备注，如：`"表名的长度"`

### 新项目（go-noah）的现状

**文件**: `go-noah/internal/server/migration.go`

**现状**:
- ✅ 已定义 `InspectParams` 模型（`go-noah/internal/model/insight/inspect.go`）
- ✅ 已在 `AutoMigrate` 中包含 `&insight.InspectParams{}`
- ❌ **缺少** `initialInspectParams()` 方法
- ❌ **未调用** 初始化方法

**表结构**: `inspect_params`（注意：表名**不带** `insight_` 前缀，与老项目不同）
```go
type InspectParams struct {
    gorm.Model  // ID, CreatedAt, UpdatedAt, DeletedAt
    Params datatypes.JSON  // 语法审核参数（JSON格式）
    Remark string          // 备注
}
```

## 迁移方案

### 方案1：完全迁移（推荐）

**实现步骤**:

1. **在 `go-noah/internal/server/migration.go` 中添加 `initialInspectParams()` 方法**
   - 复制老项目的所有默认参数配置（80+ 条）
   - 使用 `FirstOrCreate` 确保幂等性（不会重复创建）
   - 以 `Remark` 作为唯一标识进行查找（因为老项目没有 unique 约束）

2. **在 `MigrateServer.Start()` 中调用初始化方法**
   - 在 `AutoMigrate` 之后调用
   - 在 `initialAdminUser` 之前或之后都可以（无依赖关系）

3. **数据迁移（如果需要从老数据库迁移）**
   - 如果老数据库已有数据，可以：
     - 方案A：直接 SQL 迁移（推荐，快速）
     - 方案B：通过代码迁移（更安全，可以转换数据格式）

### 方案2：SQL 直接迁移

**适用场景**: 老数据库已有数据，需要迁移到新数据库

**实现步骤**:
1. 从老数据库导出 `insight_inspect_params` 表数据
2. 导入到新数据库
3. 仍然需要实现初始化方法，用于新环境部署

### 方案3：混合方案（推荐用于生产）

**实现步骤**:
1. 实现初始化方法（用于新环境）
2. 提供 SQL 迁移脚本（用于从老环境迁移数据）
3. 在迁移脚本中检查数据是否存在，不存在则初始化

## 推荐实现方案（方案1 + 方案3）

### 步骤1：实现初始化方法

**文件**: `go-noah/internal/server/migration.go`

**添加方法**:
```go
func (m *MigrateServer) initialInspectParams(ctx context.Context) error {
    var params []map[string]interface{} = []map[string]interface{}{
        // TABLE
        {"params": map[string]int{"MAX_TABLE_NAME_LENGTH": 32}, "remark": "表名的长度"},
        {"params": map[string]bool{"CHECK_TABLE_COMMENT": true}, "remark": "检查表是否有注释"},
        // ... 复制所有老项目的参数配置
    }
    
    for _, i := range params {
        var inspectParams insight.InspectParams
        jsonParams, err := json.Marshal(i["params"])
        if err != nil {
            m.log.Error("marshal inspect params failed", zap.Error(err))
            return err
        }
        
        // 使用 Remark 作为唯一标识查找
        result := m.db.Where("remark = ?", i["remark"].(string)).First(&inspectParams)
        if result.Error != nil {
            if result.Error == gorm.ErrRecordNotFound {
                // 记录不存在，创建新记录
                if err := m.db.Create(&insight.InspectParams{
                    Params: jsonParams,
                    Remark: i["remark"].(string),
                }).Error; err != nil {
                    m.log.Error("create inspect params failed", 
                        zap.String("remark", i["remark"].(string)),
                        zap.Error(err))
                    return err
                }
            } else {
                return result.Error
            }
        } else {
            // 记录已存在，跳过（幂等性）
            m.log.Debug("inspect params already exists", 
                zap.String("remark", i["remark"].(string)))
        }
    }
    
    m.log.Info("initialInspectParams success")
    return nil
}
```

**在 `Start()` 方法中调用**:
```go
// 在 initialFlowDefinitions 之后添加
err = m.initialInspectParams(ctx)
if err != nil {
    m.log.Error("initialInspectParams error", zap.Error(err))
}
```

### 步骤2：提供 SQL 迁移脚本（可选）

**文件**: `go-noah/scripts/migrate_inspect_params.sql`

**内容**:
```sql
-- 从老数据库迁移 inspect_params 数据
-- 使用方式：
-- 1. 从老数据库导出：SELECT * FROM insight_inspect_params;
-- 2. 在新数据库执行 INSERT 语句（跳过已存在的记录）

-- 示例：使用 INSERT IGNORE 或 ON DUPLICATE KEY UPDATE
-- 注意：新项目表名是 inspect_params（不带 insight_ 前缀），老项目表名是 insight_inspect_params
INSERT IGNORE INTO inspect_params (params, remark, created_at, updated_at)
SELECT params, remark, created_at, updated_at
FROM goinsight_db.insight_inspect_params;
```

### 步骤3：验证数据

**验证方法**:
1. 运行迁移：`go run cmd/migration/main.go`
2. 检查数据库：`SELECT COUNT(*) FROM inspect_params;` （应该 >= 80）
3. 检查日志：确认没有错误信息

## 注意事项

1. **幂等性**：使用 `FirstOrCreate` 或 `Where().First()` + `Create()` 确保不会重复创建
2. **数据格式**：确保 JSON 格式与老项目一致
3. **字符集格式**：特别注意 `TABLE_SUPPORT_CHARSET` 的格式必须是 `{"charset": "...", "recommend": "..."}`
4. **向后兼容**：如果老数据库已有自定义数据，迁移时需要保留
5. **表名差异**：
   - 老项目：`insight_inspect_params`（带 `insight_` 前缀）
   - 新项目：`inspect_params`（不带 `insight_` 前缀）
   - SQL 迁移时需要注意表名差异

## 实施计划

1. ✅ 创建迁移方案文档（本文档）
2. ⏳ 实现 `initialInspectParams()` 方法
3. ⏳ 在 `MigrateServer.Start()` 中调用
4. ⏳ 测试验证（新环境初始化）
5. ⏳ （可选）提供 SQL 迁移脚本（老环境迁移）

## 参考文件

- 老项目初始化逻辑：`goinsight/bootstrap/db.go` (第 217-316 行)
- 新项目迁移逻辑：`go-noah/internal/server/migration.go`
- 模型定义：`go-noah/internal/model/insight/inspect.go`

