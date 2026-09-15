import { Router, type Response } from "express";
import {
	signAccessToken,
	verifyAccessToken,
	generateRefreshToken,
	hashToken,
	newTokenId,
} from "../utils/token.ts";
import type { TokenStore } from "../services/tokenStore.ts";
import { env } from "../config/env.ts";

const COOKIE = "rt";

function setRefreshCookie(res: Response, token: string) {
	res.cookie(COOKIE, token, {
		httpOnly: true,
		secure: env.isProd,
		sameSite: "strict",
		path: "/auth", // 只有 /auth/* 能收到
		maxAge: env.refreshExpiresSec * 1000,
	});
}

export function authRoutes(store: TokenStore) {
	const r = Router();

	// ---------- POST /auth/login ----------
	r.post("/login", async (req, res) => {
		const { username, password } = req.body ?? {};
		// TODO: 替换为真实用户校验
		const user =
			username === "admin" && password === "123456" ? { id: "u_1" } : null;
		if (!user) {
			res.status(401).json({ error: "invalid_credentials" });
			return;
		}

		const refreshToken = generateRefreshToken();
		const familyId = newTokenId(); // 每次登录新建一条 family 链
		await store.saveRefresh(hashToken(refreshToken), {
			userId: user.id,
			familyId,
		});

		const accessToken = signAccessToken({
			userId: user.id,
			jti: newTokenId(),
		});
		setRefreshCookie(res, refreshToken);
		res.json({ accessToken });
	});

	// ---------- POST /auth/refresh ----------
	r.post("/refresh", async (req, res) => {
		const oldToken = req.cookies?.[COOKIE];
		if (!oldToken) {
			res.status(401).json({ error: "no_refresh_token" });
			return;
		}

		const oldHash = hashToken(oldToken);

		// 1) 检查是否是"已轮换出去的旧 token" → 复用检测
		const usedFamilyId = await store.getUsed(oldHash);
		if (usedFamilyId) {
			// ⚠️ 旧 token 再次出现 = 被盗用，立即吊销整条 family 链
			await store.revokeFamily(usedFamilyId);
			res.status(401).json({ error: "reuse_detected" });
			return;
		}

		// 2) 查 active 表
		const rec = await store.getActive(oldHash);
		if (!rec) {
			res.status(401).json({ error: "invalid_refresh_token" });
			return;
		}

		// 3) 轮换：旧标记 used + 删 active；新 token 写入
		const newToken = generateRefreshToken();
		await store.rotate(oldHash, hashToken(newToken), rec);

		const accessToken = signAccessToken({
			userId: rec.userId,
			jti: newTokenId(),
		});
		setRefreshCookie(res, newToken);
		res.json({ accessToken });
	});

	// ---------- POST /auth/logout ----------
	r.post("/logout", async (req, res) => {
		const refreshToken = req.cookies?.[COOKIE];
		if (!refreshToken) {
			res.clearCookie(COOKIE, { path: "/auth" });
			res.json({ ok: true });
			return;
		}

		const hash = hashToken(refreshToken);
		const rec = await store.getActive(hash);

		if (rec) {
			await store.revokeOne(hash, rec.familyId); // 仅登出当前设备

			// 同时拉黑 Access Token（让当前 access 也立即失效）
			const authHeader = req.headers.authorization;
			if (authHeader?.startsWith("Bearer ")) {
				try {
					const payload = verifyAccessToken(authHeader.slice(7));
					const ttl = Math.max(payload.exp - Math.floor(Date.now() / 1000), 1);
					await store.blacklistAccess(payload.jti, ttl);
				} catch {
					/* access 已过期就不管了 */
				}
			}
		}
		res.clearCookie(COOKIE, { path: "/auth" });
		res.json({ ok: true });
	});

	return r;
}
