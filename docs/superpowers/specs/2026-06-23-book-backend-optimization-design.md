# Book Backend Optimization — Design Spec

**Date:** 2026-06-23  
**Status:** Approved  
**Context:** Frontend book module (`/home/zqw/Downloads/win_share/ysx/src/modules/pages/book`) requires backend enhancements for production. The existing backend has a basic `book` table with single-image, no-pagination, no-search CRUD. This spec covers the changes needed to support the full frontend feature set.

**Decisions made during brainstorming:**
- Student-facing `/api/v1/books` APIs only (admin `/api/books` stays as-is)
- Simple want toggle (no want-list page, no notifications)
- `school_id` column for multi-school pattern consistency
- Separate `book_images` table for multi-image support
- Database-driven `book_categories` table (admin-manageable)

---

## 1. Database Changes

### 1.1 `book` table — new columns

Add via ALTER TABLE (migration checks column existence first, following existing pattern):

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `description` | TEXT | NULL | Book description (max 500 chars enforced by frontend) |
| `condition` | VARCHAR(20) | '几乎全新' | 新旧程度: 全新, 几乎全新, 有笔记, 较旧 |
| `school_id` | VARCHAR(50) | 'hbut' | School code, consistent with shops/foods/clubs/users |

### 1.2 New table: `book_images`

