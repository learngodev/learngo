# Learn GO Backend

基于 Go + Gin + GORM 的校园教学互动后端。支持账号认证、师生管理、作业布置与批改、随笔笔记、即时对话等功能。

## 技术栈概览

- **语言**：Go 1.22+
- **Web 框架**：Gin
- **ORM**：GORM（支持 SQLite / PostgreSQL）
- **鉴权**：JWT（`Authorization: Bearer <token>`）
- **WebSocket**：gorilla/websocket 实现会话消息推送

## 运行准备

1. 在项目根目录创建 `.env`，配置数据库、JWT 等参数。
2. 执行迁移（应用启动时自动执行 `AutoMigrate`）。
3. 启动服务：

   ```bash
   go run ./cmd/server
   ```

## 认证说明

所有受保护的接口均需在 Header 提供 `Authorization: Bearer <access_token>`。中间件会解析 JWT，识别账号 ID 与角色：

- **Admin**：可访问管理、教师、学生接口。
- **Teacher**：可访问作业教师端接口、对话、笔记等。
- **Student**：可访问作业学生端、笔记模块、对话等。

## API 列表

### 认证

| 方法 | 路径 | 描述 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 账号登录，返回 `access_token`、`refresh_token` 以及账号信息。|
| `POST` | `/api/v1/auth/refresh` | 使用 `refresh_token` 获取新的访问令牌对。|
| `POST` | `/api/v1/auth/password/reset/request` | 当账号被管理员要求重置密码时，发起重置请求并获取一次性令牌。|
| `POST` | `/api/v1/auth/password/reset/confirm` | 携带一次性令牌与新密码，完成密码重置并重新激活账号。|

#### 请求示例

```json
{
  "school_id": "school-1",
  "identifier": "admin001",
  "password": "pass123"
}
```

#### 响应示例

```json
{
  "success": true,
  "data": {
    "access_token": "...",
    "refresh_token": "...",
    "account": {
      "id": "acc-001",
      "school_id": "school-1",
      "role": "admin",
      "identifier": "admin001",
      "display_name": "管理员"
    }
  }
}
```

---

### 管理后台（管理员角色）

| 方法 | 路径 | 描述 |
| --- | --- | --- |
| `POST` | `/api/v1/admin/teachers` | 创建教师账号。|
| `POST` | `/api/v1/admin/students` | 创建学生账号并绑定班级、任课教师。|
| `POST` | `/api/v1/admin/departments` | 新建院系。|
| `POST` | `/api/v1/admin/classes` | 新建班级。|
| `GET` | `/api/v1/admin/departments` | 列出院系列表。|
| `GET` | `/api/v1/admin/departments/:id/classes` | 查看指定院系下的班级。|

#### 请求字段说明

- 创建教师：`school_id`, `number`, `name`, `email`, `phone`, `default_password`
- 创建学生：`school_id`, `number`, `name`, `email`, `phone`, `class_id`, `teacher_ids[]`, `default_password`
- 创建院系：`school_id`, `name`
- 创建班级：`school_id`, `department_id`, `name`

---

### 作业模块

#### 教师端（教师或管理员角色）

| 方法 | 路径 | 描述 |
| --- | --- | --- |
| `POST` | `/api/v1/assignments` | 创建作业及题目。|
| `GET` | `/api/v1/assignments/:id` | 查看作业详情。|
| `GET` | `/api/v1/assignments/:id/submissions` | 列出该作业所有提交概况。|
| `GET` | `/api/v1/assignments/:id/submissions/:submissionID` | 查看指定提交详情与批注。|
| `PATCH` | `/api/v1/assignments/:id/submissions/:submissionID/grade` | 批改：更新总分、子题得分、评语、教师批注。|

#### 创建作业请求

```json
{
  "course_id": "course-1",
  "teacher_id": "teacher-1",
  "class_id": "class-1",
  "type": "homework",
  "title": "Chapter 1",
  "description": "完成课后题",
  "start_at": "2025-09-01T08:00:00Z",
  "due_at": "2025-09-07T23:59:59Z",
  "max_score": 100,
  "allow_resubmit": true,
  "questions": [
    {
      "type": "text",
      "prompt": "解释概念A",
      "options": "",
      "answer": "示例答案",
      "score": 40,
      "order_index": 1
    }
  ]
}
```

#### 批改请求

