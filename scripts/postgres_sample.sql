-- PostgreSQL schema & seed data for learn-go project testing
-- Run on a clean database. Adjust schema name or wrap in transaction as needed.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Drop existing tables in dependency order (if they exist)
DROP TABLE IF EXISTS note_comments CASCADE;
DROP TABLE IF EXISTS notes CASCADE;
DROP TABLE IF EXISTS ai_chat_messages CASCADE;
DROP TABLE IF EXISTS ai_chat_sessions CASCADE;
DROP TABLE IF EXISTS ai_agent_setting_audits CASCADE;
DROP TABLE IF EXISTS ai_agent_settings CASCADE;
DROP TABLE IF EXISTS message_receipts CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS conversation_members CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS submission_comments CASCADE;
DROP TABLE IF EXISTS submission_items CASCADE;
DROP TABLE IF EXISTS assignment_submissions CASCADE;
DROP TABLE IF EXISTS assignment_questions CASCADE;
DROP TABLE IF EXISTS assignments CASCADE;
DROP TABLE IF EXISTS course_sessions CASCADE;
DROP TABLE IF EXISTS courses CASCADE;
DROP TABLE IF EXISTS course_slots CASCADE;
DROP TABLE IF EXISTS teacher_student_links CASCADE;
DROP TABLE IF EXISTS students CASCADE;
DROP TABLE IF EXISTS teachers CASCADE;
DROP TABLE IF EXISTS classes CASCADE;
DROP TABLE IF EXISTS departments CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;
DROP TABLE IF EXISTS schools CASCADE;

