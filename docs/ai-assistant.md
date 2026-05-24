# Future AI 站内助手

本次新增了一个最小可用的 AI 助手能力：

- 前端右下角悬浮 `Future AI` 聊天入口
- 后端接口：`POST /api/ai/chat`
- 支持 OpenAI 兼容接口，例如 DeepSeek、通义千问兼容网关、火山方舟兼容网关等
- 游客每天 5 次，登录用户每天 20 次的内存限流
- 未配置 API Key 时自动进入本地体验模式，方便先看界面效果

## 环境变量

```bash
export AI_API_KEY="你的大模型 API Key"
export AI_API_BASE="https://api.deepseek.com"
export AI_MODEL="deepseek-chat"
```

如果使用其他 OpenAI 兼容服务，只需要把 `AI_API_BASE` 改成对应服务的 base url，后端会请求：

```text
${AI_API_BASE}/v1/chat/completions
```

## 接口示例

```bash
curl -X POST http://127.0.0.1:8088/api/ai/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"这个网站怎么用？"}'
```

## 后台管理

新增后台入口：

```text
/admin
```

默认只有用户名为 `admin` 的登录用户可以访问。也可以通过环境变量扩展管理员名单：

```bash
export ADMIN_USERS="admin,你的用户名"
```

后台能力：

- 查看用户总数、AI 调用总数、真实模型调用数、本地体验回复数
- 查看最近用户列表、注册时间、登录次数、最后登录时间
- 配置 AI Base URL、模型名称、API Key
- API Key 保存后只展示脱敏结果，留空保存不会覆盖已有 Key

## 后续可扩展方向

1. 增加管理员角色表，替代环境变量管理员名单
2. 把完整对话记录落库：`ai_conversations`、`ai_messages`
3. 增加后台配置常见问答
4. 接入本站文档/公告/教程做 RAG 知识库问答
5. 把用户中心数据接入提示词，生成更个性化的欢迎语和使用报告
