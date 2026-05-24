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

## 后续可扩展方向

1. 把对话记录落库：`ai_conversations`、`ai_messages`、`ai_usage_logs`
2. 增加后台配置常见问答
3. 接入本站文档/公告/教程做 RAG 知识库问答
4. 把用户中心数据接入提示词，生成更个性化的欢迎语和使用报告
