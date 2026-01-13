# SQL审核模块完整迁移清单

## ⚠️ 当前迁移状态

**严重问题**：当前迁移只实现了基础的 SQL 审核框架，缺少了**完整的规则引擎架构**和**大部分审核规则**。

## 📊 缺失统计

### 1. 规则定义缺失

#### AlterTable 规则（缺失 18/19 条）
- ✅ 检查表是否存在
- ❌ 检查TiDBMergeAlter
- ❌ DROP列和索引检查
- ❌ DropTiDBColWithCoveredIndex检查
- ❌ 表Options检查
- ❌ 列字符集检查
- ❌ Add列After检查
- ❌ Add列Options检查
- ❌ Add主键检查
- ❌ Add重复列检查
- ❌ Add索引前缀检查
- ❌ Add索引数量检查
- ❌ AddConstraint检查
- ❌ Add重复索引检查
- ❌ Add冗余索引检查
- ❌ BLOB/TEXT类型不能设置为索引
- ❌ Modify列Options检查
- ❌ Change列Options检查
- ❌ RenameIndex检查
- ❌ RenameTblName检查
- ❌ 索引InnodbLargePrefix
- ❌ 检查表定义的行是否超过65535

#### CreateTable 规则（缺失 18/19 条）
- ✅ 检查表是否存在（部分实现）
- ❌ 检查CreateTableAs语法
- ❌ 检查CreateTableLike语法
- ❌ 表Options检查
- ❌ 主键检查
- ❌ 约束检查
- ❌ 审计字段检查
- ❌ 列Options检查
- ❌ 列重复定义检查
- ❌ 列字符集检查
- ❌ 索引前缀检查
- ❌ 索引数量检查
- ❌ 索引重复定义检查
- ❌ 冗余索引检查
- ❌ BLOB/TEXT类型不能设置为索引
- ❌ 索引InnodbLargePrefix
- ❌ 检查InnoDB表定义的RowSize
- ❌ 检查InnoDB表RowFormat

#### DML 规则（缺失 8/8 条）
- ❌ 限制部分表进行语法审核
- ❌ 是否允许INSERT INTO SELECT语法
- ❌ 必须要有WHERE条件
- ❌ INSERT必须指定列名
- ❌ 不能有LIMIT/ORDERBY/SubQuery
- ❌ JOIN操作必须要有ON语句
- ❌ 更新影响行数
- ❌ 插入影响行数

#### 其他规则（缺失 5/5 条）
- ❌ DropTable检查
- ❌ TruncateTable检查
- ❌ RenameTable检查
- ❌ CreateView检查
- ❌ CreateDatabase检查
- ❌ AnalyzeTable检查

### 2. 架构组件缺失

#### 规则引擎架构
- ❌ `rules/` 目录：规则定义（10个文件）
  - alter.go
  - create.go
  - dml.go
  - drop.go
  - rename.go
  - view.go
  - database.go
  - analyze.go
  - rule.go（规则基础结构）

#### 逻辑实现
- ❌ `logics/` 目录：规则逻辑实现（8个文件）
  - alter.go（约668行）
  - create.go
  - dml.go
  - drop.go
  - rename.go
  - create_view.go
  - database.go
  - analyze.go

#### 语法树遍历器
- ❌ `traverses/` 目录：语法树遍历器（8个文件）
  - alter.go（约854行）
  - create.go
  - dml.go
  - drop.go
  - rename.go
  - create_view.go
  - database.go
  - analyze.go

#### 辅助工具
- ❌ `process/` 目录：处理辅助结构（多个文件）
- ❌ `extract/` 目录：表名提取等功能
- ❌ `kv/` 包：缓存工具
- ❌ `query/` 包：SQL指纹等工具

### 3. 配置文件缺失

- ❌ 完整的 `InspectParams` 配置（当前只有部分参数）
- ❌ 默认配置值初始化

## 📋 完整迁移计划

### 阶段一：架构迁移（必须）

1. **复制规则引擎基础结构**
   ```bash
   # 复制规则引擎基础文件
   goinsight/internal/inspect/controllers/rules/rule.go → go-noah/internal/inspect/rules/rule.go
   ```

2. **复制规则定义文件**（10个文件）
   ```
   goinsight/internal/inspect/controllers/rules/* → go-noah/internal/inspect/rules/
   ```

3. **复制逻辑实现文件**（8个文件）
   ```
   goinsight/internal/inspect/controllers/logics/* → go-noah/internal/inspect/logics/
   ```

4. **复制遍历器文件**（8个文件）
   ```
   goinsight/internal/inspect/controllers/traverses/* → go-noah/internal/inspect/traverses/
   ```

5. **复制辅助工具**
   ```
   goinsight/internal/inspect/controllers/process/* → go-noah/internal/inspect/process/
   goinsight/internal/inspect/controllers/extract/* → go-noah/internal/inspect/extract/
   goinsight/pkg/kv/* → go-noah/pkg/kv/
   goinsight/pkg/query/* → go-noah/pkg/query/
   ```

### 阶段二：适配改造（重要）

1. **修改导入路径**
   - 将所有 `goInsight/` 路径改为 `go-noah/`

2. **适配数据库连接**
   - 将 `dao.DB` 的创建方式适配到新框架
   - 确保 `RuleHint.DB` 正确初始化

3. **适配配置结构**
   - 将 `config.InspectParams` 完整迁移
   - 适配默认参数初始化

4. **适配上下文传递**
   - 将 `gin.Context` 改为新框架的上下文方式
   - 确保 `RuleHint` 正确初始化

### 阶段三：重构 checker.go（必须）

1. **替换简单实现**
   - 删除当前的简化版 `checker.go`
   - 使用规则引擎架构重构 `Checker`

2. **实现规则调用**
   - 按照老代码的方式调用各个规则
   - 确保规则链正确执行

3. **适配返回格式**
   - 将规则引擎的 `ReturnData` 适配为新框架的 `AuditResult`

### 阶段四：测试验证（必须）

1. **单元测试**
   - 为每个规则编写单元测试
   - 验证规则逻辑正确性

2. **集成测试**
   - 测试完整审核流程
   - 验证所有规则都能正确执行

3. **回归测试**
   - 对比老代码和新代码的审核结果
   - 确保一致性

## ⚠️ 重要说明

### 为什么不能"直接复制"

1. **路径差异**：`goInsight` → `go-noah`
2. **架构差异**：需要适配新框架的上下文、依赖注入等
3. **配置差异**：需要适配新框架的配置系统
4. **返回格式差异**：需要适配新框架的响应格式

### 迁移原则

1. **完整性优先**：所有规则都必须迁移，不能遗漏
2. **一致性保证**：迁移后的审核结果必须与老代码一致
3. **可测试性**：每个规则都要有测试用例
4. **可维护性**：保持规则引擎的可扩展性

## 📝 下一步行动

1. **立即停止**在简化版 `checker.go` 上继续开发
2. **制定详细**的迁移计划和时间表
3. **开始迁移**规则引擎基础架构
4. **逐个迁移**所有规则和逻辑
5. **完整测试**确保功能完整性

## 📊 工作量估算

- 架构迁移：2-3天
- 规则迁移：5-7天
- 适配改造：3-4天
- 测试验证：3-4天
- **总计：13-18天**