```sql
CREATE TABLE IF NOT EXISTS book_images (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  book_id INT UNSIGNED NOT NULL,
  image_url VARCHAR(500) NOT NULL,
  sort_order TINYINT UNSIGNED DEFAULT 0,
  FOREIGN KEY (book_id) REFERENCES book(book_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- CASCADE delete ensures images are cleaned up when a book is deleted.
- Max 3 images enforced in application layer (models + handlers validate count).
- The `image_url` column on `book` is kept for backward compatibility; it stores the first image URL for list-card display.

### 1.3 New table: `book_categories`

```sql
CREATE TABLE IF NOT EXISTS book_categories (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  school_id VARCHAR(50) NOT NULL DEFAULT 'hbut',
  sort_order INT DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- Migration seeds existing hard-coded categories: 数学, 外语, 计算机, 理工类, 思政类, 文学类, 经管类, 其他.
- Admin CRUD for categories is added to `/api/books/categories`.

### 1.4 New table: `book_wants`

```sql
CREATE TABLE IF NOT EXISTS book_wants (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  book_id INT UNSIGNED NOT NULL,
  user_id INT UNSIGNED NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_book_user (book_id, user_id),
  FOREIGN KEY (book_id) REFERENCES book(book_id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- UNIQUE constraint enforces one-want-per-user-per-book.
- CASCADE deletes clean up when book or user is deleted.

---

## 2. API Endpoints

All endpoints under `/api/v1`, authenticated via `middleware.StudentAuth`.

### 2.1 Modified endpoints

#### `GET /api/v1/books/categories`
- **Change:** Query `book_categories` table (filtered by `school_id`) instead of returning hard-coded list.
- **Response:** `{ success: true, data: ["全部", "数学", "外语", ...] }` — "全部" prepended in handler for frontend filter bar.

#### `GET /api/v1/books`
- **New query params:**
  - `keyword` (string, optional) — search in `title` and `isbn` (LIKE %keyword%)
  - `page` (int, default 1)
  - `pageSize` (int, default 20)
  - `category` (string, optional) — existing, unchanged
  - `school_id` (string, default from context) — filter by school
- **Response:** `{ success: true, data: [...], total: 56 }`
- List items include `image_url` (first image only, for thumbnail display) but NOT the full `images` array.

#### `GET /api/v1/books/mine`
- Add same pagination params as `/books`.
- Filter by current user.

#### `GET /api/v1/books/:id`
- **Response now includes:**
  - `images: [{ id, url, sort_order }]` — from `book_images` table
  - `want_count: int` — count from `book_wants`
  - `is_wanted: bool` — whether current user has wanted this book
  - `condition: string`
  - `description: string`

#### `POST /api/v1/books`
- **New accepted fields:** `description`, `condition`, `image_urls` (comma-separated URL string, e.g. `"/uploads/a.jpg,/uploads/b.jpg"`)
- **Flow:** Create `book` row → insert `book_images` rows from `image_urls` → set `image_url` to first image.
- Validation: max 3 images, max 5 active books per user (existing limit kept).

#### `PUT /api/v1/books/:id`
- Accept same new fields. Ownership check unchanged.
- **Image sync:** On update, delete all existing `book_images` rows for this book, then insert new ones from submitted `image_urls`.
- The `delete_image` form field from old single-image flow is deprecated for V1 (but kept working for admin APIs).

### 2.2 New endpoints

#### `POST /api/v1/books/upload-image`
- Accepts `multipart/form-data` with a single `file` field.
- Validates: allowed extensions (.jpg, .jpeg, .png, .webp), max 5MB.
- Saves to `UPLOAD_DIR` with UUID filename.
- **Returns:** `{ success: true, data: { url: "/uploads/uuid.jpg" } }`
- Frontend calls this immediately after `Taro.chooseImage`, before form submit.

#### `DELETE /api/v1/books/images/:imageId`
- Deletes the `book_images` row by ID AND the file from disk.
- Ownership verified through book → user_id chain.
- **Returns:** `{ success: true, message: "删除成功" }`

#### `POST /api/v1/books/:id/want`
- **Toggle logic:**
  - If `book_wants` row exists for (book_id, user_id) → DELETE it, return `{ wanted: false, want_count: N }`
  - If not exists → INSERT it, return `{ wanted: true, want_count: N }`
- Uses `GetStudentUserID(c)` from middleware for user identity.

### 2.3 Admin endpoints (minor addition)

#### `GET /api/book-categories`
- Already exists as `/api/books/categories` but updated to query DB.

#### `POST /api/book-categories` (new)
- Admin creates a category with `name`, `school_id`, `sort_order`.

#### `PUT /api/book-categories/:id` (new)
- Admin updates a category.

#### `DELETE /api/book-categories/:id` (new)
- Admin deletes a category.

---

## 3. Implementation

### 3.1 Files to change

| File | Changes |
|------|---------|
| `cmd/migrate/main.go` | Add migration SQL for `book_images`, `book_categories`, `book_wants` tables; ALTER `book` table for new columns; seed default categories |
| `internal/models/book.go` | Add structs (`BookImage`, `BookCategory`, `BookWant`); add functions for paginated+search list, image CRUD, want toggle, category DB queries; add `GetBookDetailWithImages` |
| `internal/handlers/v1_book.go` | Update existing handlers; add `V1UploadBookImage`, `V1DeleteBookImage`, `V1ToggleWant` |
| `internal/handlers/book.go` | Update `GetBookCategories` to query DB; add admin CRUD for categories |
| `cmd/server/main.go` | Register new V1 routes (`/books/upload-image`, `/books/images/:imageId`, `/books/:id/want`) |

### 3.2 Patterns to follow

- **Closure factories:** Handlers return `gin.HandlerFunc`, capturing `*config.Config` when needed.
- **Model signatures:** All data functions take `*sqlx.DB` first, use raw SQL with sqlx.
- **Error handling:** Use `dto.Success`, `dto.Error`, `dto.BadRequest`, `dto.InternalError`, `dto.Forbidden` helpers.
- **Auth:** `middleware.GetStudentUserID(c)` for student identity.
- **Image upload:** Reuse existing `saveUploadedImage` helper in `v1_book.go`.

### 3.3 Key SQL queries

**Paginated list with search:**
```sql
SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId
FROM book b
LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
WHERE b.status = 'active'
  AND b.school_id = ?
  AND (? = '' OR b.category = ?)
  AND (? = '' OR b.title LIKE CONCAT('%', ?, '%') OR b.isbn LIKE CONCAT('%', ?, '%'))
ORDER BY b.book_id DESC
LIMIT ? OFFSET ?
```

**Detail with images, want info:**
```sql
-- Main query (book + user)
SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId
FROM book b LEFT JOIN users u ON b.user_id = u.id
WHERE b.book_id = ?

-- Images (separate query, then merged in Go)
SELECT id, image_url, sort_order FROM book_images WHERE book_id = ? ORDER BY sort_order

-- Want count
SELECT COUNT(*) FROM book_wants WHERE book_id = ?

-- Is wanted by current user
SELECT COUNT(*) > 0 FROM book_wants WHERE book_id = ? AND user_id = ?
```

### 3.4 Image upload flow

```
Frontend                           Backend
  │                                  │
  │ Taro.chooseImage()               │
  │ Taro.uploadFile() ──────────────>│ POST /api/v1/books/upload-image
  │                                  │   → validates ext/size
  │                                  │   → saves to /uploads/uuid.ext
  │ <─ { url: "/uploads/uuid.jpg" }  │   → returns URL (no DB row created)
  │                                  │
  │ (store URL in local state[])     │
  │                                  │
  │ Form submit ────────────────────>│ POST /api/v1/books
  │   { image_urls: "url1,url2" }    │   → INSERT book row
  │                                  │   → INSERT book_images rows (one per URL)
  │                                  │   → UPDATE book.image_url = first URL
```

Image deletion (frontend taps × on an image in the edit form):
```
Frontend                           Backend
  │ Taro.request() ────────────────>│ DELETE /api/v1/books/images/:imageId
  │                                  │   → verify ownership (book.user_id)
  │                                  │   → delete file from disk
  │                                  │   → DELETE book_images row
  │ <─ { success: true }             │
  │ (remove URL from local state[])  │
```

---

## 4. Error Handling & Edge Cases

- **Upload before submit:** If user uploads images then abandons the form, orphaned files remain on disk. This is acceptable for now (files are small, disk is cheap). Could add a cleanup cron later.
- **Image count validation:** Both upload and create/update enforce max 3 images per book. Handler returns 400 if exceeded.
- **Want toggle race:** The UNIQUE key on `(book_id, user_id)` prevents duplicate wants at DB level. The handler attempts INSERT first; if duplicate key error, it deletes instead.
- **Soft delete:** Books set to `status='deleted'` are excluded from all list queries (WHERE status='active'). Related `book_images` and `book_wants` rows remain (not cascaded) for data integrity.
- **Ownership check:** Image delete and book update/delete verify `book.user_id == current_user_id` before proceeding.

---

## 5. Seed Data (Migration)

The migration inserts these default categories for school `hbut`:

```
数学 (sort 1), 外语 (sort 2), 计算机 (sort 3), 理工类 (sort 4),
思政类 (sort 5), 文学类 (sort 6), 经管类 (sort 7), 其他 (sort 8)
```

---

## 6. What's NOT in Scope

- Admin `/api/books` endpoints are left unchanged (single image, no pagination). Admin can view/delete but not get the new fields in their existing UI.
- No want-list page for users (simple toggle only).
- No notifications on want events.
- No cleanup job for orphaned uploaded images.
- No image CDN/object storage migration (local disk storage kept).
