import { Router, type Request, type Response } from "express";
import { env } from "../config/env.ts";

export const authRoutes = Router();

authRoutes.post("/api/login", (req, res) => {
	req.session.userId = "u1";
	res.json({ ok: true });
});

authRoutes.post("/api/logout", (req: Request, res: Response) => {
	// 1. 没有会话：直接清 cookie 并返回
	if (!req.session) {
		res.clearCookie(env.sessionName, {
			httpOnly: true,
			sameSite: "lax",
			secure: env.isProd,
		});
		return res.json({ ok: "true", message: "logged out" });
	}
	req.session.destroy((err) => {
		// 2. 删 Redis 里的 session（connect-redis 自动 DEL sess:<sid>）
		if (err) {
			console.error("[logout] destroy error", err);
			return res.status(500).json({ error: "logout failed" });
		}
		// 3. 清浏览器 cookie —— options 必须与 session cookie 的 path/domain 一致
		res.clearCookie(env.sessionName, {
			httpOnly: true,
			sameSite: "lax",
			secure: env.isProd,
			// path: "/",        // 若 session cookie 显式设了 path，这里也要一致
			// domain: "example.com",
		});
		// 4. 告诉前端丢弃内存里的 csrfToken（下次登录重新取）
		return res.json({ ok: true, message: "logged out" });
	});
});
