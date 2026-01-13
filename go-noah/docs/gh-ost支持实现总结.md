# gh-ost 支持实现总结

## 一、实现概述

已成功实现 gh-ost 在线 DDL 支持，新系统现在与老系统保持一致，ALTER TABLE 语句使用 gh-ost 在线执行，避免锁表。

## 二、实现内容

### 2.1 配置文件支持

**文件：`go-noah/config/local.yml`**

```yaml
# gh-ost 配置（用于在线 DDL 变更）
ghost:
  path: "/Users/lee/Downloads/gh-ost"  # gh-ost 工具路径
  args:                                 # gh-ost 参数列表
    - "--allow-on-master"
    - "--assume-rbr"
    - "--initially-drop-ghost-table"
    - "--initially-drop-old-table"
    - "-initially-drop-socket-file"
```

### 2.2 SQL 解析功能

**文件：`go-noah/internal/inspect/parser/parser.go`**

新增函数：
- `GetSqlStatement(sqltext string) (string, error)` - 获取 SQL 语句类型
  - 返回：`AlterTable`, `CreateDatabase`, `CreateTable`, `CreateView`, `DropTable`, `DropIndex`, `TruncateTable`, `RenameTable`, `CreateIndex`, `DropDatabase`
  
- `GetTableNameFromAlterStatement(sqltext string) (string, error)` - 从 ALTER TABLE 语句中提取表名
  - 支持 `schema.table` 格式
  - 支持不带 schema 的表名

### 2.3 命令执行工具

**文件：`go-noah/pkg/utils/command.go`**

新增函数：
- `Command(ctx context.Context, ch chan<- string, cmd string) error` - 执行系统命令并实时推送输出
  - 支持实时输出到 channel
  - 支持 stdout 和 stderr 读取
  - 支持 context 取消
  - 返回退出状态码

### 2.4 DDL 执行逻辑

**文件：`go-noah/internal/orders/executor/mysql.go`**

#### 修改内容：

1. **`ExecuteDDL()` 方法重构**
   - 根据 SQL 类型选择执行方式
   - `AlterTable` → 使用 `ExecuteDDLWithGhost()`（gh-ost）
   - 其他 DDL → 使用 `ExecuteOnlineDDL()`（直接执行）
   - 特殊处理：`RenameTable`、`CreateIndex`、`DropDatabase` 返回错误提示

2. **新增 `ExecuteOnlineDDL()` 方法**
   - 直接执行 DDL（CREATE/DROP/TRUNCATE 等）
   - 连接数据库
   - 获取 Connection ID
   - 执行 SQL 并记录日志
   - 实时推送日志到 WebSocket

3. **新增 `ExecuteDDLWithGhost()` 方法**
   - 解析 ALTER TABLE 语句，提取表名和 ALTER 子句
   - 生成 gh-ost 命令
   - 执行 gh-ost 命令并实时推送日志
   - 支持阿里云 RDS 特殊参数
   - 处理超时（最多 2 小时）

### 2.5 表名处理逻辑

**支持情况：**

1. **带 schema 的表名**：`schema.table` → 分离为 `--database=schema --table=table`
2. **不带 schema 的表名**：`table` → 使用工单的 `Schema` 作为 database
3. **反引号处理**：自动去除反引号

### 2.6 gh-ost 命令生成

**命令格式：**

```bash
gh-ost [args] \
  --user="username" --password="password" \
  --host="hostname" --port=3306 \
  --database=schema --table=table \
  --alter="ALTER子句" --execute \
  [-aliyun-rds=true] \
  [-assume-master-host="hostname:port"]
```

**参数说明：**
- `path`: gh-ost 工具路径
- `args`: gh-ost 参数列表（从配置文件读取）
- `--database`: 数据库名（自动处理 schema.table 格式）
- `--table`: 表名
- `--alter`: ALTER 子句（从 SQL 中提取）
- `--execute`: 执行变更（测试模式不使用此参数）
- `-aliyun-rds=true`: 阿里云 RDS 支持（自动检测）
- `-assume-master-host`: 阿里云 RDS 主库地址

