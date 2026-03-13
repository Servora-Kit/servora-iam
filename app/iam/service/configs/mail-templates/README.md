# 邮件 HTML 模板（可选）

将本目录路径配置到 `bootstrap.yaml` 的 `mail.template_dir` 后，IAM 会优先从此目录加载模板，未配置或文件缺失时使用内嵌默认内容。

## 文件名约定

| 文件 | 用途 | 模板变量 |
|------|------|----------|
| `verify_email.html` | 邮箱验证邮件 | `{{.Link}}` 验证链接，`{{.ExpiryHours}}` 展示用（如 "24"） |
| `reset_password.html` | 密码重置邮件 | `{{.Link}}` 重置链接，`{{.ExpiryHours}}` 展示用（如 "1"） |

使用 Go `html/template` 语法，变量会自动转义，避免 XSS。

## 配置示例

**本地**（在 `app/iam/service` 下运行）：

```yaml
mail:
  template_dir: "./configs/mail-templates"
  # ... 其他 mail 配置
```

**Docker**（配置挂载到 `/app/configs/` 时）：

```yaml
mail:
  template_dir: "/app/configs/mail-templates"
  # ... 其他 mail 配置
```

不设置 `template_dir` 时，仍使用代码内嵌的默认 HTML，无需本目录即可运行。
