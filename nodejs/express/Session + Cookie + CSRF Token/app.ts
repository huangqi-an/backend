import express, { type Express } from "express";
import helmet from "helmet";
import cors from "cors";
import cookieParser from "cookie-parser";
import { buildSessionMiddleware } from "./middleware/session.ts";
import { csrfRouter } from "./routes/csrf.routes.ts";
import { csrfGuard } from "./middleware/csrf.ts";

export async function createApp(): Promise<Express> {
	const app = express();

	app.set("trust proxy", 1); // 生产在反向代理后；开发可设 0/false

	app.use(helmet());
	app.use(
		cors({
			origin: true,
			credentials: true,
		}),
	);
	app.use(express.json());
	app.use(cookieParser());

	// session 必须在依赖 req.session 的中间件之前
	app.use(await buildSessionMiddleware());

	// 取 token：放 guard 之前
	app.use(csrfRouter);

	// 对写操作统一拦截
	app.use(csrfGuard);

	app.post("/api/login", (req, res) => {
		req.session.userId = "u1";
		res.json({ ok: true });
	});

	app.post("/api/danger", (req, res) => {
		res.json({ ok: true, useId: req.session.userId });
	});
	return app;
}