## 三、执行流程

### 3.1 ALTER TABLE 执行流程

```
1. 解析 SQL 类型（GetSqlStatement）
   └─ 判断为 "AlterTable"

2. 调用 ExecuteDDLWithGhost()
   ├─ 检查 gh-ost 配置
   ├─ 提取表名（GetTableNameFromAlterStatement）
   │  └─ 处理 schema.table 格式
   ├─ 正则匹配提取 ALTER 子句
   ├─ 生成 gh-ost 命令
   │  ├─ 基础参数（path, args, user, password, host, port）
   │  ├─ 数据库参数（database, table）
   │  ├─ ALTER 子句（--alter="..." --execute）
   │  └─ 阿里云 RDS 参数（如果检测到）
   ├─ 执行 gh-ost 命令
   │  ├─ 启动命令（bash -c）
   │  ├─ 实时读取 stdout/stderr
   │  └─ 推送日志到 WebSocket
   └─ 返回执行结果
```

### 3.2 其他 DDL 执行流程

```
1. 解析 SQL 类型（GetSqlStatement）
   └─ 判断为 CreateTable/DropTable/TruncateTable 等

2. 调用 ExecuteOnlineDDL()
   ├─ 连接数据库
   ├─ 获取 Connection ID
   ├─ 执行 SQL（db.ExecContext）
   └─ 返回执行结果
```

## 四、执行方式对比

### 4.1 ALTER TABLE 语句

| 特性 | 老系统（goInsight） | 新系统（go-noah） |
|------|---------------------|-------------------|
| 执行方式 | gh-ost 在线执行 | ✅ gh-ost 在线执行 |
| 锁表情况 | 不锁表 | ✅ 不锁表 |
| 日志推送 | 实时推送 | ✅ 实时推送 |
| 进度统计 | 无 | ✅ 已实现 |
| 超时处理 | 无限制 | ✅ 无限制（与老系统一致） |
| 阿里云 RDS | 支持 | ✅ 支持 |

### 4.2 其他 DDL 语句

| 特性 | 老系统（goInsight） | 新系统（go-noah） |
|------|---------------------|-------------------|
| 执行方式 | 直接执行 | ✅ 直接执行 |
| 锁表情况 | 可能锁表 | ✅ 可能锁表 |
| 日志推送 | 实时推送 | ✅ 实时推送 |
| 监控 | SHOW PROCESSLIST | ✅ 已实现 |

## 五、配置说明

### 5.1 必需配置

1. **gh-ost 工具路径**（`ghost.path`）
   - 必须是可执行文件的绝对路径
   - 示例：`/Users/lee/Downloads/gh-ost`
   - 需要确保有执行权限

2. **gh-ost 参数**（`ghost.args`）
   - 常用参数：
     - `--allow-on-master`: 允许在主库上执行
     - `--assume-rbr`: 假设使用 RBR（Row-Based Replication）
     - `--initially-drop-ghost-table`: 如果 ghost 表已存在，先删除
     - `--initially-drop-old-table`: 如果旧表已存在，先删除
     - `-initially-drop-socket-file`: 如果 socket 文件已存在，先删除

### 5.2 可选配置

- 阿里云 RDS 特殊参数（自动检测）：
  - `-aliyun-rds=true`: 启用阿里云 RDS 支持
  - `-assume-master-host`: 主库地址

## 六、使用示例

### 6.1 修改表结构（使用 gh-ost）

**SQL：**
```sql
ALTER TABLE test_table MODIFY COLUMN username VARCHAR(50) NOT NULL DEFAULT '' COMMENT '用户名';
```

