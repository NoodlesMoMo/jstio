jstio init container
--------------------

  k8s初始化镜像。
  
  manual page
  
    NAME:
       pod-init - init pod command-line for jstio

    USAGE:
       init-container [global options] command [command options] [arguments...]

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
       --port value, -p value          xds manager server port (default: 8080) [$JSTIO_PORT]
       --admin_port value, --ap value  envoy admin port (default: 9901) [$JSTIO_ADMIN_PORT]
       --help, -h                      show help
       --version, -v                   print the version

  **说明:**
  
  1. app, cluster, namespace是必须参数。
  2. 支持环境变量，支持命令行参数。命令行参数的优先级高于环境变量。
  3. 环境变量:
    
    JSTIO_APPNAME
    JSTIO_CLUSTER
    JSTIO_NAMESPACE
    
    JSTIO_INPUT
    JSTIO_OUTPUT
    JSTIO_HOST
    JSTIO_ENV
    JSTIO_PORT
    JSTIO_ADMIN_PORT
