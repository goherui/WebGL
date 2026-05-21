# 未来实验室 WebGL 登录系统设计文档

**日期**: 2026-05-21  
**项目**: 未来实验室 - WebGL交互体验系统  
**状态**: 设计完成，待实现

---

## 一、项目概述

### 1.1 项目目标
构建一个包含用户认证系统的"未来实验室"风格WebGL网页，用户登录后进入沉浸式的3D实验室环境。

### 1.2 技术栈
- **后端**: Go 1.22+ + gorilla/sessions + Redis
- **前端**: 原生HTML/CSS/JavaScript + Three.js
- **数据库**: MySQL (用户存储) + Redis (Session存储)
- **Session管理**: gorilla/sessions库 + Redis后端

### 1.3 系统架构
```
┌─────────────┐      HTTP请求      ┌──────────────────┐
│   浏览器     │ ────────────────→ │   Go后端服务器   │
│  (前端UI)    │ ←──────────────── │  (8088端口)      │
└─────────────┘    Cookie(Session) └────────┬─────────┘
                                            │
                      ┌─────────────────────┼─────────────────────┐
                      ↓                     ↓                     ↓
              ┌───────────────┐   ┌─────────────────┐   ┌───────────────┐
              │   MySQL数据库  │   │   Redis缓存     │   │   静态文件    │
              │  (用户表)      │   │  (Session存储)  │   │  (HTML/CSS/JS)│
              └───────────────┘   └─────────────────┘   └───────────────┘
```

---

## 二、后端设计

### 2.1 新增依赖
```go
github.com/gorilla/sessions  // Session管理
github.com/go-redis/redis/v8  // Redis客户端
```

### 2.2 API接口设计

#### 登录接口 `/api/login`
- **方法**: POST
- **请求体**: 
```json
{
  "username": "string",
  "password": "string"
}
```
- **成功响应**: 
```json
{
  "code": 0,
  "msg": "登录成功",
  "data": {
    "username": "xxx"
  }
}
```
- **失败响应**:
```json
{
  "code": 1,
  "msg": "错误信息"
}
```

#### 注册接口 `/api/register`
- **方法**: POST
- **请求体**: 同登录接口
- **响应**: 标准响应结构

#### 退出接口 `/api/logout`
- **方法**: POST
- **功能**: 清除Session
- **响应**: 
```json
{
  "code": 0,
  "msg": "退出成功"
}
```

#### 检查认证 `/api/check-auth`
- **方法**: GET
- **响应**:
```json
{
  "code": 0,
  "data": {
    "username": "xxx"
  }
}
// 或
{
  "code": 1,
  "msg": "未登录"
}
```

### 2.3 静态文件路由
| 路由 | 文件 | 认证要求 |
|------|------|---------|
| `/` | index.html | 无（已登录跳转/welcome） |
| `/welcome` | welcome.html | 需要 |
| `/lab` | lab.html | 需要 |
| `/static/*` | 静态资源 | 无 |

### 2.4 Session设计
- **Session名称**: `session_id`
- **存储内容**: username, user_id, login_time
- **过期时间**: 24小时 (86400秒)
- **Redis Key格式**: `session:<session_id>`
- **安全配置**:
  - HttpOnly: true (防XSS)
  - Secure: false (开发环境，生产环境需启用)
  - SameSite: Lax

---

## 三、数据库设计

### 3.1 MySQL用户表（已存在）
```sql
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 3.2 Redis Session结构
```
Key: session:<随机session_id>
Type: Hash
TTL: 86400秒
Fields:
  - username: 用户名
  - user_id: 用户ID
  - login_time: 登录时间戳
```

---

## 四、前端设计

### 4.1 页面流转
```
登录页 (index.html) 
    ↓ 登录成功
欢迎页 (welcome.html) 
    ↓ 点击"进入实验室"
实验室页 (lab.html)
    ↓ 点击"退出登录"