**执行流程：**
1. 系统识别为 `AlterTable` 类型
2. 提取表名：`test_table`
3. 提取 ALTER 子句：`MODIFY COLUMN username VARCHAR(50) NOT NULL DEFAULT '' COMMENT '用户名'`
4. 生成 gh-ost 命令：
   ```bash
   /Users/lee/Downloads/gh-ost --allow-on-master --assume-rbr ... \
     --user="root" --password="***" \
     --host="127.0.0.1" --port=3306 \
     --database=test_db --table=test_table \
     --alter="MODIFY COLUMN username VARCHAR(50) NOT NULL DEFAULT '' COMMENT '用户名'" \
     --execute
   ```
5. 执行命令并实时推送日志

### 6.2 创建表（直接执行）

**SQL：**
```sql
CREATE TABLE test_table2 (id INT PRIMARY KEY);
```

**执行流程：**
1. 系统识别为 `CreateTable` 类型
2. 直接执行 SQL
3. 记录日志

## 七、注意事项

### 7.1 gh-ost 工具要求

1. **必须安装 gh-ost 工具**
   - 下载地址：https://github.com/github/gh-ost/releases
   - 确保有执行权限
   - 配置正确的路径

2. **权限要求**
   - gh-ost 需要有数据库的读写权限
   - 需要能够创建临时表
   - 需要能够修改表结构

### 7.2 阿里云 RDS 特殊处理

- 自动检测 `rds.aliyuncs.com` 域名
- 自动添加 `-aliyun-rds=true` 参数
- 自动添加 `-assume-master-host` 参数

### 7.3 表名处理

- 支持 `schema.table` 格式
- 自动处理反引号
- 如果 SQL 中未指定 schema，使用工单的 `Schema` 字段

### 7.4 超时处理

**老系统：** 没有设置超时时间，使用 `context.WithCancel`，命令会一直执行直到完成或被手动取消。

**新系统：** 已修复，与老系统保持一致：
- ✅ 使用 `context.WithCancel`，**没有超时限制**
- ✅ 命令会一直执行直到完成（适用于改表等长时间运行的任务）
- ✅ 不会强制终止 gh-ost 进程，确保数据一致性

**修复说明：**
- 移除了 `context.WithTimeout(2*time.Hour)` 的超时限制
- 改为使用 `context.WithCancel`，与老系统完全一致
- 对于改表系统，不应该设置超时时间，因为 gh-ost 可能需要很长时间来完成大表的在线变更

## 八、测试建议

### 8.1 功能测试

1. **ALTER TABLE 语句**
   - 测试单列修改
   - 测试多列修改
   - 测试添加/删除列
   - 测试添加/删除索引
   - 测试表选项修改

2. **其他 DDL 语句**
   - CREATE TABLE
   - DROP TABLE
   - TRUNCATE TABLE

3. **边界情况**
   - 带 schema 的表名
   - 不带 schema 的表名
   - 阿里云 RDS
   - 超时情况

### 8.2 性能测试

1. **大表 ALTER**
   - 测试大表（百万级数据）的 ALTER 操作
   - 验证不锁表特性

2. **并发执行**
   - 测试多个 ALTER 语句并发执行
   - 验证日志推送正确性

## 九、SHOW PROCESSLIST 监控功能（已实现）

### 9.1 功能说明

已实现 `SHOW PROCESSLIST` 监控功能，与老系统保持一致。该功能用于实时监控直接执行的 DDL/DML SQL 语句的执行状态。

**注意**：对于使用 `gh-ost` 的 ALTER TABLE 语句，不需要此监控（gh-ost 有自己的输出）。

### 9.2 实现方式

**文件：`go-noah/internal/orders/executor/mysql.go`**

1. **新增 `GetProcesslist` 方法**
   - 在单独的 goroutine 中运行
   - 使用独立的数据库连接（避免影响主连接）
   - 每 500ms 查询一次 `INFORMATION_SCHEMA.PROCESSLIST`
   - 通过 channel 控制启动和停止
   - 将进程信息发布到 Redis（WebSocket 推送）

