# Learn GO Backend

基于 Go + Gin + GORM 的校园教学互动后端。支持账号认证、师生管理、作业布置与批改、随笔笔记、即时对话等功能。

## 技术栈概览

- **语言**：Go 1.22+
- **Web 框架**：Gin
- **ORM**：GORM（支持 SQLite / PostgreSQL）
- **鉴权**：JWT（`Authorization: Bearer <token>`）
- **WebSocket**：gorilla/websocket 实现会话消息推送

## 快速开始

1. **准备依赖**：安装 Go 1.22+，并准备 SQLite（默认内置）或 PostgreSQL 数据库；若需聊天推送请开放 WebSocket 端口。
2. **复制环境变量**：在项目根目录创建 `.env`，并按需调整下方示例：

  ```dotenv
  APP_ENV=local
  HTTP_ADDR=:8080
  DATABASE_DSN=postgres://user:pass@localhost:5432/learn_go?sslmode=disable
  JWT_SECRET=change-me
  JWT_EXPIRES_IN=24h
  REFRESH_TOKEN_EXPIRES_IN=720h
  OSS_PROVIDER=local
  ```

3. **初始化数据库**：首次启动会自动执行 `AutoMigrate`。在生产环境建议手动备份，并根据需要预置院系 / 班级 / 账号数据。
4. **运行服务**：

  ```bash
  go run ./cmd/server
  ```

5. **运行测试/格式化**：

  ```bash
  gofmt -w ./internal ./pkg
  go test ./...
  ```

> Windows PowerShell 可使用 `;` 连接命令，例如 `cd f:/Projects/Go/learn-go; go test ./...`。

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
| `POST` | `/api/v1/admin/accounts/batch` | 批量锁定/解锁/要求重置密码/删除账号。|
| `GET` | `/api/v1/admin/departments` | 列出院系列表。|
| `GET` | `/api/v1/admin/departments/:id/classes` | 查看指定院系下的班级。|

#### 请求字段说明

- 创建教师：`school_id`, `number`, `name`, `email`, `phone`, `default_password`
- 创建学生：`school_id`, `number`, `name`, `email`, `phone`, `class_id`, `teacher_ids[]`, `default_password`
- 创建院系：`school_id`, `name`
- 创建班级：`school_id`, `department_id`, `name`

#### 批量账号操作

支持的 `action`：

- `lock`：将账号状态改为锁定；
- `unlock`：将账号状态改回 `active`；
- `reset_password`：要求账号下次登录前重置密码；
- `delete`：删除账号（仅影响老师/学生账号）。

请求体：

```json
{
  "school_id": "school-1",
  "action": "lock",
  "account_ids": [
    "acc-teacher-1",
    "acc-student-9"
  ]
}
```

响应示例：

```json
{
  "success": true,
  "data": {
    "succeeded": ["acc-teacher-1", "acc-student-9"],
    "failed": {}
  }
}
```

若部分账号失败，`failed` 字段会以 `{"account_id": "原因"}` 的形式列出原因（例如账号不属于当前学校、已处于目标状态等）。

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
| `GET` | `/api/v1/teacher/assignments` | 教师门户作业列表，可通过 `types=homework,exam`、`class_id`、`limit` 等参数筛选。|
| `GET` | `/api/v1/teacher/assignments/:id` | 教师端专用详情；默认隐藏题目答案，可通过 `include_answers=true` 展示完整题目、班级人数、未提交统计及评分分布。|
| `GET` | `/api/v1/teacher/exams` | 仅返回考试类作业（`type=exam`），方便教师快速查看考试进度与统计。|
| `GET` | `/api/v1/teacher/assignments/:id/export` | 导出指定作业/考试的学生成绩 CSV，包含学号、姓名、状态、得分与提交时间。|

##### 作业看板指标

教师门户作业列表现已返回更丰富的统计字段，便于快速评估班级完成度：

| 字段 | 说明 |
| --- | --- |
| `submission_count` | 已存在的提交总数（含草稿/已提交/已批改）。|
| `submitted_count` | 状态为 `submitted/graded` 的人数。|
| `graded_count` | 完成批改的人数。|
| `pending_grade_count` | `submitted_count - graded_count`，表示待批改量。|
| `missing_count` | 班级学生总数 - `submitted_count`，直观反映“未提交”人数。|
| `score_average` / `score_max` / `score_min` | 评分分布，自动统计已批改作业的平均/最高/最低分。|
| `latest_submission_at` | 最近一次提交时间，便于追踪截止前后活跃度。|
| `score_distribution` | 更细粒度的成绩区间统计，包含 `below_60`、`between_60_70`、`between_70_80`、`between_80_90`、`above_90` 五个桶，可直接绘制柱状图或饼图。|

所有字段可通过教师侧的 assignments 列表接口一次性获取，前端可直接渲染进度条、分数段统计或提醒“未交”学生。

**成绩导出**：教师可访问 `/api/v1/teacher/assignments/:id/export` 直接下载 CSV，文件包含 `student_id、student_number、student_name、status、score、submitted_at` 列，可用于成绩上报或进一步分析。

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

### 学生提醒 API

自定义提醒用于补充“作业/考试/课表”之外的个人待办。所有接口均需学生角色访问。

| 方法 | 路径 | 描述 |
| --- | --- | --- |
| `GET` | `/api/v1/student/reminders` | 列出本人自定义提醒，按创建时间倒序。|
| `POST` | `/api/v1/student/reminders` | 新建提醒（标题必填，可选描述、时间、图标、跳转路由、优先级）。|
| `PATCH` | `/api/v1/student/reminders/:id` | 更新提醒任意字段，含 `completed` 状态。|
| `DELETE` | `/api/v1/student/reminders/:id` | 删除提醒。|
| `POST` | `/api/v1/student/reminders/:id/completion` | 仅更新单条提醒的完成状态。|
| `POST` | `/api/v1/student/reminders/completion/batch` | 批量设置多条提醒的完成/未完成状态。|
| `POST` | `/api/v1/student/reminders/completion/all` | 一键将所有提醒标记为完成或重新激活。兼容旧路径 `/reminders/complete_all`。|

#### 新建提醒请求示例

```json
{
  "title": "准备英语演讲",
  "description": "整理 PPT，练习 3 遍",
  "time_label": "周五 18:00 前",
  "priority": "high",
  "icon": "assignment",
  "route": "/student/notes"
}
```

#### 单条完成状态请求

> `POST /api/v1/student/reminders/:id/completion`

```json
{
  "completed": true
}
```

- `completed` 省略时默认为 `true`，用于“标记完成”；
- 传 `false` 可重新激活提醒；
- 成功时返回更新后的提醒对象，字段 `is_completed`、`completed_at` 会同步更新。

#### 批量完成/撤销请求

```json
{
  "reminder_ids": ["rem-101", "rem-205"],
  "completed": false
}
```

当部分 ID 不存在或不属于当前学生时，接口将返回 `404` 并提示 `reminders not found`。全部成功则返回 `204 No Content`。

#### 一键完成

```json
{
  "completed": true
}
```

若请求体为空（或仅包含 `completed`），接口同样有效。前端可在“全部完成”按钮中直接调用。

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