登录页 (index.html)
```

### 4.2 登录页面 (index.html)
- **功能**: 用户登录和注册
- **现有功能**: 保持不变
- **修改点**: 登录成功后跳转到 `/welcome`

### 4.3 欢迎页面 (welcome.html)
**布局**:
- 全屏渐变背景 (#2dd4bf → #3b82f6)
- 居中白色卡片
- 毛玻璃效果

**内容**:
- 用户头像（圆形，渐变背景，首字母）
- 欢迎文本："欢迎回来，{用户名}"
- 主按钮："进入实验室 →"（渐变，悬停放大）
- 次按钮："退出登录"（透明边框）

**交互**:
- 页面加载时检查Session有效性
- 无效Session重定向到登录页
- 点击"进入实验室"跳转到 `/lab`
- 点击"退出登录"调用 `/api/logout`

### 4.4 实验室页面 (lab.html)
**设计要点**:
- 全屏WebGL Canvas背景
- 顶部HUD导航栏
- 右下角退出按钮

**顶部导航栏**:
- 左侧: "FUTURE LAB" 标题
- 右侧: 用户名 + 设置图标
- 配色: 半透明黑色背景

**底部导航**:
- 四个导航项: INIT, EXPLORE, ANALYZE, SYSTEM
- 悬停高亮效果
- 右下角: "退出登录" 按钮

**WebGL场景** (基于之前创建的FutureLab类):
- 深色主题背景
- 动态3D网格
- 漂浮几何体
- 粒子系统
- 鼠标交互响应

**配色方案**:
```css
--primary-1: #2dd4bf;    /* 青绿色 */
--primary-2: #3b82f6;    /* 蓝色 */
--bg-dark: #0a0a0f;       /* 深色背景 */
--text-white: #ffffff;    /* 白色文字 */
--glass-bg: rgba(0, 0, 0, 0.5);  /* 毛玻璃背景 */
```

### 4.5 样式统一规范
- **主色调**: #2dd4bf (青绿) → #3b82f6 (蓝色)
- **背景色**: #1a1a2e (深蓝灰)
- **文字色**: #ffffff (白色)
- **按钮样式**: 圆角6px, 渐变背景, 悬停放大
- **字体**: PingFang SC, Microsoft YaHei, Helvetica Neue, sans-serif
- **毛玻璃效果**: backdrop-filter: blur(10px)
- **阴影**: box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35)

---

## 五、安全策略

### 5.1 Session安全
- 使用crypto/rand生成256位随机Session ID
- HttpOnly Cookie防止XSS攻击
- Secure Flag (生产环境启用)
- SameSite=Lax防止CSRF

### 5.2 密码安全
- bcrypt加密 (cost=10)
- 最小长度6位
- 用户输入trim和验证

### 5.3 API安全
- CORS配置 (开发环境允许所有源)
- Content-Type验证
- 请求方法限制 (POST用于写操作)

### 5.4 前端安全
- Session验证页面加载时
- 无效Session自动重定向
- 输入内容基本转义

---

## 六、项目文件结构

```
d:\桌面\WebGL/
├── main.go                      # Go后端主程序 (重构)
├── go.mod                       # Go模块定义
├── go.sum                       # 依赖版本锁定
├── restart.sh                   # 重启脚本
├── server.log                   # 服务器日志
├── static/
│   ├── index.html               # 登录/注册页面 (修改)
│   ├── welcome.html             # 欢迎页面 (新增)
│   ├── lab.html                # WebGL实验室页面 (新增)
│   ├── css/
│   │   ├── style.css           # 通用样式
│   │   └── lab.css             # 实验室页面样式
│   ├── js/
│   │   ├── auth.js             # 认证逻辑
│   │   └── lab.js              # Three.js场景
│   └── assets/
│       └── (图片资源)
└── docs/
    └── specs/
        └── 2026-05-21-future-lab-design.md  # 本文档
```

---

## 七、实现计划

### 阶段一：后端Session支持
1. 添加gorilla/sessions和redis依赖
2. 实现Redis连接
3. 配置Session Store
4. 修改登录/注册/退出接口
5. 添加认证中间件
6. 添加检查认证接口
7. 实现页面路由保护

### 阶段二：前端页面开发
1. 创建welcome.html欢迎页
2. 创建lab.html实验室页
3. 创建共享CSS样式
4. 实现auth.js认证逻辑
5. 重构lab.js Three.js场景
6. 统一UI风格

### 阶段三：功能完善
1. 添加用户统计功能
2. 优化WebGL效果
3. 性能优化 (LOD, 剔除)
4. 安全加固
5. 错误处理优化

### 阶段四：测试与部署
1. 单元测试
2. 集成测试
3. 性能测试
4. 部署配置

---

## 八、验收标准

### 8.1 功能验收
- [ ] 用户可以注册新账户
- [ ] 用户可以登录
- [ ] 登录后跳转到欢迎页
- [ ] 欢迎页显示用户名
- [ ] 可以进入实验室
- [ ] WebGL场景正常显示
- [ ] 可以退出登录
- [ ] Session过期后需要重新登录

### 8.2 视觉验收
- [ ] 登录页保持现有风格
- [ ] 欢迎页配色统一
- [ ] 实验室页WebGL效果流畅
- [ ] 鼠标交互响应正常
- [ ] UI风格整体协调

### 8.3 性能验收
- [ ] 页面加载时间 < 3秒
- [ ] WebGL帧率 > 30FPS
- [ ] 登录响应时间 < 500ms
- [ ] 内存占用合理

---

## 九、已知限制

1. **Redis依赖**: 系统需要运行Redis服务器
2. **HTTPS**: 生产环境需要配置HTTPS以启用Secure Cookie
3. **浏览器支持**: 需要现代浏览器支持WebGL 2.0
4. **移动端**: WebGL页面移动端体验可能受限

---

## 十、后续扩展建议

1. **用户权限**: 添加管理员和普通用户角色
2. **个人中心**: 用户可以修改密码和个人信息
3. **记住我**: 添加"记住我"功能延长Session
4. **社交登录**: 支持微信、GitHub等第三方登录
5. **WebGL交互**: 在实验室中添加更多可交互元素
6. **数据统计**: 记录用户访问行为和偏好

---

**文档版本**: 1.0  
**创建日期**: 2026-05-21  
**最后更新**: 2026-05-21  
**作者**: AI Assistant
