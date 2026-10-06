# 15: 判题契约扩展——结构体类题支持

**What to build:** 让判题链路支持输入/输出为结构体（链表/树/多返回值/有状态类）的题目：判题契约 v2——TestCase 增加可选的「构造器 setup / 比较器 compare」声明（不声明 = 现行 eval/repr 行为一字不改，完全向后兼容）；Python 侧构造辅助库（list_to_nodes / build_tree 等进沙箱 harness 注入包）+ 规范化 repr（链表→[1,2,3] 风格、树→层序列表，None/循环安全）；循环链表特例（带环构造 + 按节点身份的入环比较，规范化 repr 对环有界输出）；有状态类题的调用序列协议（LRU/MyQueue：setup 构造对象 + calls 逐方法 eval + 每步即时比对首败即停）；judge proto 扩展（JudgeCase 可选 setup/compare 字段）+ 三端重生成；每类结构体至少 1 个真实容器端到端样题。完成后约 20 道 LeetCode 结构体题（链表 12、树 8、LRU、用栈实现队列）具备入库条件。

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] 契约 v2 落地：TestCase（Go ports.go）与 case dict（Python executor）携带可选 setup/compare；缺省路径走现行 eval/repr
- [ ] 向后兼容硬性：现役模板 + smoke 客户端 + 全部既有测试不改一字全绿
- [ ] 构造辅助库进沙箱（经 stdin 注入包）：list_to_nodes / build_tree / 规范化 repr（环有界）；setup/compare 作为题作者代码注入前过与玩家代码相同的静态卫兵
- [ ] 循环链表：值序列 + pos（尾指向下标，-1=无环）表示；身份比较沿入环节点走环；repr 有界（超时断言不死循环）
- [ ] 有状态类调用序列协议：setup 构造 + calls 逐个 eval + 每步期望即时比对，首败即停沿用现行协议
- [ ] judge proto 扩展（optional 字段）+ scripts/gen-proto.sh 三端重生成；smoke 如引用 case 结构已同步
- [ ] 端到端样题×4 真实容器（@needs_docker）：反转链表（比较器）/ 环形链表（循环构造+身份比较）/ 层序遍历（树构造）/ LRU（调用序列）；其 Call/Expected/setup/compare 写成 12 号票可直接抄的格式范例，附 docs/design/002 附录
- [ ] 虚假 pass 零容忍断言：比较器各 false 分支、setup 先于玩家代码、环有界 repr（超时断言）、expected 不进容器（stdin 载荷契约测试强化版）
- [ ] docs/adr/0004-judge-struct-contract.md：题作者代码进容器的信任边界记录 + 否决项（宿主侧比较需结构体跨容器序列化破坏零泄漏；JSON schema 声明式表达不了节点身份）
