# 短剧生产工作台 —— 全栈部署快捷命令(Docker Compose)。
# 代码级命令(test/server/cli/build)见 backend/Makefile。
# 一键部署前后端:  make deploy

COMPOSE := docker compose -f docker-compose.deploy.yml

.PHONY: help deploy deploy-build logs ps restart down clean

help:
	@echo "make deploy        构建并后台启动全栈(postgres + 后端 + 前端)"
	@echo "make deploy-build  仅构建镜像,不启动"
	@echo "make logs          跟随查看所有服务日志"
	@echo "make ps            查看服务状态"
	@echo "make restart       重启后端+前端(不重建镜像)"
	@echo "make down          停止并移除容器(保留数据库卷)"
	@echo "make clean         停止并移除容器与数据库卷(清空历史数据)"

deploy:
	$(COMPOSE) up -d --build
	@echo ""
	@echo "✅ 部署完成: 前端 http://localhost:$${FRONTEND_PORT:-3000}  ·  后端 http://localhost:$${BACKEND_PORT:-8080}"
	@echo "   登录默认 admin / admin  ·  查看日志 make logs  ·  停止 make down"

deploy-build:
	$(COMPOSE) build

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

restart:
	$(COMPOSE) restart backend frontend

down:
	$(COMPOSE) down

clean:
	$(COMPOSE) down -v
