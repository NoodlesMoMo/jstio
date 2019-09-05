CHANGELOG
=========

##v0.0.6 (2019-07-17)

### Changed
  1. 添加应用时，自动添加4中基本类型资源到资源表中。这样今后可以通过后台配置资源属性，实验并应用其他envoy其他特性。
  2. 修改snapshot version计算方法: 对配置进行哈希计算，取20位长度字符串作为配置版本号。当etcd有watch事件时，如果pod ip不变则无需
  更新配置。
  3. 修改 resource validation 方法: 对cluster, endpoint这两种列表资源进行特殊处理
  4. 其他小问题若干

##v0.0.7 (2019-07-22)

### Changed
  1. 调整`route`资源模版`retry`配置，杀掉pod时，不再报警。
  
### feature
  
  多节点支持

##v0.0.8 (2019-08-19)

### Changed
  1. 分布式支持
  2. 修复后台bug若干
  3. 增加HTTP post client。（DISABLE_ENVOY_PROXY=1)时关闭请求代理，可进行本地调试

##v0.0.9 (2019-09-04)

### Changed
  1. 修复添加上游时，自动添加应用资源bug。
  
### Warning
  1. 关联上游时，自动添加`cluster`, `route`, `endpoint`资源。但移除应用时，这些资源并不相应删除，需要手动根据实际情况删除。
