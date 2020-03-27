# gateway
tcp，websocket网关服务，

为什么需要网关
* 游戏服的高可用架构，对公网仅暴漏一个网关入口。在需要时进行水平扩展
* DDOS攻击时，IDC为了保障总出口只能屏蔽IP，网关可以避免服务直接暴漏而导致的服务无法访问
* 需要使用高防ip服务时，因为端口数量的限制，网关在高防ip后可以避免过多的端口占用
* 当服务部署在高延迟地区时，网关可以通过中转的方式达到优化延迟的效果

如何连接
1. 生成 Passphrase 。并保存配置。
2. 部署服务
3. 使用 AES 算法加密文本格式的后端地址，生成 base64 编码的密文。
4. 两种方式建立连接
    * TCP网关模式

        客户端与网关建立连接后，先发送值为0x90的一个字节（byte），
        再发送值为二进制密文长度的一个字节（byte）， 再将后端地址的二进制密文发送给网关。
    * Websocket网关模式

		客户端与网关建立连接后，将后端地址的密文文本发送给网关。

tcp网关需要解决的问题
* too many open files 问题
* 客户端真实ip获取
* 简单的限流策略
* websocket 转发
