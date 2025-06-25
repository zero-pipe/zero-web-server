# 提交前检查清单

## zero-web-kit

```bash
cp configs/config.example.yaml configs/config.yaml   # 本地配置不入库
make tidy
make test
make build
```

确认未 `git add`：`logs/`、`config/jwk.json`、`configs/config.yaml`。

## ZMS（`../zms`）

```bash
cmake --build build --config Release   # 或 Linux: cmake --build build -j
```

确认未 `git add`：`conf/*.bak`、`conf/config.local.ini`、`config.ini`、`zms_media_server.log`、`www/`、`build/`。

**`conf/` 应提交**：`config.ini.example`、`config.server.ini.example`、`config.embedded.ini.example`、`config.zero-web-kit.ini`。

日志规范：`log_level=info` 为默认；调试期临时 `debug`，提交前勿把生产 ini 改成长期 `debug`。

## 首次提交建议

```bash
git init   # 已完成
git add -A
git status
git commit -m "chore: initial zero-web-kit layout with mediakit branding and structured logging"
```

## 子模块说明

`3rdpart/onvif-go` 以 vendored 源码形式纳入，不使用 git submodule。若原目录带 `.git`，已移除以避免嵌套仓库。

## 运行依赖

- MySQL + Redis：`make docker-up`
- ZMS：单独编译运行 `demo_media_server --config conf/config.zero-web-kit.ini`
- zero-web-kit `configs/config.yaml` 的 `media` 段与 ZMS HTTP 地址、secret、Hook 一致
