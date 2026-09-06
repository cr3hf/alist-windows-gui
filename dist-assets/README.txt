AList 桌面版 (alist_win) v1.0.0
================================

一个为 alist 文件列表程序提供的 Windows 桌面外壳：
内置 alist 内核，无需命令行即可在本机快速部署与使用 alist。

■ 快速上手
1. 双击 alist_win.exe 启动（首次运行自动初始化，账号密码均为 admin）。
2. 点击"浏览器打开"进入 alist 管理页面（默认 http://127.0.0.1:5244）。
3. 关闭窗口 = 最小化到系统托盘；托盘右键"退出"才真正停止服务。

■ SmartScreen 提示
本程序未做代码签名。首次运行如出现 "Windows 已保护你的电脑"，
点击"更多信息" → "仍要运行" 即可。

■ 账号与密码
- 默认账号密码均为 admin；本程序不保存任何密码。
- 在 alist 网页端修改过密码后，可点击主界面"重置密码"一键还原为 admin / admin
  （服务运行中会短暂重启；若在网页端改过用户名，密码仍会重置，用户名需在网页端改回）。

■ 数据存放位置
- 安装模式：C:\Users\<你>\AppData\Local\alist_win\data
- 便携模式（目录内存在 portable.ini）：与本程序同目录的 data 文件夹

■ 开源与许可
- 内置内核：alist (https://github.com/AlistGo/alist)，AGPL-3.0 许可证。
  许可证全文见 bin\LICENSE-alist.txt。
- 对应内核版本源码：https://github.com/AlistGo/alist/tree/<tag>
  （"关于"页可查看当前内核版本号）
- 本桌面外壳不含任何激活 / 授权 / 试用期机制。

■ 卸载
控制面板或开始菜单"卸载 AList 桌面版"；卸载时可选是否删除用户数据。
