# 一键更新与回退

生产环境不要直接长期运行可变的 `latest` 标签。`newapi-update` 会先拉取
`latest`，确认它对应 GitHub `main` 的最新提交，再将 Compose 固定到不可变的
`sha-xxxxxxx` 镜像。

## 安装

```bash
install -m 755 scripts/newapi-release /usr/local/bin/newapi-release
ln -sfn /usr/local/bin/newapi-release /usr/local/bin/newapi-update
ln -sfn /usr/local/bin/newapi-release /usr/local/bin/newapi-rollback
newapi-release init sha-上一稳定版本
```

默认部署目录为 `/opt/newapi`。可通过 `NEWAPI_APP_DIR` 等环境变量覆盖脚本顶部的配置。

## 更新最新版本

```bash
newapi-update
```

更新流程包括：部署锁、最新提交校验、完整 MySQL 备份、临时数据库迁移预检、
固定 SHA 部署、有超时上限的健康检查、发布状态原子切换和失败自动恢复。
如果容器已经是目标镜像，但 Compose 或发布状态被其他命令改乱，更新命令会固定
Compose 并校准回退状态，不会为此重启正常运行的容器。
指定版本时执行：

```bash
newapi-update sha-abcdefg
```

只检查、不修改生产环境：

```bash
newapi-update --dry-run
```

## 回退上一版本

```bash
newapi-rollback
```

回退只切换应用镜像，不会自动恢复 MySQL，避免覆盖更新后产生的新业务数据。
数据库备份保存在 `/opt/newapi/backups/one-click/`，仅在确认需要时人工恢复。
如果新版本执行了不向后兼容的数据库迁移，旧镜像可能无法直接运行，此时应先
停止写入，再由管理员评估数据库恢复方案，不能盲目覆盖更新后的业务数据。

回退预演和状态检查：

```bash
newapi-rollback --dry-run
newapi-release status
```

所有更新都应通过这两个命令执行。旧的 `newapi-auto-update.timer` 应保持禁用，
不要再手工修改 Compose 为 `latest`，也不要运行旧的自动更新脚本，避免绕过备份、
迁移预检和健康检查直接覆盖线上版本。
