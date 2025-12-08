# Docker 运行服务

## 需求
1. 要有portainer的一些主要功能
    1.1 Containers管理
        1.1.1 容器日志
        1.1.2 可以进入容器命令行功能
        1.1.3 portainer Containers列表相关功能
    1.2 Images管理
    1.3 Networks管理
    1.4 Volumes管理
2. 可以自己创建一个shell脚本然后进行容器部署，并且部署后的容器和这个脚本关联，当然脚本还可以进行编辑和删除。脚本内容示例如下：
```shell
#!/usr/bin/env bash

VERSION=2.33.5
IMAGE_NAME=docker-registry.ingotcloud.top/portainer/portainer-ce
SERVICE_NAME=portainer

# 停止运行当前容器
docker ps -q --filter name="${SERVICE_NAME}" | xargs -r docker rm -f
# 删除当前镜像
docker images -q --filter reference="${IMAGE_NAME}" | xargs -r docker rmi -f
# 拉取最新镜像
docker pull ${IMAGE_NAME}

docker run -d --name portainer \
 --network ingot-net \
 --restart always \
 -p 9000:9000 \
 -e AGENT_SECRET=123456 \
 -v /var/run/docker.sock:/var/run/docker.sock \
 -v portainer_data:/data \
 ${IMAGE_NAME}:${VERSION}
```
当然如果脚本里面的执行内容是docker componse，那么可以关联docker-componse文件，也就是脚本中也要满足docker-compose部署

3. 检查当前linux系统中是否安装了docker以及docker compose。如果没有安装那么要进行安装相关内容，所以要检查一些内容：
    1. 检查当前linux是否可以连通互联网，如果可以，那么判断系统、架构、版本，然后下载docker镜像，并安装docker compose
    2. 如果当前linux不可访问互联网，那么使用离线安装方案，比如我们预留的RPM/DEB包，对应不同的系统架构进行安装
    3. 如果是离线环境，并且无法使用apt，yum这类命令，那么就需要使用官方docker包下载地址中的内容安装（https://download.docker.com/linux/static/stable/x86_64/）。当然这个包肯定是已经提前下好的。
这个需求很重要，因为我们所有的功能都是基于docker的，所以我们可能就需要提前下好指定版本的docker和docker-componse包，以及RPM/DEB包，当然不同架构和版本的都要下载好

## 技术选型

1. 前端
前端使用Vue3+TypeScript+Pinia+Element Plus+Vite构建
2. 后端
后端使用Go+Gin+Gorm进行构建

## 部署
服务调试成功后，最后打包是打成一个docker镜像，我只需要run这个镜像，就可以在指定端口打开前端页面进行使用了。
当然第一次运行程序的时候，要检查当前环境，也就是需求3中的内容，然后判断是否需要安装docker。
然后也要有一个简单的账号密码校验逻辑，所有接口都需要token（jwt），安全性还是要考虑的，第一次运行提示配置admin的密码。
