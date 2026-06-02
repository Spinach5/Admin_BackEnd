# /api/v1 普通用户路由设计

## 概述

新增 `/api/v1` 路由组，专门给普通学生用户使用。不走 JWT，通过请求 Body 中的身份参数（id、stuId、schoolId）识别用户。

## 认证中间件 V1Auth

- 从请求 Body 提取 `id`、`stuId`、`schoolId`（三者 required）
- 用 `id` 查 `users` 表，校验用户存在且 `isDeleted = 0`
- 通过后将 `user_id`、`stuId`、`schoolId` 注入 `gin.Context`
- 失败返回 401

## 数据库变更

foods、shops、affairs 三表新增 `school_id` 列：

```sql
ALTER TABLE foods ADD COLUMN school_id VARCHAR(50) NOT NULL DEFAULT '' AFTER canteen_name;
ALTER TABLE shops ADD COLUMN school_id VARCHAR(50) NOT NULL DEFAULT '' AFTER canteen_name;
ALTER TABLE affairs ADD COLUMN school_id VARCHAR(50) NOT NULL DEFAULT '' AFTER channel;
```

迁移脚本 `cmd/migrate/main.go` 同步更新建表语句。

## 接口列表

全部 POST，Body 均含 `id`、`stuId`、`schoolId`。

| 路由 | 功能 | 额外 Body 字段 | 说明 |
|---|---|---|---|
| `POST /api/v1/foods` | 食物列表 | 无 | WHERE school_id = ? |
| `POST /api/v1/shops` | 店铺列表 | 无 | WHERE school_id = ? |
| `POST /api/v1/affairs` | 事务列表 | 无 | WHERE school_id = ? |
| `POST /api/v1/books` | 书籍列表 | 无 | 全部书籍 + 发布者昵称/学号 |
| `POST /api/v1/books/add` | 添加书籍 | title, category?, image_url?, price?, isbn?, contact? | user_id 自动取身份参数，最多 5 本活跃 |
| `POST /api/v1/books/delete` | 软删除书籍 | book_id | 只能删自己的，SET status='deleted' |

## 业务规则

- **5 本限制**：添加前 `SELECT COUNT(*) FROM book WHERE user_id=? AND status='active'`，>=5 拒绝
- **软删除**：`UPDATE book SET status='deleted' WHERE book_id=? AND user_id=?`
- **身份校验**：books/add 的 user_id 从身份参数获取，不由前端传入

## 涉及文件

- `cmd/migrate/main.go` — 建表语句加 school_id
- `cmd/server/main.go` — 注册 /api/v1 路由组
- `internal/middleware/v1_auth.go` — 新建 V1Auth 中间件
- `internal/handlers/v1.go` — 新建 v1 handler（foods/shops/affairs/books 列表 + books/add + books/delete）
- `internal/models/food.go` — struct 加 SchoolID，查询加 WHERE school_id
- `internal/models/shop.go` — struct 加 SchoolID，查询加 WHERE school_id
- `internal/models/affair.go` — struct 加 SchoolID，查询加 WHERE school_id
- `internal/models/book.go` — 新增 GetActiveBookCountByUser 函数
- `internal/dto/request.go` — 新增 v1 请求 DTO
