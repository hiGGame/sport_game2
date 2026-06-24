# 竞彩赛事竞猜系统 (sport_game2)

从竞彩网爬取足球/篮球赛事数据，为微信小程序提供赛事信息、竞猜、开奖结果及三角色预测（AI狗 + 资深大拿 + 用户竞猜）。

## 技术栈

- Go 1.26 + Gin + PostgreSQL 16
- 爬虫：内置 HTTP 客户端（带 WAF 绕过 header）
- 部署：Docker Compose（postgres + spider + server）

## 项目结构

```
cmd/spider/          爬虫入口（定时爬取赔率+开奖结果）
cmd/server/          后端 API 服务入口
internal/
  adapter/apifox/    Apifox schema 对齐的数据模型（14 个类型）
  crawler/sporttery/  竞彩官网爬虫（足球5玩法+篮球4玩法+开奖）
  config/            配置加载（YAML + 环境变量）
  handler/v1/        Gin HTTP handler
  middleware/         JWT 鉴权 + CORS
  repo/              数据访问层
  service/
    auth/            微信授权登录（code→openid→JWT）
    match/           赛事列表/详情/热门
    bet/             用户竞猜（积分扣减+锁定）
    result/          开奖结果+三角色预测聚合+结算
    predictor/       AI 预测 Provider 接口+规则占位实现
pkg/
  jwt/              JWT 签发与校验
  wechat/           微信 code2session 客户端
  credits/          积分扣减/退还（事务封装）
  sporttery/        玩法常量定义
migrations/         SQL 迁移脚本
scripts/            种子数据
configs/            配置文件
```

## 快速开始

### 本地运行

```bash
# 1. 启动 PostgreSQL
pg_isready || brew services start postgresql

# 2. 创建数据库
psql -U $(whoami) -d postgres -c "CREATE DATABASE sport_game2;"
psql -U $(whoami) -d sport_game2 -f migrations/001_init.sql
psql -U $(whoami) -d sport_game2 -f scripts/seed_experts.sql

# 3. 运行爬虫（单次）
DB_USER=$(whoami) DB_PASSWORD="" go run ./cmd/spider --once

# 4. 启动后端服务
DB_USER=$(whoami) DB_PASSWORD="" go run ./cmd/server
```

### Docker Compose 部署

```bash
cp configs/.env.example configs/.env  # 编辑配置
docker-compose up -d
```

## API 端点

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/v1/customer/login/wechat` | - | 微信授权登录 |
| GET | `/v1/lottery/match/bet-list` | - | 赛事投注列表 |
| GET | `/v1/lottery/match/bet-info` | - | 单场投注信息 |
| GET | `/v1/lottery/match/hot/list` | - | 热门赛事 |
| GET | `/v1/lottery/match/draw/list` | - | 开奖大厅列表 |
| GET | `/v1/matches/:matchId/predict` | - | 三角色预测 |
| GET | `/v1/user/info` | JWT | 用户信息 |
| PUT | `/v1/user/info` | JWT | 更新用户资料 |
| POST | `/v1/bets` | JWT | 创建竞猜（扣积分） |
| GET | `/v1/bets/mine` | JWT | 我的竞猜记录 |
| POST | `/v1/matches/:matchId/settle` | JWT | 结算赛事 |

## 竞彩玩法

### 足球（lotteryType=227）
| subType | 玩法 | poolCode |
|---|---|---|
| 6 | 胜平负 | had |
| 1 | 让球胜平负 | hhad |
| 2 | 总进球 | ttg |
| 3 | 比分 | crs |
| 4 | 半全场 | hafu |

### 篮球（lotteryType=228）
| subType | 玩法 | poolCode |
|---|---|---|
| 1 | 胜负 | mnl |
| 2 | 让分胜负 | hdc |
| 3 | 胜分差 | wsf |
| 4 | 大小分 | hhu |

## 三角色预测

1. **AI狗**：`predictor.Provider` 接口 + `RuleProvider`（基于赔率反推概率），预留 LLM 接入点
2. **资深大拿**：`experts` 表 + `expert_predictions` 表，管理后台录入或硬编码
3. **用户竞猜**：`predictions` 表，按 matchId 聚合

## 爬虫调度

- 赔率刷新：每 5 分钟
- 开奖结果：每 10 分钟
- 失败重试 3 次，记录 `spider_job_log` 表
- Docker `restart: always` 保活
