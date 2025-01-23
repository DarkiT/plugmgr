### redbean.ini 的配置选项

[redbean](https://redbean.dev/) 是一个开源的网络服务器，以zip可执行文件的形式存在，可在六种操作系统上运行。

##### 服务器设置：

- `port`：服务器监听的端口号（默认：8080）
- `addr`：绑定的IP地址（默认：0.0.0.0）
- `daemon`：作为守护进程运行（true/false）
- `log`：日志文件路径
- `access_log`：访问日志文件路径
- `error_log`：错误日志文件路径
- `pid`：PID文件路径
- `user`：运行服务器的用户
- `group`：运行服务器的用户组
- `chroot`：chroot的目录
- `ssl`：启用SSL（true/false）
- `ssl_cert`：SSL证书文件路径
- `ssl_key`：SSL密钥文件路径
- `ssl_password`：SSL密钥文件密码

1. ##### MIME类型：

    - `mime_type`：设置自定义MIME类型（例如，`mime_type.xyz=application/x-xyz`）

2. ##### URL重写：

    - `rewrite`：URL重写规则

3. ##### 目录列表：

    - `dir_list`：启用目录列表（true/false）
    - `dir_index`：默认索引文件名（逗号分隔）

4. ##### CGI设置：

    - `cgi_timeout`：CGI脚本超时时间（秒）
    - `cgi_dir`：CGI脚本目录

5. ##### Lua设置：

    - `lua_path`：Lua模块搜索路径
    - `lua_cpath`：Lua C模块搜索路径

6. ##### 安全设置：

    - `access_control_allow_origin`：设置CORS头
    - `strict_transport_security`：设置HSTS头
    - `content_security_policy`：设置CSP头

7. ##### 性能设置：

    - `workers`：工作线程数
    - `max_connections`：最大同时连接数
    - `keep_alive_timeout`：保持连接超时时间（秒）
    - `gzip`：启用gzip压缩（true/false）
    - `gzip_types`：需要压缩的MIME类型（逗号分隔）

8. ##### 缓存：

    - `cache_control`：设置Cache-Control头
    - `etag`：启用ETag头（true/false）

9. ##### 自定义错误页面：

    - `error_page`：设置自定义错误页面（例如，`error_page.404=/custom_404.html`）

10. ##### 虚拟主机：

    - `vhost`：配置虚拟主机

11. ##### 代理设置：

    - `proxy_pass`：配置反向代理设置

12. ##### WebSocket设置：

    - `websocket`：启用WebSocket支持（true/false）

13. ##### 基本认证：

    - `auth_basic`：启用基本认证
    - `auth_basic_user_file`：基本认证用户文件路径

14. ##### 速率限制：

    - `limit_req`：配置请求速率限制

15. ##### IP过滤：

    - `allow`：允许特定IP地址或范围
    - `deny`：拒绝特定IP地址或范围

16. ##### 文件服务：

    - `alias`：为目录创建别名
    - `try_files`：指定服务请求时尝试的文件列表

17. ##### 其他：

    - `server_tokens`：控制Server头的发送（on/off）
    - `client_max_body_size`：允许的客户端请求正文最大大小

> [!重要]
>
> 请记住，这些选项的可用性和语法可能会随着您使用的redbean版本略有不同。始终参考[官方redbean文档](https://redbean.dev/)以获取最新和特定版本的信息。
