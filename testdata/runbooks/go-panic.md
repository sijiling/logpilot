# Go Panic 处理手册

当日志出现 `panic: runtime error: invalid memory address or nil pointer dereference` 时：

1. 从堆栈第一条业务函数开始定位空指针来源。
2. 检查数据库查询、类型断言和指针返回值是否可能为空。
3. 为失败分支增加 nil 校验和单元测试。
4. 修复后执行 `go test -race ./...`，确认没有并发数据竞争。

不要只依赖全局 recover 掩盖错误；recover 用于阻止整个进程退出，根因仍需修复。
