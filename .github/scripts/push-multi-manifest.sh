

docker_manifest="$IMAGE_NAMESPACE/${{ matrix.component }}:${{ github.event.client_payload.version }}"
docker_amd_image="$docker_manifest-amd64"
docker_arm_image="$docker_manifest-arm64"
ali_manifest="$DOMESTIC_BASE_NAME/$DOMESTIC_NAMESPACE/${{ matrix.component }}:${{ github.event.client_payload.version }}"
ali_amd_image="$ali_manifest-amd64"
ali_arm_image="$ali_manifest-arm64"
docker login -u $DOCKER_USERNAME -p $DOCKER_PASSWORD
docker login -u $DOMESTIC_DOCKER_USERNAME -p $DOMESTIC_DOCKER_PASSWORD ${DOMESTIC_BASE_NAME}
docker pull $docker_amd_image
docker pull $docker_arm_image
docker tag $docker_amd_image $ali_amd_image
docker tag $docker_arm_image $ali_arm_image
docker manifest create $docker_manifest $docker_amd_image $docker_arm_image
docker manifest create $ali_manifest $ali_amd_image $ali_arm_image
docker manifest push $docker_manifest
docker manifest push $ali_manifest