-- Core reference tables ------------------------------------------------------
CREATE TABLE schools (
    id          CHAR(36) PRIMARY KEY,
    name        VARCHAR(128) UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE accounts (
    id            CHAR(36) PRIMARY KEY,
    school_id     CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    role          VARCHAR(16) NOT NULL,
    identifier    VARCHAR(64) NOT NULL,
    password_hash VARCHAR(128) NOT NULL,
    display_name  VARCHAR(128) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

ALTER TABLE accounts
    ADD CONSTRAINT "uni_accounts_identifier" UNIQUE (identifier);

CREATE INDEX accounts_role_idx ON accounts(role);
CREATE INDEX accounts_school_idx ON accounts(school_id);

CREATE TABLE departments (
    id         CHAR(36) PRIMARY KEY,
    school_id  CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name       VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE classes (
    id            CHAR(36) PRIMARY KEY,
    school_id     CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    department_id CHAR(36) NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    name          VARCHAR(128) NOT NULL,
    homeroom_id   CHAR(36),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE teachers (
    id         CHAR(36) PRIMARY KEY,
    school_id  CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    account_id CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    number     VARCHAR(64) NOT NULL,
    department_id CHAR(36) REFERENCES departments(id) ON DELETE SET NULL,
    email      VARCHAR(128) NOT NULL,
    phone      VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE teachers
    ADD CONSTRAINT "uni_teachers_account_id" UNIQUE (account_id);

ALTER TABLE teachers
    ADD CONSTRAINT "uni_teachers_number" UNIQUE (number);

CREATE TABLE students (
    id         CHAR(36) PRIMARY KEY,
    school_id  CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    account_id CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    number     VARCHAR(64) NOT NULL,
    class_id   CHAR(36) REFERENCES classes(id) ON DELETE SET NULL,
    email      VARCHAR(128) NOT NULL,
    phone      VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE students
    ADD CONSTRAINT "uni_students_account_id" UNIQUE (account_id);

ALTER TABLE students
    ADD CONSTRAINT "uni_students_number" UNIQUE (number);

CREATE TABLE teacher_student_links (
    id         CHAR(36) PRIMARY KEY,
    teacher_id CHAR(36) NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    student_id CHAR(36) NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT "uni_teacher_student_links_teacher_id_student_id" UNIQUE (teacher_id, student_id)
);

-- Course/assignment tables ---------------------------------------------------
CREATE TABLE courses (
    id          CHAR(36) PRIMARY KEY,
    school_id   CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name        VARCHAR(128) NOT NULL,
    description VARCHAR(512),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE time_slots (
    id          CHAR(36) PRIMARY KEY,
    school_id   CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name        VARCHAR(64) NOT NULL,
    start_time  VARCHAR(5) NOT NULL,
    end_time    VARCHAR(5) NOT NULL,
    sort_order  INT DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE course_schedules (
    id          CHAR(36) PRIMARY KEY,
    school_id   CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    course_id   CHAR(36) NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    class_id    CHAR(36) NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    teacher_id  CHAR(36) NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    slot_id     CHAR(36) NOT NULL REFERENCES time_slots(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL,
    location    VARCHAR(128),
    start_date  TIMESTAMPTZ NOT NULL,
    end_date    TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE course_sessions (
    id          CHAR(36) PRIMARY KEY,
    course_id   CHAR(36) NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    class_id    CHAR(36) NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    teacher_id  CHAR(36) NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    slot_id     CHAR(36) REFERENCES time_slots(id) ON DELETE SET NULL,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    location    VARCHAR(128),
    source      VARCHAR(32),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE teaching_assignments (
    id          CHAR(36) PRIMARY KEY,
    school_id   CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    course_id   CHAR(36) NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    teacher_id  CHAR(36) NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    class_id    CHAR(36) NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT "uni_teaching_assignments" UNIQUE (course_id, teacher_id, class_id)
);

CREATE TABLE assignments (
    id             CHAR(36) PRIMARY KEY,
    course_id      CHAR(36) NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    teacher_id     CHAR(36) NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    class_id       CHAR(36) NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    type           VARCHAR(16) NOT NULL,
    title          VARCHAR(256) NOT NULL,
    description    VARCHAR(1024),
    start_at       TIMESTAMPTZ,
    due_at         TIMESTAMPTZ,
    max_score      NUMERIC(6,2) NOT NULL,
    allow_resubmit BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE assignment_questions (
    id            CHAR(36) PRIMARY KEY,
    assignment_id CHAR(36) NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    type          VARCHAR(16) NOT NULL,
    prompt        TEXT NOT NULL,
    options       TEXT,
    answer        TEXT,
    score         NUMERIC(6,2) NOT NULL,
    order_index   INT NOT NULL
);

CREATE TABLE assignment_submissions (
    id            CHAR(36) PRIMARY KEY,
    assignment_id CHAR(36) NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    student_id    CHAR(36) NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    submitted_at  TIMESTAMPTZ,
    score         NUMERIC(6,2),
    feedback      TEXT,
    status        VARCHAR(32) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX assignment_submissions_unique ON assignment_submissions(assignment_id, student_id);

CREATE TABLE submission_items (
    id            CHAR(36) PRIMARY KEY,
    submission_id CHAR(36) NOT NULL REFERENCES assignment_submissions(id) ON DELETE CASCADE,
    question_id   CHAR(36) NOT NULL REFERENCES assignment_questions(id) ON DELETE CASCADE,
    answer        TEXT,
    score         NUMERIC(6,2)
);

CREATE TABLE submission_comments (
    id            CHAR(36) PRIMARY KEY,
    submission_id CHAR(36) NOT NULL REFERENCES assignment_submissions(id) ON DELETE CASCADE,
    author_id     CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    author_role   VARCHAR(16) NOT NULL,
    content       TEXT NOT NULL,
    attachment_uri VARCHAR(256),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Conversation/messaging tables ---------------------------------------------
CREATE TABLE conversations (
    id         CHAR(36) PRIMARY KEY,
    type       VARCHAR(16) NOT NULL,
    school_id  CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_members (
    id              CHAR(36) PRIMARY KEY,
    conversation_id CHAR(36) NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    account_id      CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    role            VARCHAR(16) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (conversation_id, account_id)
);

CREATE TABLE messages (
    id              CHAR(36) PRIMARY KEY,
    conversation_id CHAR(36) NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    sender_role     VARCHAR(16) NOT NULL,
    kind            VARCHAR(16) NOT NULL,
    text            TEXT,
    media_uri       VARCHAR(256),
    metadata        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE message_receipts (
    id         CHAR(36) PRIMARY KEY,
    message_id CHAR(36) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    account_id CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (message_id, account_id)
);

-- Notes ---------------------------------------------------------------------
CREATE TABLE notes (
    id         CHAR(36) PRIMARY KEY,
    school_id  CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    owner_id   CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    owner_role VARCHAR(16) NOT NULL,
    title      VARCHAR(256) NOT NULL,
    content    TEXT NOT NULL,
    visibility VARCHAR(16) NOT NULL,
    status     VARCHAR(16) NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE note_comments (
    id         CHAR(36) PRIMARY KEY,
    note_id    CHAR(36) NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    author_id  CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    author_role VARCHAR(16) NOT NULL,
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed data -----------------------------------------------------------------
INSERT INTO schools (id, name) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Horizon International School');

INSERT INTO accounts (id, school_id, role, identifier, password_hash, display_name)
VALUES
    -- admin001 / Admin@123
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'admin',   'admin001',   '$2a$10$r55hZB6LyJ6858Y7V31ReOp.E7zQ/uVnmT0rok7UG3WiH6iH4mNbW', '校区管理员'),
    -- tch-1001 / Teacher@123
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111', 'teacher', 'tch-1001',   '$2a$10$HsCf6BKbYMzXQR.VXtE9Z.9hry4fFdiPotuLxW9o./KCfB2kOvnDm', '李老师'),
    -- stu-2025001 / Student@123
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111', 'student', 'stu-2025001', '$2a$10$8/0qgqABMO7pYcPxTVGHX.S6ezGYUFuIUgzvQVNMDAxHcYexgBhz2', '张三');

INSERT INTO departments (id, school_id, name) VALUES
    ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', '信息工程系');

INSERT INTO classes (id, school_id, department_id, name, homeroom_id)
VALUES 
    ('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', '信工 2025 级 1 班', NULL),
    ('33333333-3333-3333-3333-333333333334', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', '信工 2025 级 2 班', NULL);

INSERT INTO teachers (id, school_id, account_id, number, email, phone)
VALUES ('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'tch-1001', 'teacher@example.com', '13800001111');

INSERT INTO students (id, school_id, account_id, number, class_id, email, phone)
VALUES ('55555555-5555-5555-5555-555555555555', '11111111-1111-1111-1111-111111111111', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 'stu-2025001', '33333333-3333-3333-3333-333333333333', 'student@example.com', '13900002222');

INSERT INTO teacher_student_links (id, teacher_id, student_id)
VALUES ('66666666-6666-6666-6666-666666666666', '44444444-4444-4444-4444-444444444444', '55555555-5555-5555-5555-555555555555');

INSERT INTO courses (id, school_id, name, description)
VALUES 
    ('77777777-7777-7777-7777-777777777777', '11111111-1111-1111-1111-111111111111', '计算机网络', '大二核心课程'),
    ('77777777-7777-7777-7777-777777777778', '11111111-1111-1111-1111-111111111111', '操作系统', '深入理解计算机系统核心'),
    ('77777777-7777-7777-7777-777777777779', '11111111-1111-1111-1111-111111111111', '数据结构与算法', '编程基础必修课');

INSERT INTO time_slots (id, school_id, name, start_time, end_time)
VALUES
    ('slot-001', '11111111-1111-1111-1111-111111111111', '第1-2节', '08:00', '09:40'),
    ('slot-002', '11111111-1111-1111-1111-111111111111', '第3-4节', '10:00', '11:40');

INSERT INTO course_schedules (id, school_id, course_id, class_id, teacher_id, slot_id, day_of_week, location, start_date, end_date)
VALUES
    ('sched-001', '11111111-1111-1111-1111-111111111111', '77777777-7777-7777-7777-777777777777', '33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'slot-001', 1, 'A101', '2025-09-01 00:00:00+00', '2026-01-31 00:00:00+00'),
    ('sched-002', '11111111-1111-1111-1111-111111111111', '77777777-7777-7777-7777-777777777778', '33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'slot-002', 2, 'B202', '2025-09-01 00:00:00+00', '2026-01-31 00:00:00+00');

INSERT INTO course_sessions (id, course_id, class_id, teacher_id, slot_id, starts_at, ends_at, location, source)
VALUES
    ('sess-001', '77777777-7777-7777-7777-777777777777', '33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'slot-001', '2025-09-16 08:00:00+00', '2025-09-16 09:40:00+00', 'A101', 'system'),
    ('sess-002', '77777777-7777-7777-7777-777777777778', '33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'slot-002', '2025-09-16 10:00:00+00', '2025-09-16 11:40:00+00', 'B202', 'system'),
    ('sess-003', '77777777-7777-7777-7777-777777777777', '33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'slot-001', '2025-12-15 08:00:00+00', '2025-12-15 09:40:00+00', 'A101', 'system'),
    ('sess-004', '77777777-7777-7777-7777-777777777778', '33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'slot-002', '2025-12-16 10:00:00+00', '2025-12-16 11:40:00+00', 'B202', 'system');

INSERT INTO teaching_assignments (id, school_id, course_id, teacher_id, class_id)
VALUES
    ('a0000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '77777777-7777-7777-7777-777777777777', '44444444-4444-4444-4444-444444444444', '33333333-3333-3333-3333-333333333333'),
    ('a0000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', '77777777-7777-7777-7777-777777777778', '44444444-4444-4444-4444-444444444444', '33333333-3333-3333-3333-333333333333'),
    ('a0000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', '77777777-7777-7777-7777-777777777779', '44444444-4444-4444-4444-444444444444', '33333333-3333-3333-3333-333333333333');

INSERT INTO assignments (id, course_id, teacher_id, class_id, type, title, description, start_at, due_at, max_score, allow_resubmit)
VALUES (
    '88888888-8888-8888-8888-888888888888',
    '77777777-7777-7777-7777-777777777777',
    '44444444-4444-4444-4444-444444444444',
    '33333333-3333-3333-3333-333333333333',
    'homework',
    '第 1 章作业',
    '阅读教材第 1 章并回答问题',
    NOW() - INTERVAL '7 days',
    NOW() + INTERVAL '3 days',
    100,
    TRUE
);

INSERT INTO assignment_questions (id, assignment_id, type, prompt, options, answer, score, order_index)
VALUES
    ('99999999-9999-9999-9999-999999999999', '88888888-8888-8888-8888-888888888888', 'essay', '解释 OSI 七层模型的每一层职责。', NULL, '参考教材示例答案', 40, 1),
    ('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee', '88888888-8888-8888-8888-888888888888', 'choice', '第 3 层是以下哪一项？', '{"options":["会话层","网络层","物理层"]}', '网络层', 60, 2);

INSERT INTO assignment_submissions (id, assignment_id, student_id, submitted_at, score, feedback, status)
VALUES ('bbbbbbbb-cccc-dddd-eeee-ffffffffffff', '88888888-8888-8888-8888-888888888888', '55555555-5555-5555-5555-555555555555', NOW() - INTERVAL '2 days', 88.5, '整体表现良好，复习第 3 题', 'graded');

INSERT INTO submission_items (id, submission_id, question_id, answer, score)
VALUES
    ('cccccccc-dddd-eeee-ffff-000000000000', 'bbbbbbbb-cccc-dddd-eeee-ffffffffffff', '99999999-9999-9999-9999-999999999999', '从应用层到物理层逐一说明', 38),
    ('dddddddd-eeee-ffff-0000-111111111111', 'bbbbbbbb-cccc-dddd-eeee-ffffffffffff', 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee', '选择：网络层', 50.5);

INSERT INTO submission_comments (id, submission_id, author_id, author_role, content)
VALUES ('eeeeeeee-ffff-0000-1111-222222222222', 'bbbbbbbb-cccc-dddd-eeee-ffffffffffff', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'teacher', '请再次阅读教材第 2 章，加深理解。');

INSERT INTO conversations (id, type, school_id)
VALUES ('ffffffff-0000-1111-2222-333333333333', 'direct', '11111111-1111-1111-1111-111111111111');

INSERT INTO conversation_members (id, conversation_id, account_id, role)
VALUES
    ('00000000-1111-2222-3333-444444444444', 'ffffffff-0000-1111-2222-333333333333', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'teacher'),
    ('11111111-2222-3333-4444-555555555555', 'ffffffff-0000-1111-2222-333333333333', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 'student');

INSERT INTO messages (id, conversation_id, sender_id, sender_role, kind, text)
VALUES ('22222222-3333-4444-5555-666666666666', 'ffffffff-0000-1111-2222-333333333333', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'teacher', 'text', '记得周五前提交实验报告。');

INSERT INTO message_receipts (id, message_id, account_id, read_at)
VALUES ('33333333-4444-5555-6666-777777777777', '22222222-3333-4444-5555-666666666666', 'cccccccc-cccc-cccc-cccc-cccccccccccc', NOW());

INSERT INTO notes (id, school_id, owner_id, owner_role, title, content, visibility, status)
VALUES ('44444444-5555-6666-7777-888888888888', '11111111-1111-1111-1111-111111111111', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 'student', '网络实验心得', '记录一次路由实验的心得体会。', 'class', 'published');

INSERT INTO note_comments (id, note_id, author_id, author_role, content)
VALUES ('55555555-6666-7777-8888-999999999999', '44444444-5555-6666-7777-888888888888', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'teacher', '很好，补充下实验截图会更完整。');

-- AI Assistant tables -------------------------------------------------------
CREATE TABLE ai_agent_settings (
    id                        CHAR(36) PRIMARY KEY,
    school_id                 CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    provider                  VARCHAR(32),
    model                     VARCHAR(128),
    api_key                   VARCHAR(256),
    base_url                  VARCHAR(256),
    temperature               REAL,
    top_p                     REAL,
    max_output_tokens         INT DEFAULT 0,
    max_daily_requests        INT DEFAULT 0,
    max_concurrent_requests   INT DEFAULT 0,
    max_conversation_messages INT DEFAULT 0,
    system_prompt             TEXT,
    vision_enabled            BOOLEAN DEFAULT FALSE,
    updated_by                CHAR(36),
    updated_by_name           VARCHAR(128),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ai_agent_setting_audits (
    id            CHAR(36) PRIMARY KEY,
    school_id     CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    operator_id   CHAR(36),
    operator_name VARCHAR(128),
    action        VARCHAR(64),
    detail        TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ai_chat_sessions (
    id              CHAR(36) PRIMARY KEY,
    school_id       CHAR(36) NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    account_id      CHAR(36) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    role            VARCHAR(16),
    title           VARCHAR(256),
    last_message_at TIMESTAMPTZ,
    message_count   INT DEFAULT 0,
    token_count     INT DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at       TIMESTAMPTZ
);

CREATE INDEX ai_chat_sessions_last_message_at_idx ON ai_chat_sessions(last_message_at);

CREATE TABLE ai_chat_messages (
    id            CHAR(36) PRIMARY KEY,
    session_id    CHAR(36) NOT NULL REFERENCES ai_chat_sessions(id) ON DELETE CASCADE,
    sender        VARCHAR(16),
    content       TEXT,
    prompt_tokens INT DEFAULT 0,
    result_tokens INT DEFAULT 0,
    latency_ms    INT DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ai_chat_messages_created_at_idx ON ai_chat_messages(created_at);