2. **集成到执行方法**
   - `ExecuteOnlineDDL`：直接执行的 DDL（CREATE/DROP/TRUNCATE 等）
   - `ExecuteDML`：DML 语句（INSERT/UPDATE/DELETE）

3. **监控流程**
   ```
   1. 获取 Connection ID（SELECT CONNECTION_ID()）
   2. 启动监控 goroutine（GetProcesslist）
   3. 发送开始信号（ch1 <- 1）
   4. 执行 SQL
   5. 关闭 channel（close(ch1)），通知监控停止（通过 defer 确保）
   ```

### 9.3 主要用途

1. **实时监控 SQL 执行状态**
   - 显示进程状态（State）：如 `executing`、`waiting for table lock`、`Writing to net` 等
   - 显示执行时间（Time）：SQL 已经执行了多长时间（秒）
   - 显示当前执行的 SQL（Info）：完整或截断的 SQL 语句

2. **帮助用户了解执行进度**
   - 知道 SQL 是否还在执行中
   - 知道 SQL 已经执行了多长时间
   - 判断是否有锁等待或其他问题

### 9.4 消息格式

**WebSocket 消息类型：** `processlist`

**INFORMATION_SCHEMA.PROCESSLIST 字段说明：**
- `ID`：连接 ID（Connection ID）
- `USER`：执行 SQL 的用户名
- `HOST`：客户端主机地址
- `DB`：当前使用的数据库
- `COMMAND`：命令类型（Query、Sleep 等）
- `TIME`：执行时间（秒）
- `STATE`：当前状态（executing、waiting for table lock 等）
- `INFO`：正在执行的 SQL 语句（可能被截断）

**示例输出：**
```json
{
  "type": "processlist",
  "data": {
    "ID": 12345,
    "USER": "root",
    "HOST": "127.0.0.1:54321",
    "DB": "test_db",
    "COMMAND": "Query",
    "TIME": 15,
    "STATE": "executing",
    "INFO": "ALTER TABLE test_table ADD COLUMN new_col INT"
  }
}
```

### 9.5 前端显示

前端在工单详情页的 "OSC Progress" 标签页中显示进程信息：

```javascript
if (result.type === 'processlist') {
  // Format processlist data
  logText = '当前SQL SESSION ID的SHOW PROCESSLIST输出:\n';
  for (const key in result.data) {
    logText += `${key}: ${result.data[key]}\n`;
  }
  // processlist 类型替换整个内容（不是追加）
  oscContent.value = logText;
  logBuffer.value = []; // 清空缓冲区
}
```

**显示效果：**
```
当前SQL SESSION ID的SHOW PROCESSLIST输出:
ID: 12345
USER: root
HOST: 127.0.0.1:54321
DB: test_db
COMMAND: Query
TIME: 15
STATE: executing
INFO: ALTER TABLE test_table ADD COLUMN new_col INT
```

### 9.6 使用场景

- **DDL 执行**：CREATE/DROP/TRUNCATE 等语句，可能执行时间较长
- **DML 执行**：UPDATE/DELETE 等语句，影响大量数据时执行时间较长
- **长时间运行的 SQL**：帮助用户了解执行进度，判断是否需要取消

### 9.7 注意事项

1. **仅用于直接执行的 SQL**
   - `gh-ost` 执行的 ALTER TABLE 不需要此监控（gh-ost 有自己的输出）
   - 只用于直接执行的 SQL（CREATE/DROP/UPDATE/DELETE 等）

2. **监控频率**
   - 每 500ms 查询一次，避免对数据库造成过大压力

3. **自动停止**
   - SQL 执行完成后，通过关闭 channel 自动停止监控（使用 defer 确保）
   - 如果进程信息查询失败（进程已不存在），也会自动停止

4. **多连接处理**
   - 每个执行任务使用独立的 Connection ID
   - 每个任务有独立的监控 goroutine

