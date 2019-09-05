# Jstio

  一个轻量级的适合odin平台的xds protocol manager application。
  
  它与业务pod之间的关系如下图所示:
  
  ![jstio](http://img.shouji.sogou.com/wapdl/hole/201909/05/2019090514395801664548.png)
  
  现阶段，`odin.sogou`域名及其子域名下的所有服务都解析到odin集群的nginx集群。`odin k8s`内部的访问都经过nginx LB做反向代理。
  
  经过本土化的mesh改造后，集群内部的请求将不再通过nginx转发，而是在原有业务容器基础上再额外配置一个`envoy`容器（`sidecar`模式）。
  由`envoy`来做`正向代理`。业务容器请求127.0.0.1:80，将请求主动送给envoy，由envoy来代理请求。
  
  那么，envoy是如何知道该怎样正确代理？这就是`jstio`的主要工作了:
  
  1. jstio是一个envoy配置管理的服务。
  2. envoy与jstio之间通过grpc来通信。grpc之上运行`xds protocol`的资源交换协议。
  
## 依赖
  - [x] **ETCD**: jstio需要及时感知到pod节点变动。
  - [x] **Mysql**: 存储应用的组织关系。
  - [x] **NSQ**: jstio分布式模式下，资源变动消息队列。
  
## 快速开始

  假设项目部署在 `srvdotandroid` 之后（应用作为srvdotandroid的上游）。配置过程如下：
  
  1. 添加应用:
  访问： http://jstio.shouji.sogou/index ,打开jstio web console，添加一个想要接入的应用。
  
  ![jstio_app_add](http://img.shouji.sogou.com/wapdl/hole/201909/05/2019090515200246677126.png)
  
  ![jstio_app_edit](http://img.shouji.sogou.com/wapdl/hole/201909/05/2019090515241742219460.png)
  
  **注意** 这一步非常关键，要填写的内容不多，但一定要三思而确定。
  
  
  2. 在原有`odin deployment`资源配置基础上稍作改动
  
  ![deployment](http://img.shouji.sogou.com/wapdl/hole/201909/05/2019090514594428846214.png)
  
  - 增加`initcontainer`。这是一个短生命周期的特殊阶段容器，主要为业务容器做初始化相关的工作。这里我们用这个容器执行一个命令行，生成
  envoy的基础启动配置。
  
  ```sh
NAME:
   pod-init - init pod command-line for jstio

USAGE:
   init [global options] command [command options] [arguments...]

VERSION:
   2019-08-29

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --app value, -a value           application name [$JSTIO_APPNAME]
   --cluster value, -c value       odin cluster [$JSTIO_CLUSTER]
   --namespace value, -n value     odin namespace [$JSTIO_NAMESPACE]
   --input value, -i value         config abs path (default: "template/envoy.yaml.template") [$JSTIO_INPUT]
   --output value, -o value        config output path (default: "/etc/envoy/envoy.yaml") [$JSTIO_OUTPUT]
   --host value, -x value          xds manager server host (default: "jstio.shouji.sogou") [$JSTIO_HOST]
   --environment value, -e value   app run environment (default: "product") [$JSTIO_ENV]
   --region value, -r value        where node? (default: "beijing") [$JSTIO_REGION]
   --port value, -p value          xds manager server port (default: 8080) [$JSTIO_PORT]
   --admin_port value, --ap value  envoy admin port (default: 9901) [$JSTIO_ADMIN_PORT]
   --help, -h                      show help
   --version, -v                   print the version
```

  除了支持命令行传参，`envoy-init`同样支持环境变量。
  ```bash
JSTIO_APPNAME: 应用名称
JSTIO_CLUSTER: 集群名称
JSTIO_NAMESPACE: 集群命名空间 (默认值: test集群为oneclass, venus为planet)
JSTIO_HOST: jstio服务IP或域名
JSTIO_PORT: jstio服务运行端口号
JSTIO_ADMIN_PORT: jstio web console端口号
```

  **注意**: ⚠️, 命令行与环境变量同时指定时，以命令行参数为准。
  
