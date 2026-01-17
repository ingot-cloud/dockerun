# Dockerun 开发路线图

## 📋 文档信息

- **版本**: v1.1
- **更新日期**: 2026-01-17
- **团队**: Ingot Cloud
- **维护者**: JY
- **状态**: 持续更新中

---

## 🎯 总体目标

将 Dockerun 从单机部署工具升级为支持分布式 Agent 架构的容器管理平台，类似于 GitLab + Runner 的架构设计。

**核心价值**:
- 🚀 支持多机器分布式部署
- 🔄 统一的部署管理界面
- 🎯 灵活的任务调度策略
- 📊 完整的监控和日志
- 🔐 企业级安全特性

---

## 📅 开发阶段

### Phase 0: 准备阶段（当前）

**时间**: 2025-01 Week 1-2  
**状态**: ✅ 进行中

#### 任务清单

- [x] 完成架构设计文档
- [x] 创建开发路线图
- [ ] 技术选型确认
- [ ] 开发环境搭建
- [ ] 项目结构调整

#### 技术选型

| 组件 | 技术选择 | 原因 |
|------|---------|------|
| 通信协议 | gRPC + Protobuf | 高性能、类型安全、流式支持 |
| 备选协议 | WebSocket | 兼容性好、易于调试 |
| 数据库 | SQLite / MySQL | 已有支持，易于扩展 |
| 日志 | 结构化日志 + File | 易于分析和追踪 |
| 监控 | Prometheus (可选) | 标准化指标收集 |

#### 项目结构

```
dockerun/
├── backend/
│   ├── internal/
│   │   ├── agent/          # 新增：Agent 管理
│   │   │   ├── server.go      # Master 端服务
│   │   │   ├── registry.go    # Agent 注册中心
│   │   │   ├── dispatcher.go  # 任务分发器
│   │   │   ├── monitor.go     # 健康监控
│   │   │   └── token.go       # Token 管理
│   │   ├── models/
│   │   │   ├── agent.go       # 新增：Agent 模型
│   │   │   └── agent_task.go  # 新增：Agent 任务模型
│   │   └── ... (现有目录)
│   └── proto/              # 新增：gRPC 协议定义
│       └── agent/
│           ├── agent.proto
│           └── agent.pb.go
├── agent/                  # 新增：Agent 独立程序
│   ├── cmd/
│   │   └── agent/
│   │       └── main.go
│   ├── internal/
│   │   ├── client/         # Agent 客户端
│   │   ├── executor/       # 任务执行器
│   │   └── config/         # Agent 配置
│   ├── Makefile
│   └── README.md
├── frontend/
│   └── src/
│       └── views/
│           └── AgentManagement.vue  # 新增：Agent 管理界面
└── docs/                   # 新增：详细文档
    ├── agent-setup.md
    ├── api-reference.md
    └── troubleshooting.md
```

#### 交付物

- ✅ [AGENT_ARCHITECTURE_DESIGN.md](./AGENT_ARCHITECTURE_DESIGN.md)
- ✅ [ROADMAP.md](./ROADMAP.md)
- ⏳ 技术方案评审文档
- ⏳ 开发环境配置指南

---

### Phase 1: 基础架构（v0.9.0）

**时间**: 2025-01 Week 3-4 到 2025-02 Week 1-2  
**状态**: ⏳ 待开始  
**目标**: 建立 Agent 基础通信框架

#### 里程碑 1.1: gRPC 通信框架

**时间**: Week 1-2  
**优先级**: 🔥 高

**任务**:

1. **Protocol Buffers 定义** (2 天)
   - [ ] 定义 Agent Service 接口
   - [ ] 定义消息结构
   - [ ] 生成 Go 代码
   - [ ] 编写单元测试

