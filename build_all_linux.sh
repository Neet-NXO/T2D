#!/bin/bash

# T2D 多架构编译脚本
# 支持编译所有常见的 Linux 架构版本

set -e

APP_NAME="T2D"
CMD_DIR="cmd/t2d"
BUILD_DIR="build"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 定义支持的架构和操作系统
declare -A PLATFORMS=(
    ["linux-amd64"]="linux amd64"
    ["linux-386"]="linux 386"
    ["linux-arm64"]="linux arm64"
    ["linux-arm"]="linux arm"
    ["linux-armv5"]="linux arm"
    ["linux-armv6"]="linux arm"
    ["linux-armv7"]="linux arm"
    ["linux-mips"]="linux mips"
    ["linux-mipsle"]="linux mipsle"
    ["linux-mips64"]="linux mips64"
    ["linux-mips64le"]="linux mips64le"
    ["linux-ppc64"]="linux ppc64"
    ["linux-ppc64le"]="linux ppc64le"
    ["linux-riscv64"]="linux riscv64"
    ["linux-s390x"]="linux s390x"
)

# ARM 版本特殊处理
declare -A ARM_VERSIONS=(
    ["linux-armv5"]="5"
    ["linux-armv6"]="6"
    ["linux-armv7"]="7"
)

# 编译标志
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}T2D 多架构编译脚本${NC}"
echo -e "${BLUE}版本: ${VERSION}${NC}"
echo -e "${BLUE}构建时间: ${BUILD_TIME}${NC}"
echo -e "${BLUE}Git提交: ${GIT_COMMIT}${NC}"
echo ""

# 清理旧的构建文件
echo -e "${YELLOW}清理旧的构建文件...${NC}"
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到 Go 编译器${NC}"
    exit 1
fi

echo -e "${GREEN}Go 版本: $(go version)${NC}"
echo ""

# 下载依赖
echo -e "${YELLOW}下载依赖...${NC}"
go mod download
go mod tidy
echo ""

# 编译函数
build_platform() {
    local platform=$1
    local goos=$2
    local goarch=$3
    local goarm=$4
    
    local output_name="${APP_NAME}-${platform}"
    local output_path="${BUILD_DIR}/${output_name}"
    
    echo -e "${BLUE}编译 ${platform}...${NC}"
    
    # 设置环境变量
    export GOOS=${goos}
    export GOARCH=${goarch}
    
    if [[ -n "$goarm" ]]; then
        export GOARM=${goarm}
    else
        unset GOARM
    fi
    
    # 编译
    if go build -ldflags="${LDFLAGS}" -o "${output_path}" "./${CMD_DIR}"; then
        # 获取文件大小
        local size=$(du -h "${output_path}" | cut -f1)
        echo -e "${GREEN}✓ ${platform} 编译成功 (${size})${NC}"
        
        # 创建压缩包
        cd ${BUILD_DIR}
        tar -czf "${output_name}.tar.gz" "${output_name}"
        rm "${output_name}"
        cd ..
        
        local compressed_size=$(du -h "${BUILD_DIR}/${output_name}.tar.gz" | cut -f1)
        echo -e "${GREEN}✓ ${platform} 压缩包创建成功 (${compressed_size})${NC}"
    else
        echo -e "${RED}✗ ${platform} 编译失败${NC}"
        return 1
    fi
}

# 开始编译
echo -e "${YELLOW}开始编译所有平台...${NC}"
echo ""

success_count=0
fail_count=0
failed_platforms=()

for platform in "${!PLATFORMS[@]}"; do
    IFS=' ' read -r goos goarch <<< "${PLATFORMS[$platform]}"
    
    # 检查是否是 ARM 特殊版本
    if [[ -n "${ARM_VERSIONS[$platform]}" ]]; then
        goarm="${ARM_VERSIONS[$platform]}"
    else
        goarm=""
    fi
    
    if build_platform "$platform" "$goos" "$goarch" "$goarm"; then
        ((success_count++))
    else
        ((fail_count++))
        failed_platforms+=("$platform")
    fi
    echo ""
done

# 重置环境变量
unset GOOS GOARCH GOARM

# 生成校验和文件
echo -e "${YELLOW}生成校验和文件...${NC}"
cd ${BUILD_DIR}
sha256sum *.tar.gz > SHA256SUMS
md5sum *.tar.gz > MD5SUMS
cd ..
echo -e "${GREEN}✓ 校验和文件生成完成${NC}"
echo ""

# 创建发布信息文件
cat > ${BUILD_DIR}/RELEASE_INFO.txt << EOF
T2D Release Information
============================

版本: ${VERSION}
构建时间: ${BUILD_TIME}
Git提交: ${GIT_COMMIT}
编译器版本: $(go version)

支持的平台:
EOF

for platform in "${!PLATFORMS[@]}"; do
    if [[ ! " ${failed_platforms[@]} " =~ " ${platform} " ]]; then
        echo "- ${platform}" >> ${BUILD_DIR}/RELEASE_INFO.txt
    fi
done

echo "" >> ${BUILD_DIR}/RELEASE_INFO.txt
echo "安装说明:" >> ${BUILD_DIR}/RELEASE_INFO.txt
echo "1. 下载对应平台的压缩包" >> ${BUILD_DIR}/RELEASE_INFO.txt
echo "2. 解压: tar -xzf T2D-<platform>.tar.gz" >> ${BUILD_DIR}/RELEASE_INFO.txt
echo "3. 运行: ./T2D-<platform> -config <config_file>" >> ${BUILD_DIR}/RELEASE_INFO.txt

# 输出编译结果
echo -e "${BLUE}==================== 编译完成 ====================${NC}"
echo -e "${GREEN}成功: ${success_count} 个平台${NC}"
if [[ $fail_count -gt 0 ]]; then
    echo -e "${RED}失败: ${fail_count} 个平台${NC}"
    echo -e "${RED}失败的平台: ${failed_platforms[*]}${NC}"
fi
echo ""
echo -e "${YELLOW}构建文件位置: ${BUILD_DIR}/${NC}"
echo -e "${YELLOW}文件列表:${NC}"
ls -lh ${BUILD_DIR}/
echo ""
echo -e "${GREEN}所有文件已准备就绪，可以进行发布！${NC}"
