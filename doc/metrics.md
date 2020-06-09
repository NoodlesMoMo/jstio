
envoy metrics
-------------

cluster manager statistics
--------------------------
| metric | type | describe | mark |
| --- | --- | --- | --- |
| cluster_added | Counter | 添加的cluster数  | 静态配置+CDS 
| cluster_modified | Counter | cluster更新数 | 单指CDS |
| cluster_removed | Counter | cluster删除数 | 单指CDS |
| cluster_updated_via_merge | Counter | cluster merge数 |
| cluster_merge_cancelled | Counter | TODO |
| cluster_out_of_merge_window | Counter | TODO |
| active_cluster | Gauge | 当前活跃+warmed cluster数 |
| warming_cluster | Gauge | 当前warming状态 cluster数 |


cluster statistics
------------------

| metric | type | describe | mark |
| --- | --- | --- | --- |
| upstream_cx_total | Conter | 当前上游累计连接数 | |
| upstream_cx_http1_total | Counter | HTTP/1.1 累计连接数 ||
| upstream_cx_http2_total | Counter | HTTP/2 累计连接数 ||
| upstream_cx_connect_fail | Counter | 上游连接累计失败数 ||
| upstream_cx_connect_timeout | Counter | 上游连接超时累加数||
| upstream_cx_idle_timeout | Counter | 上游连接累计空闲超时数||
| upsgream_cx_connect_attempts_exceeded | Counter | 上游连接超过配置的持续失败数 ||
| upstream_cx_overflow | Counter | 断路溢出累计数 |
| upstream_cx_connect_ms | Histogram | 连接建立毫秒统计分布 |
| upstream_cx_length_ms | Histogram | 连接时长分布 | （毫秒）|
| upstream_cx_destroy | Counter | 销毁连接累计数 | |
| upstream_cx_destroy_local | Counter | 销毁连接累计数 | local side | |
| upstream_cx_destroy_remote | Counter | 销毁连接累计数 | remote side | |
| upstream_cx_destroy_with_active_rq | Counter | Total connections destroyed with 1+ active request|
| upstream_cx_destroy_local_with_active_rq| Counter | Total connections destroyed locally with 1+ active request|
| upstream_cx_destroy_remote_with_active_rq| Counter|Total connections destroyed remotely with 1+ active request|
| upstream_cx_close_notify | Counter | HTTP1.1 “Connection:close”数或者HTTP2 GoAWAY累计数|
| upstream_cx_rx_bytes_total| Counter | 接收字节累计数 |
| upstream_cx_rx_bytes_buffered | Gauge | 当前缓冲区缓冲接收字节数|
| upstream_cx_tx_bytes_total | Counter | 发送连接字节数|
| upstream_cx_tx_bytes_buffered | Gauge | 当前缓冲区发送字节数|
| upstream_cx_pool_overflow | Counter | 连接池断路器溢出累计数|
| upstream_cx_protocol_error | Counter | 连接协议失败累计数 |
| upstream_cx_max_requests | Counter | 因最大请求导致连接关闭累计数|
| upstream_cx_none_healthy | Counter | 因健康检查失败导致的连接失败累计数|
| upstream_rq_total | Counter | 请求累计数 |
| upstream_rq_active | Gauge | 活跃请求累计数 |
| upstream_rq_pending_total | Counter | 请求pending累计数|
| upstream_rq_pending_overflow | Counter | 因断路溢出导致的请求失败累计数 |
| upstream_rq_pending_failure_eject| Counter | 因连接失败导致的请求失败累计数|
| upstream_rq_pending_active | Gauge | 当前活跃请求pending数|
| upstream_rq_cancelled | Counter | 从连接池获取一个可用连接过程中导致请求被cancel数 |
| upstream_rq_maintenance_mode | Counter | TODO | Total rquests that resulted in an immediate 503 due to maintenance mode |
| upstream_rq_timeout | Counter | 请求超时累计数 |
| upstream_rq_max_duration_reached | Counter | 因最大时常导致的请求关闭累计数|
| upstream_rq_per_try_timeout | Counter | Total request that hit the per try timeout|
| upstream_rq_rx_reset | Counter | remote side reset
| upstream_cx_active | Gauge | 当前活跃的上游连接数  |  |
| upstream_rq_timeout | Counter | 超时请求累计数 |
| upstream_rq_retry | Counter | 请求重试累计数 |
| upstream_rq_retry_limit_exceeded | Counter | 因达到最大重试数而导致的没有得到重试机会的累计数|
| upstream_rq_retry_success | Counter | 重试成功累计数 |
| upstream_rq_retry_overflow | Counter | 因达到断路器设置导致没机会重试的累计请求数 |
| upstream_flow_control_paused_reading_total | Counter | 因流控暂停从上游读取累计数 |
| upstream_flow_control_resumed_reading_total | Counter | 因流控恢复从上游读取累计数 |
| upstream_flow_control_backed_up_total | Counter | 上游backend up，同时抑制下游读取累计数 |
| upstream_flow_control_drained_total | Counter | 上游排空恢复下游读取累计数 |
| upstream_internal_redirect_failed_total | Counter | 因内部重定向失败导致传递到下游的累计数 |
| upstream_internal_redirect_successed_total | Counter | 
| membership_total  | Gauge | 网格中membership数量 | 
| membership_healthy | Gauge | 网格中membership健康数量 |  |
| membership_degraded | Gauge | 网格中membership降级数 | |
| assignment_stale | Counter | 旧Endpoint更新数 |  |
| assignment_timeout_received | Counter | Endpoints租约过期数 |  |
| bind_error | Counter | socket bind错误数 |  |


