import express, { type Express } from "express";
import helmet from "helmet";
import cors from "cors";
import cookieParser from "cookie-parser";
import { getRedisClient } from "./redis/client.ts";
import { createTokenStore } from "./services/tokenStore.ts";
import { authRoutes } from "./routes/auth.routes.ts";
import { authGuard } from "./middleware/auth.ts";

export async function createApp(): Promise<Express> {
	const app = express();

	app.set("trust proxy", 1);
	app.use(helmet());
	app.use(cors({ origin: true, credentials: true }));
	app.use(express.json());
	app.use(cookieParser());

	const redis = await getRedisClient();
	const store = createTokenStore(redis);

	// 公开路由：登录 / 刷新 / 登出
	app.use("/auth", authRoutes(store));

	// 需要鉴权的路由示例
	app.get("/api/me", authGuard(store), (req, res) => {
		res.json({ userId: req.userId });
	});

	return app;
}
