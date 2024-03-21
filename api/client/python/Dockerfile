# We build our image as the official autorest Dockerfile is outdated
# and not compatible with the latest autorest.
# More specifically, the latest autorest npm package depends on
# other Azure packages that require a higher node version.
# Official image: https://github.com/Azure/autorest/blob/63ffe68961e24ed8aa59a2ca4c16a8019c271e45/docker/base/ubuntu/Dockerfile

# Autorest is incompatible with latest node version
FROM node:20-alpine

# Install autorest
RUN npm install -g autorest

# Install python/pip
RUN apk add --no-cache python3 py3-pip

ENTRYPOINT [ "autorest" ]
