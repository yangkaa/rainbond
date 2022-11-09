VERSION=kaniko3 ./release.sh chaos
docker tag rainbond/rbd-chaos:kaniko3 registry.cn-hangzhou.aliyuncs.com/yangkaa/rbd-chaos:kaniko3
docker push registry.cn-hangzhou.aliyuncs.com/yangkaa/rbd-chaos:kaniko3