```json
{
  "score": 88.5,
  "feedback": "整体不错，注意第2题",
  "item_scores": {
    "item-1": 40,
    "item-2": 48.5
  },
  "comment": {
    "content": "请复习第二章内容"
  }
}
```

#### 学生端（学生角色）

| 方法 | 路径 | 描述 |
| --- | --- | --- |
| `POST` | `/api/v1/assignments/:id/submissions` | 提交或更新作业答案。|
| `GET` | `/api/v1/assignments/:id` | 查看作业详情。|
| `GET` | `/api/v1/assignments/:id/submissions/me` | 查看自己的提交、评分、教师批注。|

#### 提交作业请求

```json
{
  "student_id": "student-1",
  "status": "submitted",
  "score": null,
  "feedback": "",
  "answers": [
    {
      "question_id": "question-1",
      "answer": "我的回答",
      "score": null
    }
  ]
}
```

#### 学生查看提交响应（节选）

```json
{
  "success": true,
  "data": {
    "submission": {
      "id": "submission-1",
      "assignment_id": "assign-1",
      "student_id": "student-1",
      "status": "graded",
      "score": 90,
      "feedback": "表现良好",
      "submitted_at": "2025-09-05T10:00:00Z",
      "items": [
        {
          "id": "item-1",
          "question_id": "question-1",
          "answer": "我的回答",
          "score": 45
        }
      ]
    },
    "comments": [
      {
        "id": "comment-1",
        "submission_id": "submission-1",
        "author_id": "teacher-1",
        "author_role": "teacher",
        "content": "请关注第3题",
        "created_at": "2025-09-05T12:00:00Z"
      }
    ]
  }
}
```

---

### 笔记 / 随笔模块（学生角色，部分接口教师也可访问）

| 方法 | 路径 | 描述 |
| --- | --- | --- |
| `POST` | `/api/v1/notes` | 创建笔记。|
| `GET` | `/api/v1/notes` | 查看本人全部笔记（含草稿/删除状态）。|
| `GET` | `/api/v1/notes/published` | 查看全校公开笔记。|
| `PATCH` | `/api/v1/notes/:id` | 更新笔记属性。|
| `DELETE` | `/api/v1/notes/:id` | 软删除笔记。|
| `POST` | `/api/v1/notes/:id/restore` | 恢复笔记。|
| `POST` | `/api/v1/notes/:id/comments` | 新增评论。|
| `GET` | `/api/v1/notes/:id/comments` | 查看评论列表。|

#### 创建笔记请求

```json
{
  "title": "化学实验心得",
  "content": "...",
  "visibility": "class",
  "status": "published"
}
```

---

### 对话 / 聊天模块（学生、教师、管理员均可访问）

| 方法 | 路径 | 描述 |
| --- | --- | --- |
| `POST` | `/api/v1/conversations` | 创建会话，对 `participant_ids` 可指定多账号实现群聊。|
| `GET` | `/api/v1/conversations` | 列出本人参与的会话。|
| `GET` | `/api/v1/conversations/:id/messages` | 按时间倒序分页获取历史消息。|
| `POST` | `/api/v1/conversations/:id/messages` | 发送消息，支持 text/image/video/audio/file。|
| `POST` | `/api/v1/conversations/:id/read` | 标记已读状态。|
| `GET` | `/api/v1/conversations/:id/stream` | WebSocket 接口，获取实时消息与事件。|

#### WebSocket

- 握手路径：`GET /api/v1/conversations/:id/stream`
- 协议：Bearer Token 仍需置于 HTTP Header。
- 事件：
  - 出站消息推送（`message`）
  - 未读计数更新（`receipt`）
  - 成员加入/离开等（可按需扩展）

#### 发送消息请求

```json
{
  "kind": "text",
  "text": "大家好",
  "media_uri": "",
  "metadata": ""
}
```

---

## 错误响应约定

所有接口失败时均返回如下格式：

```json
{
  "success": false,
  "error": {
    "message": "错误描述",
    "details": "可选的内部错误信息"
  }
}
```

- `message` 为用户可读提示。
- `details` 在生产环境可选择移除或隐藏。

---

## 后续拓展建议

- 集成 `swaggo/swag` 自动生成 Swagger/OpenAPI 文档。
- 增加刷新令牌、密码重置等账号管理能力。
- 对接 IM/通知系统，实现更丰富的广播和系统消息。
- 增加单元测试与集成测试覆盖。