Dynamic HTTP statistics
-----------------------
| metric | type | describe | mark |
| --- | --- | --- | --- |
| upstream_rq_completed | Counter | 上游请求完成累计数 | |
| upstream_rq_<XX> | Counter | HTTP nXX累计数 |
| upstream_rq_<x> | Counter | e.g.: 302累计数 |
| upstream_rq_time | Hisgogram | 请求时间统计分布 | 单位: ms|
| internal.upsgream_rq_completed | Counter | 内部请求完成累计数|
| internal.upstream_rq_<xx> | Counter | 内部请求响应码统计, e.g. internal.upstream_rq_5xx |
| internal.upstream_rq_x | Counter | 内部具体相应状态码请求累计数|
| internal.upstream_rq_time | Histogram | 内部请求时间分布 | 单位: ms |
| external.upstream_rq_completed | Counter | 外部请求完成累计数|
| external.upstream_rq_<xx> | Counter | 外部nXX聚合响应码累计数 |
| external.upstream_rq_x | Counter | 外部特定响应码统计累计数 |
| external.upstream_rq_time | Hisgogram | 外部请求时间统计分布|


Circuit breaker statistics
--------------------------

| metric | type | describe | mark |
| --- | --- | --- | --- |
| cx_open | Gauge | 断路器连接状态  | 1:打开 0:关闭 |
| cx_pool_open | Gauge | 断路器连接池状态 | 1:打开 0:关闭 |
| rq_pending_open | Gauge | Pending断路器请求状态 | 1:打开 0:关闭 |
| rq_retry_open | Gauge | 断路器重试状态 | 1:打开 0:关闭 |


xDS subscription statistics
---------------------------
| metric | type | describe | mark |
| --- | --- | --- | --- |
| init_fetch_timeout | Counter | xds超时数  |  |
| config_reload | Counter | xds配置重载数  |  |
| update_attempt | Counter | xds API重试数 |
| update_success | Counter | xds完成数 |
| update_failure | Counter | 网络故障导致xds获取失败数 |
| update_rejected | Counter | schema/validation等原因导致xds失败数 |
| update_time | Gauge | 最近一次xds更新时间 | (单位:毫秒)|
| version | Gauge | 最近成功更新的xds数据哈希 | |
| version_text | TextReadout | version text | |
| control_plane.connected_state | Gauge | 当前xds-manager连接状态 | 1:ok 0:断开 |
| rate_limit_enforced | Counter | xds-manager请求速率受限统计数| |
| pending_requests | Gauge | 因请求速率限制导致的xds-manager pending请求数|
