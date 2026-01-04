<a href="https://trendshift.io/repositories/9047" target="_blank"><img src="https://trendshift.io/api/badge/repositories/9047" alt="go-nunu%2Fnunu | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>


# Nunu — A CLI tool for building Go applications.

Nunu is a scaffolding tool for building Go applications. Its name comes from a game character in League of Legends, a little boy riding on the shoulders of a Yeti. Just like Nunu, this project stands on the shoulders of giants, as it is built upon a combination of popular libraries from the Go ecosystem. This combination allows you to quickly build efficient and reliable applications.

🚀Tips: This project is very complete, so updates will not be very frequent, welcome to use.

- [简体中文介绍](https://github.com/go-nunu/nunu/blob/main/README_zh.md)
- [Português](https://github.com/go-nunu/nunu/blob/main/README_pt.md)
- [日本語](https://github.com/go-nunu/nunu/blob/main/README_jp.md)

![Nunu](https://github.com/go-nunu/nunu/blob/main/.github/assets/banner.png)

## Documentation
* [User Guide](https://github.com/go-nunu/nunu/blob/main/docs/en/guide.md)
* [Architecture](https://github.com/go-nunu/nunu/blob/main/docs/en/architecture.md)
* [Getting Started Tutorial](https://github.com/go-nunu/nunu/blob/main/docs/en/tutorial.md)
* [Unit Testing](https://github.com/go-nunu/nunu/blob/main/docs/en/unit_testing.md)
* [MCP Server](https://github.com/go-nunu/nunu-layout-mcp/blob/main/README.md)


## Features
- **Gin**: https://github.com/gin-gonic/gin
- **Gorm**: https://github.com/go-gorm/gorm
- **Wire**: https://github.com/google/wire
- **Viper**: https://github.com/spf13/viper
- **Zap**: https://github.com/uber-go/zap
- **Golang-jwt**: https://github.com/golang-jwt/jwt
- **Go-redis**: https://github.com/go-redis/redis
- **Testify**: https://github.com/stretchr/testify
- **Sonyflake**: https://github.com/sony/sonyflake
- **Gocron**:  https://github.com/go-co-op/gocron
- **Go-sqlmock**:  https://github.com/DATA-DOG/go-sqlmock
- **Gomock**:  https://github.com/golang/mock
- **Swaggo**:  https://github.com/swaggo/swag
- **Casbin**:  https://github.com/casbin/casbin
- **Pitaya**:  https://github.com/topfreegames/pitaya
- **MCP-GO**:  https://github.com/mark3labs/mcp-go

- More...

## Key Features
* **Low Learning Curve and Customization**: Nunu encapsulates popular libraries that Gophers are familiar with, allowing you to easily customize the application to meet specific requirements.
* **High Performance and Scalability**: Nunu aims to be high-performance and scalable. It uses the latest technologies and best practices to ensure that your application can handle high traffic and large amounts of data.
* **Security and Reliability**: Nunu uses stable and reliable third-party libraries to ensure the security and reliability of your application.
* **Modular and Extensible**: Nunu is designed to be modular and extensible. You can easily add new features and functionality by using third-party libraries or writing your own modules.
* **Complete Documentation and Testing**: Nunu has comprehensive documentation and testing. It provides extensive documentation and examples to help you get started quickly. It also includes a test suite to ensure that your application works as expected.

## Concise Layered Architecture
Nunu adopts a classic layered architecture. In order to achieve modularity and decoupling, it uses the dependency injection framework `Wire`.

![Nunu Layout](https://github.com/go-nunu/nunu/blob/main/.github/assets/layout.png)

## Nunu CLI

![Nunu](https://github.com/go-nunu/nunu/blob/main/.github/assets/screenshot.jpg)


## Directory Structure
```
.
├── api
│   └── v1
├── cmd
│   ├── migration
│   ├── server
│   │   ├── wire
│   │   │   ├── wire.go
│   │   │   └── wire_gen.go
│   │   └── main.go
│   └── task
├── config
├── deploy
├── docs
├── internal
│   ├── handler
│   ├── middleware
│   ├── model
│   ├── repository
│   ├── server
│   └── service
├── pkg
├── scripts
├── test
│   ├── mocks
│   └── server
├── web
├── Makefile
├── go.mod
└── go.sum

```

The project architecture follows a typical layered structure, consisting of the following modules:

* `cmd`: This module contains the entry points of the application, which perform different operations based on different commands, such as starting the server or executing database migrations. Each sub-module has a `main.go` file as the entry file, as well as `wire.go` and `wire_gen.go` files for dependency injection.
* `config`: This module contains the configuration files for the application, providing different configurations for different environments, such as development and production.
* `deploy`: This module is used for deploying the application and includes deployment scripts and configuration files.
* `internal`: This module is the core module of the application and contains the implementation of various business logic.

  - `handler`: This sub-module contains the handlers for handling HTTP requests, responsible for receiving requests and invoking the corresponding services for processing.

  - `job`: This sub-module contains the logic for background tasks.

  - `model`: This sub-module contains the definition of data models.

  - `repository`: This sub-module contains the implementation of the data access layer, responsible for interacting with the database.

  - `server`: This sub-module contains the implementation of the HTTP server.

  - `service`: This sub-module contains the implementation of the business logic, responsible for handling specific business operations.

* `pkg`: This module contains some common utilities and functions.

* `scripts`: This module contains some script files used for project build, test, and deployment operations.

* `storage`: This module is used for storing files or other static resources.

* `test`: This module contains the unit tests for various modules, organized into sub-directories based on modules.

* `web`: The frontend project is located in the parent directory (`../web`), which contains the frontend-related files, such as HTML, CSS, and JavaScript.

In addition, there are some other files and directories, such as license files, build files, and README. Overall, the project architecture is clear, with clear responsibilities for each module, making it easy to understand and maintain.

## Requirements
To use Nunu, you need to have the following software installed on your system:

* Go 1.19 or higher
* Git
* Docker (optional)
* MySQL 5.7 or higher (optional)
* Redis (optional)

### Installation

You can install Nunu with the following command:

```bash
go install github.com/go-nunu/nunu@latest
```

> Tips: If `go install` succeeds but the `nunu` command is not recognized, it is because the environment variable is not configured. You can add the GOBIN directory to the environment variable.

### Create a New Project

You can create a new Go project with the following command:

```bash
nunu new projectName
```

By default, it pulls from the GitHub source, but you can also use an accelerated repository in China:

```
// Use the basic template
nunu new projectName -r https://gitee.com/go-nunu/nunu-layout-basic.git
// Use the advanced template
nunu new projectName -r https://gitee.com/go-nunu/nunu-layout-advanced.git
```

This command will create a directory named `projectName` and generate an elegant Go project structure within it.

### Create Components

You can create handlers, services, repositories, and models for your project using the following commands:

```bash
nunu create handler user
nunu create service user
nunu create repository user
nunu create model user
```
or
```
nunu create all user
```

These commands will create components named `UserHandler`, `UserService`, `UserRepository`, and `UserModel`, respectively, and place them in the correct directories.

### Run the Project

You can quickly run the project with the following command:

```bash
nunu run
```

This command will start your Go project and support hot-reloading when files are updated.

### Compile wire.go

You can quickly compile `wire.go` with the following command:

```bash
nunu wire
```

This command will compile your `wire.go` file and generate the required dependencies.

## Contribution

If you find any issues or have any improvement suggestions, please feel free to raise an issue or submit a pull request. Your contributions are highly appreciated!

## License

Nunu is released under the MIT License. For more information, see the [LICENSE](LICENSE) file.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=go-nunu/nunu&type=Date)](https://star-history.com/#go-nunu/nunu&Date)

## Noah 框架（无 Wire 依赖）
- Noah 是在本项目中自研的轻量依赖装配框架，用于完全替代 Google Wire。
- 保留既有目录结构、第三方组件引用方式与封装规范，移除 Wire 后不影响现有功能。
- 入口仍在 `cmd` 下，分别构建 `server`、`task`、`migration` 三类 App。

### 关键改动
- 新增 `pkg/noah/noah.go`，实现三类应用的装配函数：
  - `noah.NewServerApp(conf, logger) (*app.App, func(), error)`
  - `noah.NewTaskApp(conf, logger) (*app.App, func(), error)`
  - `noah.NewMigrateApp(conf, logger) (*app.App, func(), error)`
- 删除 `cmd/*/wire` 目录与所有 `wire_gen.go` 文件。
- 移除 `go.mod` 中的 `github.com/google/wire` 依赖。
- 更新 `cmd/*/main.go` 将 `wire.NewWire` 替换为对应的 `noah.New*App`。
- 更新 `Makefile init` 目标，去除 Wire 安装步骤。

### 兼容性与封装
- 构造链保持不变：`DB`、`Casbin Enforcer`、`Repository`、`Transaction`、`Service`、`Handler`、`Server`。
- 第三方库保持原有使用：`Gin`、`Gorm`、`Viper`、`Zap`、`JWT`、`Casbin`、`Gocron` 等。
- 清理函数沿用 Gorm 连接关闭逻辑，维持稳定性。

### 迁移指南
- 将原入口中的 `wire.NewWire(conf, logger)` 替换为：
  - `server`：`noah.NewServerApp(conf, logger)`
  - `task`：`noah.NewTaskApp(conf, logger)`
  - `migration`：`noah.NewMigrateApp(conf, logger)`
- 删除 `cmd/*/wire` 目录及其生成文件，确保不再引用 `google/wire`。
- 执行依赖更新，保持 `go.mod` 无 Wire 依赖。
- 现有单元测试与构建脚本无需调整；如需新增用例，请围绕 `pkg/noah` 装配进行扩展。

### 验证
- 运行项目现有测试：`make test` 或 `go test ./...`。
- 查看编译与启动日志是否与原行为一致（不需要运行服务时，可仅进行编译检查）。

### 设计说明
- Noah 通过手动构造的依赖链替代生成式装配，减少编译期工具依赖，提升透明度与可控性。
- 由于保持了同样的模块边界与构造函数，后续功能扩展与组件替换成本与原项目一致。
