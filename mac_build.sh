# 打包一个基础镜像
# docker build --platform linux/amd64 -f Dockerfile.dev -t rainbond-builder .
# 正常构建命令如下
# docker run --rm --platform linux/amd64 \
  #  -v "$PWD":/go/src/github.com/goodrain/rainbond \
  #  -v rainbond-go-cache:/go/pkg/mod \
  #  -v rainbond-go-build:/root/.cache/go-build \
  #  rainbond-builder \
  #  go build -tags 'sqlite_omit_load_extension netgo' \
  #  -ldflags '-w -s' \
  #  -o rainbond-api ./cmd/api
# 本地使用直接执行该脚本即可，但是需要先打包基础镜像

#!/bin/bash

set -e

# 镜像配置
BASE_IMAGE="registry.cn-hangzhou.aliyuncs.com/goodrain/rbd-worker:1125100"
IMAGE_REGISTRY="registry.cn-hangzhou.aliyuncs.com/goodrain"

# 生成时间戳（格式：月日时分，例如：11250235）
generate_timestamp() {
    date +"%m%d%H%M"
}

# 使用说明
usage() {
    echo "Usage: $0 [api|worker|all]"
    echo "  api    - 只打包 rainbond-api"
    echo "  worker - 只打包 rainbond-worker"
    echo "  all    - 打包所有组件 (默认)"
    exit 1
}

# 构建函数
build_component() {
    local component=$1
    local output_name="rainbond-${component}"
    local cmd_path="./cmd/${component}"
    local start_time=$(date +%s)

    echo "==> 正在构建 ${output_name}..."

    docker run --rm --platform linux/amd64 \
      -v "$PWD":/go/src/github.com/goodrain/rainbond \
      -v rainbond-go-cache:/go/pkg/mod \
      -v rainbond-go-build:/root/.cache/go-build \
      rainbond-builder \
      go build -tags 'sqlite_omit_load_extension netgo' \
      -ldflags '-w -s' \
      -o "${output_name}" "${cmd_path}"

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo "✓ ${output_name} 构建完成 (耗时: ${duration}秒)"
}

# 构建并推送 Docker 镜像
build_and_push_image() {
    local component=$1
    local binary_name="rainbond-${component}"
    local timestamp=$(generate_timestamp)
    local image_name="${IMAGE_REGISTRY}/rbd-${component}:${timestamp}"
    local dockerfile_path="Dockerfile.${component}"
    local start_time=$(date +%s)

    echo ""
    echo "==> 正在构建 Docker 镜像: ${image_name}..."

    # 生成临时 Dockerfile
    cat > "${dockerfile_path}" <<EOF
FROM ${BASE_IMAGE}
COPY ${binary_name} /run/${binary_name}
EOF

    # 构建镜像
    docker build --platform linux/amd64 -f "${dockerfile_path}" -t "${image_name}" .

    # 推送镜像
    echo "==> 正在推送镜像: ${image_name}..."
    docker push "${image_name}"

    # 清理临时 Dockerfile
    rm -f "${dockerfile_path}"

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo "✓ ${image_name} 镜像构建并推送完成 (耗时: ${duration}秒)"
}

# 获取参数，默认为 all
TARGET=${1:-all}

# 记录总开始时间
TOTAL_START_TIME=$(date +%s)

case "$TARGET" in
    api)
        build_component "api"
        build_and_push_image "api"
        ;;
    worker)
        build_component "worker"
        build_and_push_image "worker"
        ;;
    all)
        build_component "api"
        build_and_push_image "api"
        echo ""
        build_component "worker"
        build_and_push_image "worker"
        ;;
    -h|--help)
        usage
        ;;
    *)
        echo "错误: 未知参数 '$TARGET'"
        usage
        ;;
esac

# 计算总耗时
TOTAL_END_TIME=$(date +%s)
TOTAL_DURATION=$((TOTAL_END_TIME - TOTAL_START_TIME))

echo ""
echo "✓ 所有构建任务完成！总耗时: ${TOTAL_DURATION}秒"