5. **错误处理**
   - 监控连接失败时会记录错误日志并停止监控
   - 查询失败时会自动停止监控，不影响主 SQL 执行

## 十、进度统计功能（已实现）

### 10.1 功能说明

已实现 gh-ost 进度统计功能，能够实时解析 gh-ost 输出中的进度信息，并通过 WebSocket 推送进度数据。

### 10.2 实现方式

**文件：`go-noah/internal/orders/executor/mysql.go`**

1. **新增 `parseAndPublishGhostProgress` 方法**
   - 解析 gh-ost 输出中的进度信息
   - 支持多种进度格式：
     - `Copy: 12345/1000000 1.23%` - 复制进度
     - `Progress: 1.23% ETA: 2h30m` - 进度和预计剩余时间
     - `Rows: 12345/1000000 (1.23%)` - 行数进度
     - 通用百分比格式 `1.23%`
   - 解析成功后，通过 WebSocket 推送进度数据（类型为 `ghost-progress`）

2. **集成到输出读取逻辑**
   - 在读取 gh-ost 输出的 goroutine 中，每收到一行输出就尝试解析进度信息
   - 如果解析成功，立即推送进度数据

### 10.3 进度数据格式

**WebSocket 消息类型：** `ghost-progress`

**消息内容示例：**

1. **复制进度格式**：
```json
{
  "type": "ghost-progress",
  "data": {
    "type": "ghost-progress",
    "current": 12345,
    "total": 1000000,
    "percent": 1.23,
    "operation": "copy"
  }
}
```

2. **进度和 ETA 格式**：
```json
{
  "type": "ghost-progress",
  "data": {
    "type": "ghost-progress",
    "percent": 1.23,
    "eta": "2h30m",
    "operation": "progress"
  }
}
```

3. **行数进度格式**：
```json
{
  "type": "ghost-progress",
  "data": {
    "type": "ghost-progress",
    "current": 12345,
    "total": 1000000,
    "percent": 1.23,
    "operation": "rows"
  }
}
```

### 10.4 支持的进度格式

- ✅ `Copy: 12345/1000000 1.23%` - 复制进度（包含当前数、总数、百分比）
- ✅ `Progress: 1.23% ETA: 2h30m` - 进度百分比和预计剩余时间
- ✅ `Rows: 12345/1000000 (1.23%)` - 行数进度（包含当前数、总数、百分比）
- ✅ 通用百分比格式 `1.23%` - 仅包含百分比

### 10.5 前端集成建议

前端可以通过以下方式显示进度：

```javascript
if (result.type === 'ghost-progress') {
  const progressData = result.data;
  
  // 更新进度条
  if (progressData.percent !== undefined) {
    progressBar.value = progressData.percent;
    progressText.value = `${progressData.percent.toFixed(2)}%`;
  }
  
  // 显示详细信息
  if (progressData.current !== undefined && progressData.total !== undefined) {
    progressDetail.value = `${progressData.current}/${progressData.total}`;
  }
  
  // 显示预计剩余时间
  if (progressData.eta) {
    etaText.value = `预计剩余时间: ${progressData.eta}`;
  }
}
```

### 10.6 注意事项

1. **解析频率**：每行输出都会尝试解析，但只有匹配到进度格式时才会推送
2. **重复推送**：同一个进度可能被多次推送（因为 gh-ost 会定期输出进度），前端需要处理去重
3. **百分比范围**：只处理 0-100 之间的百分比，过滤掉无效值
4. **匹配优先级**：按照定义的顺序匹配，第一个匹配成功的格式会被使用

## 十一、后续优化（可选）

1. **错误处理**
   - 更好的错误信息解析
   - 支持部分失败场景

2. **回滚支持**
   - gh-ost 支持暂停/取消
   - 支持回滚操作

3. **进度显示优化**
   - 前端进度条动画
   - 进度历史记录
   - 进度趋势分析

