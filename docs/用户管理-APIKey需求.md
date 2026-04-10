# 用户管理 - API Key 功能需求

> 创建日期：2026-04-10
> 状态：待开发

---

## 需求概述

后台管理人员使用平台，每个用户只能有一个 API Key。Key 的生成、查看、重置操作统一在用户列表页面进行。

---

## 功能设计

### 1. 入口
- 用户列表页面（`/admin-users`）每行操作列
- 不再单独做 API Key 列表页面

### 2. 用户列表操作列

| 用户状态 | 操作按钮 |
|---------|---------|
| 未生成 Key | 「生成 Key」 |
| 已生成 Key | 「重置 Key」+「查看 Key」 |

### 3. Key 脱敏规则

**格式：** `sk-abcde*****klmno`

- 前 5 位 + `*****` + 后 5 位
- 接口返回 **必须脱敏**，不可返回明文

### 4. 弹窗交互

| 操作 | 弹窗内容 |
|------|---------|
| 生成 Key | 显示完整 Key（只一次），自动复制到剪贴板 |
| 查看 Key | 显示完整 Key，提供「复制」按钮 |
| 重置 Key | 确认提示 → 生成新 Key → 显示完整 Key（只一次），自动复制 |

---

## 后端接口

### `GET /admin/users`

**改动：** 返回字段增加 `api_key`（脱敏格式）

**响应示例：**
```json
{
  "success": true,
  "data": [{
    "id": "xxx",
    "email": "user@example.com",
    "phone": "13800138000",
    "username": "zhangsan",
    "balance": 100.0000,
    "created_at": "2026-04-10 12:00:00",
    "api_key": "sk-abcde*****fghij"
  }]
}
```

> `api_key` 字段：有 Key 时返回脱敏值，无 Key 时返回空字符串 `""`

---

### `GET /admin/keys/:user_id`

**用途：** 查看指定用户的完整 Key（管理员专用）

**响应示例：**
```json
{
  "success": true,
  "data": {
    "api_key": "sk-abcde-fghij-klmno"
  }
}
```

**错误情况：**
- 用户不存在 → 404
- 用户未生成 Key → 404

---

### `POST /admin/api/key/create`

**用途：** 为用户创建 API Key

**请求：**
```json
{
  "user_id": "xxx"
}
```

**响应：**
```json
{
  "success": true,
  "api_key": "sk-abcde-fghij-klmno",
  "message": "API Key 创建成功，请妥善保管"
}
```

**逻辑：**
- 创建前检查该用户是否已有 Key
- 已有 Key → 返回错误，不允许重复创建
- UNIQUE 校验在应用层完成

---

### `POST /admin/api/key/reset`

**用途：** 重置用户 API Key（生成新 Key，旧 Key 立即失效）

**请求：**
```json
{
  "user_id": "xxx"
}
```

**响应：**
```json
{
  "success": true,
  "api_key": "sk-newke-yzabcd-efghijk",
  "message": "Key 已重置，新 Key 请妥善保管"
}
```

**逻辑：**
- 删除旧 Key 记录
- 生成并保存新 Key
- 返回新 Key 明文（只一次）

---

## 前端页面

### `views/user/index.vue`

每行操作列：
```vue
<el-button link type="primary" @click="openRechargeDialog(row)">充值</el-button>
<el-button link type="warning" @click="openResetDialog(row)">重置</el-button>
<el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
<el-button link type="danger" @click="openDeleteDialog(row)">删除</el-button>
<!-- 新增 -->
<template v-if="!row.api_key">
  <el-button link type="success" @click="openCreateKeyDialog(row)">生成Key</el-button>
</template>
<template v-else>
  <el-button link type="warning" @click="openResetKeyDialog(row)">重置Key</el-button>
  <el-button link type="primary" @click="openViewKeyDialog(row)">查看Key</el-button>
</template>
```

### `views/user/components/UserDialog.vue`

新增两种 mode：
| mode | 用途 |
|------|------|
| `createKey` | 显示新生成的完整 Key + 自动复制 |
| `viewKey` | 显示完整 Key + 复制按钮 |
| `resetKey` | 确认后显示新 Key + 自动复制 |

---

## 实现任务

| # | 任务 | 状态 | 备注 |
|---|------|------|------|
| 1 | 后端：修改 `GET /admin/users` 返回脱敏 api_key | ✅ 完成 | 前5位+****+后5位 |
| 2 | 后端：修改 `POST /admin/api/key/create` 校验逻辑 | ✅ 完成 | 已存在 Key 时报错 |
| 3 | 后端：修改 `POST /admin/api/key/reset` 实现 | ✅ 完成 | 用 user_id 而非 key_id |
| 4 | 前端：用户列表增加 Key 操作按钮 | ✅ 完成 | 有Key显示「重置Key」，无Key显示「生成Key」|
| 5 | 前端：UserDialog 增加 createKey/resetKey 弹窗 | ✅ 完成 | 自动复制、只显示一次 |
| 6 | 联调测试 | ⏳ 待测试 | |

## 重要说明

⚠️ **无「查看 Key」功能**：由于系统存储的是 Key 的 Hash（单向加密），无法还原原始 Key。因此：
- 「查看 Key」功能物理上无法实现
- 用户只能在**生成时**或**重置时**看到原始 Key（各一次）
- 如果用户丢失 Key，只能通过「重置」获取新 Key
