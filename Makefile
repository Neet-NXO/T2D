# T2D Makefile

# 变量定义
APP_NAME = T2D
BUILD_DIR = build
CMD_DIR = cmd/$(APP_NAME)
GO_FILES = $(shell find . -name '*.go' -type f)

# 默认目标
.PHONY: all
all: build

# 构建
.PHONY: build
build: $(BUILD_DIR)/$(APP_NAME)

$(BUILD_DIR)/$(APP_NAME): $(GO_FILES)
	@mkdir -p $(BUILD_DIR)
	@echo "Building $(APP_NAME)..."
	go build -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "Build completed: $(BUILD_DIR)/$(APP_NAME)"

# 清理
.PHONY: clean
clean:
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)
	@echo "Clean completed"

# 运行客户端
.PHONY: run-client
run-client: build
	@echo "Starting client..."
	./$(BUILD_DIR)/$(APP_NAME) -config client_config.json

# 运行服务端
.PHONY: run-server
run-server: build
	@echo "Starting server..."
	./$(BUILD_DIR)/$(APP_NAME) -config server_config.json

# 格式化代码
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Format completed"

# 代码检查
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet completed"

# 下载依赖
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated"

# 测试
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...
	@echo "Tests completed"

# 安装
.PHONY: install
install:
	@echo "Installing $(APP_NAME)..."
	go install ./$(CMD_DIR)
	@echo "Install completed"

# 显示帮助
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the application"
	@echo "  clean      - Clean build directory"
	@echo "  run-client - Build and run client"
	@echo "  run-server - Build and run server"
	@echo "  fmt        - Format Go code"
	@echo "  vet        - Run go vet"
	@echo "  deps       - Download and tidy dependencies"
	@echo "  test       - Run tests"
	@echo "  install    - Install the application"
	@echo "  help       - Show this help message"