2. **Master 端 gRPC 服务器** (3 天)
   - [ ] 实现 gRPC 服务器启动
   - [ ] 实现 RegisterAgent 接口
   - [ ] 实现 Heartbeat 接口
   - [ ] 实现基础的连接管理
   - [ ] 添加 TLS 支持
   - [ ] 编写集成测试

3. **Agent 端 gRPC 客户端** (3 天)
   - [ ] 实现 Agent 客户端连接
   - [ ] 实现自动重连机制
   - [ ] 实现心跳发送
   - [ ] 实现断线重连
   - [ ] 编写单元测试

**参考文档**: [AGENT_ARCHITECTURE_DESIGN.md - 通信协议](./AGENT_ARCHITECTURE_DESIGN.md#31-grpc-service-定义)

**验收标准**:
- ✅ Agent 可以成功连接到 Master
- ✅ 心跳正常工作
- ✅ 网络中断后可以自动重连
- ✅ 测试覆盖率 > 80%

#### 里程碑 1.2: Agent 注册和管理

**时间**: Week 3-4  
**优先级**: 🔥 高

**任务**:

1. **数据模型** (1 天)
   - [ ] 创建 Agent 表结构
   - [ ] 创建 Agent Token 表
   - [ ] 数据库迁移脚本
   - [ ] GORM 模型定义

2. **Agent Registry** (3 天)
   - [ ] 实现 Agent 注册逻辑
   - [ ] 实现 Agent 信息存储
   - [ ] 实现 Agent 查询接口
   - [ ] 实现 Agent 更新和删除
   - [ ] 实现 Agent 状态管理
   - [ ] 编写单元测试

3. **Token 管理** (2 天)
   - [ ] 实现 Token 生成
   - [ ] 实现 Token 验证
   - [ ] 实现 Token 哈希存储
   - [ ] 实现 Token 过期管理
   - [ ] 编写单元测试

4. **Backend API** (2 天)
   - [ ] 创建 Agent 管理 API
   - [ ] GET /api/agents - 列出所有 Agent
   - [ ] POST /api/agents - 创建 Agent
   - [ ] GET /api/agents/:id - 获取 Agent 详情
   - [ ] PUT /api/agents/:id - 更新 Agent
   - [ ] DELETE /api/agents/:id - 删除 Agent
   - [ ] API 文档

**参考文档**: [AGENT_ARCHITECTURE_DESIGN.md - Agent Registry](./AGENT_ARCHITECTURE_DESIGN.md#11-agent-registry)

**验收标准**:
- ✅ Agent 可以成功注册并获取 Token
- ✅ Master 可以查询和管理所有 Agent
- ✅ Token 验证正常工作
- ✅ API 文档完整

#### 里程碑 1.3: 健康监控

**时间**: Week 5-6  
**优先级**: 🔶 中

**任务**:

1. **Health Monitor** (3 天)
   - [ ] 实现心跳检查逻辑
   - [ ] 实现 Agent 状态更新
   - [ ] 实现离线检测
   - [ ] 实现告警机制
   - [ ] 编写单元测试

2. **Agent 客户端增强** (2 天)
   - [ ] 添加系统信息收集
   - [ ] 添加指标上报
   - [ ] 优化心跳性能
   - [ ] 编写单元测试

**参考文档**: [AGENT_ARCHITECTURE_DESIGN.md - Health Monitor](./AGENT_ARCHITECTURE_DESIGN.md#13-health-monitor)

**验收标准**:
- ✅ 能够检测 Agent 离线
- ✅ 离线后状态正确更新
- ✅ 系统信息正确收集和显示

#### Phase 1 交付物

- ⏳ Agent 通信框架完成
- ⏳ Agent 注册和管理完成
- ⏳ 健康监控完成
- ⏳ 基础 API 文档
- ⏳ 单元测试和集成测试

---

### Phase 2: 任务调度（v0.9.5）

**时间**: 2025-02 Week 3-4 到 2025-03 Week 1-2  
**状态**: ⏳ 待开始  
**目标**: 实现任务分发和执行

#### 里程碑 2.1: 任务分发器

**时间**: Week 1-2  
**优先级**: 🔥 高

**任务**:

1. **数据模型** (1 天)
   - [ ] 创建 AgentTask 表结构
   - [ ] 创建 AgentTaskLog 表
   - [ ] 数据库迁移
   - [ ] GORM 模型定义

2. **Task Dispatcher** (4 天)
   - [ ] 实现任务队列
   - [ ] 实现标签匹配策略
   - [ ] 实现最小负载策略
   - [ ] 实现轮询策略
   - [ ] 实现任务分发逻辑
   - [ ] 实现任务状态跟踪
   - [ ] 编写单元测试

3. **gRPC 任务接口** (2 天)
   - [ ] 实现 ReceiveTasks 流式接口
   - [ ] 实现任务推送逻辑
   - [ ] 实现任务取消
   - [ ] 编写集成测试

**参考文档**: [AGENT_ARCHITECTURE_DESIGN.md - Task Dispatcher](./AGENT_ARCHITECTURE_DESIGN.md#12-task-dispatcher)

**验收标准**:
- ✅ 任务可以成功分发到 Agent
- ✅ 标签匹配正确工作
- ✅ 负载均衡策略有效
- ✅ 任务状态正确跟踪

#### 里程碑 2.2: 任务执行器

**时间**: Week 3-4  
**优先级**: 🔥 高

**任务**:

1. **Agent 任务执行** (4 天)
   - [ ] 实现任务接收
   - [ ] 实现工作目录准备
   - [ ] 实现镜像拉取
   - [ ] 实现脚本执行
   - [ ] 实现结果收集
   - [ ] 实现错误处理
   - [ ] 编写单元测试

2. **任务状态报告** (2 天)
   - [ ] 实现状态上报
   - [ ] 实现日志收集
   - [ ] 实现进度更新
   - [ ] 编写单元测试

3. **并发控制** (2 天)
   - [ ] 实现任务并发限制
   - [ ] 实现资源管理
   - [ ] 实现队列管理
   - [ ] 编写单元测试

**参考文档**: [AGENT_ARCHITECTURE_DESIGN.md - Task Executor](./AGENT_ARCHITECTURE_DESIGN.md#22-task-executor)

**验收标准**:
- ✅ Agent 可以成功执行部署任务
- ✅ 任务结果正确上报
- ✅ 并发控制正常工作
- ✅ 错误处理完善

#### 里程碑 2.3: Webhook 集成

**时间**: Week 5  
**优先级**: 🔶 中

**任务**:

1. **Webhook 配置扩展** (2 天)
   - [ ] 扩展 WebhookConfig 模型
   - [ ] 添加 Agent 选择字段
   - [ ] 数据库迁移
   - [ ] 更新配置 API

2. **Webhook 处理逻辑** (3 天)
   - [ ] 修改 processTask 逻辑
   - [ ] 集成任务分发器
   - [ ] 实现本地/远程执行选择
   - [ ] 添加异步状态更新
   - [ ] 编写集成测试

**参考文档**: [AGENT_ARCHITECTURE_DESIGN.md - Webhook 集成](./AGENT_ARCHITECTURE_DESIGN.md#4-webhook-集成)

**验收标准**:
- ✅ Webhook 可以触发 Agent 执行
- ✅ 支持本地和远程执行切换
- ✅ 状态更新正确
- ✅ 集成测试通过

#### Phase 2 交付物

- ⏳ 任务分发器完成
- ⏳ 任务执行器完成
- ⏳ Webhook 集成完成
- ⏳ 端到端测试通过

---

### Phase 3: Web 界面（v1.0.0）

**时间**: 2025-03 Week 3-4 到 2025-04 Week 1  
**状态**: ⏳ 待开始  
**目标**: 完善用户界面和用户体验

#### 里程碑 3.1: Agent 管理界面

**时间**: Week 1-2  
**优先级**: 🔥 高

**任务**:

1. **Agent 列表页面** (3 天)
   - [ ] 创建 AgentManagement.vue
   - [ ] 实现 Agent 列表展示
   - [ ] 实现状态筛选
   - [ ] 实现搜索功能
   - [ ] 实时状态更新
   - [ ] 响应式设计

2. **Agent 详情页面** (2 天)
   - [ ] 创建 AgentDetail.vue
   - [ ] 显示 Agent 详细信息
   - [ ] 显示系统信息
   - [ ] 显示任务历史
   - [ ] 显示资源使用情况

3. **Agent 操作** (2 天)
   - [ ] 添加 Agent 对话框
   - [ ] 编辑 Agent 对话框
   - [ ] 删除确认对话框
   - [ ] 测试连接功能
   - [ ] Token 复制功能

**界面参考**:

```
┌─────────────────────────────────────────────────────────┐
│  Dockerun - Agent Management                    [+Add] │
├─────────────────────────────────────────────────────────┤
│  🔍 Search...  [Filter: All ▼]  [Group: All ▼]        │
├────┬────────┬──────────┬────────┬────────┬────────────┤
│    │ Name   │ Status   │ Tasks  │ Tags   │ Actions    │
├────┼────────┼──────────┼────────┼────────┼────────────┤
│ 🟢 │ web-01 │ Online   │ 2/5    │ web    │ [Edit][Del]│
│ 🟢 │ db-01  │ Online   │ 1/5    │ db     │ [Edit][Del]│
│ 🔴 │ app-01 │ Offline  │ 0/5    │ app    │ [Edit][Del]│
└────┴────────┴──────────┴────────┴────────┴────────────┘
```

**验收标准**:
- ✅ 所有页面正常显示
- ✅ 实时状态更新工作
- ✅ 所有操作功能正常
- ✅ 响应式设计完善

#### 里程碑 3.2: 任务管理界面

**时间**: Week 3  
**优先级**: 🔶 中

**任务**:

1. **任务列表页面** (2 天)
   - [ ] 创建 AgentTaskList.vue
   - [ ] 显示任务列表
   - [ ] 任务状态筛选
   - [ ] 任务搜索
   - [ ] 分页功能

2. **任务详情页面** (2 天)
   - [ ] 创建 AgentTaskDetail.vue
   - [ ] 显示任务详情
   - [ ] 实时日志流
   - [ ] 执行结果展示
   - [ ] 重试功能

**验收标准**:
- ✅ 任务列表正常显示
- ✅ 任务详情完整
- ✅ 实时日志工作正常
- ✅ 操作功能完善

#### 里程碑 3.3: Webhook 配置增强

**时间**: Week 4  
**优先级**: 🔶 中

**任务**:

1. **Webhook 配置表单** (2 天)
   - [ ] 添加 Agent 选择器
   - [ ] 添加标签选择器
   - [ ] 添加调度策略选择
   - [ ] 添加优先级配置

2. **配置验证** (1 天)
   - [ ] Agent 可用性验证
   - [ ] 标签匹配验证
   - [ ] 配置预览

**验收标准**:
- ✅ 配置表单完整
- ✅ 验证逻辑正确
- ✅ 用户体验良好

#### Phase 3 交付物

- ⏳ Agent 管理界面完成
- ⏳ 任务管理界面完成
- ⏳ Webhook 配置增强完成
- ⏳ UI/UX 测试通过
- ⏳ 用户文档

---

### Phase 4: Agent 程序（v1.0.0）

**时间**: 2025-04 Week 2-3  
**状态**: ⏳ 待开始  
**目标**: 完善独立的 Agent 程序

#### 里程碑 4.1: Agent 独立程序

**时间**: Week 1-2  
**优先级**: 🔥 高

**任务**:

1. **Agent 程序结构** (1 天)
   - [ ] 创建 agent/ 目录结构
   - [ ] 创建 main.go
   - [ ] 创建 Makefile
   - [ ] 创建 README.md

2. **配置管理** (2 天)
   - [ ] 实现配置文件加载
   - [ ] 实现环境变量支持
   - [ ] 实现配置验证
   - [ ] 创建默认配置

3. **命令行接口** (2 天)
   - [ ] 实现 start 命令
   - [ ] 实现 stop 命令
   - [ ] 实现 status 命令
   - [ ] 实现 version 命令

4. **系统服务** (2 天)
   - [ ] 创建 systemd 服务文件
   - [ ] 创建安装脚本
   - [ ] 创建卸载脚本
   - [ ] 创建启动脚本

**验收标准**:
- ✅ Agent 程序可以独立运行
- ✅ 配置文件正常工作
- ✅ 命令行接口完善
- ✅ 可以作为系统服务运行

#### 里程碑 4.2: 安装和部署

**时间**: Week 3  
**优先级**: 🔥 高

**任务**:

1. **安装脚本** (2 天)
   - [ ] 创建一键安装脚本
   - [ ] 支持多平台（Linux/macOS）
   - [ ] 自动下载和安装
   - [ ] 配置向导

2. **部署文档** (2 天)
   - [ ] 编写安装指南
   - [ ] 编写配置指南
   - [ ] 编写故障排查
   - [ ] 创建示例配置

**安装脚本示例**:

```bash
#!/bin/bash
# install-agent.sh

curl -fsSL https://dockerun.example.com/install-agent.sh | bash

# 或手动安装
wget https://dockerun.example.com/downloads/dockerun-agent-linux-amd64
chmod +x dockerun-agent-linux-amd64
sudo mv dockerun-agent-linux-amd64 /usr/local/bin/dockerun-agent

# 配置
sudo mkdir -p /etc/dockerun-agent
cat > /etc/dockerun-agent/config.yaml << EOF
master: https://dockerun.example.com
token: YOUR_AGENT_TOKEN
name: $(hostname)
tags:
  - production
  - web
capacity: 5
EOF

# 安装服务
sudo dockerun-agent install
sudo systemctl start dockerun-agent
sudo systemctl enable dockerun-agent
```

**验收标准**:
- ✅ 一键安装脚本工作正常
- ✅ 支持多平台
- ✅ 文档完整清晰
- ✅ 示例配置完善

#### Phase 4 交付物

- ⏳ Agent 独立程序完成
- ⏳ 安装脚本完成
- ⏳ 部署文档完成
- ⏳ 测试通过

---

### Phase 5: 优化和测试（v1.0.0）

**时间**: 2025-04 Week 4 到 2025-05 Week 1  
**状态**: ⏳ 待开始  
**目标**: 性能优化、测试和文档完善

#### 里程碑 5.1: 性能优化

**时间**: Week 1  
**优先级**: 🔶 中

**任务**:

1. **性能测试** (2 天)
   - [ ] 压力测试
   - [ ] 并发测试
   - [ ] 延迟测试
   - [ ] 资源使用测试

2. **性能优化** (3 天)
   - [ ] 连接池优化
   - [ ] 任务队列优化
   - [ ] 数据库查询优化
   - [ ] 内存使用优化
   - [ ] 网络传输优化

**性能目标**:
- Agent 连接数: ≥ 100
- 任务吞吐量: ≥ 50/秒
- 心跳延迟: < 1 秒
- 任务分发延迟: < 100ms
- 内存使用: < 500MB（Master）

#### 里程碑 5.2: 测试覆盖

**时间**: Week 2  
**优先级**: 🔥 高

**任务**:

1. **单元测试** (2 天)
   - [ ] 补充缺失的单元测试
   - [ ] 提高测试覆盖率
   - [ ] 测试覆盖率报告

2. **集成测试** (2 天)
   - [ ] 端到端测试
   - [ ] 多 Agent 测试
   - [ ] 故障转移测试
   - [ ] 网络异常测试

3. **用户测试** (1 天)
   - [ ] Beta 测试
   - [ ] 用户反馈收集
   - [ ] 问题修复

**测试目标**:
- 单元测试覆盖率: ≥ 80%
- 集成测试通过率: 100%
- 关键路径测试: 100%

#### 里程碑 5.3: 文档完善

**时间**: Week 3  
**优先级**: 🔶 中

**任务**:

1. **用户文档** (2 天)
   - [ ] 快速开始指南
   - [ ] 完整安装指南
   - [ ] 配置参考
   - [ ] 常见问题
   - [ ] 故障排查

2. **开发文档** (1 天)
   - [ ] API 参考文档
   - [ ] 架构说明
   - [ ] 贡献指南
   - [ ] 开发环境搭建

3. **运维文档** (1 天)
   - [ ] 部署最佳实践
   - [ ] 监控和告警
   - [ ] 备份和恢复
   - [ ] 升级指南

**文档清单**:
- ✅ [AGENT_ARCHITECTURE_DESIGN.md](./AGENT_ARCHITECTURE_DESIGN.md)
- ⏳ docs/quickstart.md
- ⏳ docs/installation.md
- ⏳ docs/configuration.md
- ⏳ docs/api-reference.md
- ⏳ docs/troubleshooting.md
- ⏳ docs/best-practices.md

#### Phase 5 交付物

- ⏳ 性能优化完成
- ⏳ 测试覆盖率达标
- ⏳ 文档完善
- ⏳ v1.0.0 正式发布

---

### Phase 6: 高级特性（v1.1.0+）

**时间**: 2025-05 及以后  
**状态**: ⏳ 规划中  
**目标**: 添加高级功能

#### 功能列表

1. **实时日志流** (v1.1.0)
   - [ ] WebSocket 日志流
   - [ ] 日志过滤
   - [ ] 日志搜索
   - [ ] 日志下载

2. **任务统计和报表** (v1.1.0)
   - [ ] 任务执行统计
   - [ ] Agent 性能统计
   - [ ] 图表展示
   - [ ] 导出报表

3. **Agent 分组** (v1.1.0)
   - [ ] 创建 Agent 组
   - [ ] 组级别任务调度
   - [ ] 组权限管理

4. **自动扩缩容** (v1.2.0)
   - [ ] 负载监控
   - [ ] 自动扩容触发
   - [ ] 自动缩容
   - [ ] 云平台集成

5. **Master 集群** (v2.0.0)
   - [ ] Master 高可用
   - [ ] 数据同步
   - [ ] 负载均衡
   - [ ] 故障转移

6. **插件系统** (v2.0.0)
   - [ ] 插件架构
   - [ ] 插件市场
   - [ ] 第三方集成

---

## 📊 进度跟踪

### 总体进度

```
Phase 0: 准备阶段          [████████████████████] 100% ✅
Phase 1: 基础架构          [░░░░░░░░░░░░░░░░░░░░]   0% ⏳
Phase 2: 任务调度          [░░░░░░░░░░░░░░░░░░░░]   0% ⏳
Phase 3: Web 界面          [░░░░░░░░░░░░░░░░░░░░]   0% ⏳
Phase 4: Agent 程序        [░░░░░░░░░░░░░░░░░░░░]   0% ⏳
Phase 5: 优化和测试        [░░░░░░░░░░░░░░░░░░░░]   0% ⏳
Phase 6: 高级特性          [░░░░░░░░░░░░░░░░░░░░]   0% ⏳
```

### 版本计划

| 版本 | 计划发布 | 主要功能 | 状态 |
|------|---------|---------|------|
| v0.8.x | 当前 | 基础 Webhook 功能 | ✅ 已发布 |
| v0.9.0 | 2025-02 | Agent 基础架构 | ⏳ 开发中 |
| v0.9.5 | 2025-03 | 任务调度 | ⏳ 计划中 |
| v1.0.0 | 2025-04 | Agent 架构完整版 | ⏳ 计划中 |
| v1.1.0 | 2025-06 | 增强功能 | ⏳ 计划中 |
| v2.0.0 | 2025-Q4 | 企业功能 | ⏳ 规划中 |

---

## 🎯 关键指标

### 开发指标

- **代码质量**: 测试覆盖率 ≥ 80%
- **文档完整性**: 所有 API 和功能有文档
- **代码审查**: 所有 PR 需要审查
- **持续集成**: 所有测试自动运行

### 性能指标

- **Agent 数量**: 支持 100+ Agent
- **任务吞吐**: 50+ 任务/秒
- **响应时间**: API < 100ms, 任务分发 < 100ms
- **可用性**: 99.9% 正常运行时间

### 用户指标

- **安装成功率**: > 95%
- **用户满意度**: > 4.5/5
- **文档清晰度**: > 4.0/5
- **问题解决率**: > 90%

---

## 🔄 迭代流程

### 开发流程

```
1. 需求分析 → 2. 设计方案 → 3. 技术评审 → 4. 编码实现 → 
5. 代码审查 → 6. 测试验证 → 7. 文档更新 → 8. 发布部署
```

### 每周例会

- **时间**: 每周一
- **内容**:
  - 上周进度回顾
  - 本周计划
  - 问题讨论
  - 风险识别

### 里程碑评审

- 每个里程碑完成后进行评审
- 确认交付物完成
- 验收标准检查
- 决定是否进入下一阶段

---

## 🚀 发布流程

### 版本发布

1. **代码冻结**: 发布前 3 天
2. **回归测试**: 完整功能测试
3. **文档更新**: 更新 CHANGELOG 和文档
4. **发布准备**: 构建二进制文件
5. **正式发布**: 发布到 GitHub Releases
6. **公告通知**: 发布公告和升级指南

### 发布检查清单

- [ ] 所有测试通过
- [ ] 文档已更新
- [ ] CHANGELOG 已更新
- [ ] 版本号已更新
- [ ] 构建成功
- [ ] 安装脚本测试通过
- [ ] 升级路径测试通过

---

## 📚 相关资源

### 文档

- [AGENT_ARCHITECTURE_DESIGN.md](./AGENT_ARCHITECTURE_DESIGN.md) - 架构设计
- [DEPLOYMENT.md](./DEPLOYMENT.md) - 部署指南
- [SECURITY.md](./SECURITY.md) - 安全指南
- [CONTRIBUTING.md](./CONTRIBUTING.md) - 贡献指南

### 工具

- **项目管理**: GitHub Issues / Projects
- **代码托管**: GitHub
- **持续集成**: GitHub Actions
- **文档托管**: GitHub Pages

### 社区

- **Issues**: 报告问题和建议
- **Discussions**: 讨论和交流
- **Pull Requests**: 贡献代码

---

## 🤝 参与贡献

欢迎贡献代码、文档和建议！

### 如何贡献

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

### 代码规范

- Go: 遵循 Go 官方风格指南
- Vue: 遵循 Vue 官方风格指南
- 提交信息: 使用约定式提交

---

## 📝 更新日志

| 日期 | 版本 | 更新内容 | 作者 |
|------|------|---------|------|
| 2025-01-07 | v1.0 | 初始版本，完整路线图 | Team |

---

## 📞 联系方式

如有问题或建议，请通过以下方式联系：

- **GitHub Issues**: https://github.com/ingot-cloud/dockerun/issues
- **团队**: Ingot Cloud
- **维护者**: JY
- **Email**: magician.of.technique@gmail.com

---

**注意**: 此路线图可能根据实际开发情况进行调整。我们会定期更新进度和计划。

**Made with ❤️ by Ingot Cloud Team**

